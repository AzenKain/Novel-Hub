package ebookconv

import (
	"fmt"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

func writeTXT(book *bookparser.BookData) ([]byte, error) {
	var b strings.Builder
	ruler := 0
	if len(book.Metadata.Title) > ruler {
		ruler = len(book.Metadata.Title)
	}
	if len(book.Metadata.Author) > ruler {
		ruler = len(book.Metadata.Author)
	}
	if ruler < 24 {
		ruler = 24
	}
	if book.Metadata.Title != "" {
		b.WriteString(book.Metadata.Title)
		b.WriteString("\n")
	}
	if book.Metadata.Author != "" {
		b.WriteString(book.Metadata.Author)
		b.WriteString("\n")
	}
	if book.Metadata.Title != "" || book.Metadata.Author != "" {
		b.WriteString(strings.Repeat("=", ruler))
		b.WriteString("\n\n")
	}
	for _, ch := range book.Chapters {
		if ch.Title != "" {
			b.WriteString("\n")
			b.WriteString(ch.Title)
			b.WriteString("\n")
			b.WriteString(strings.Repeat("-", ruler))
			b.WriteString("\n\n")
		}
		b.WriteString(fragmentText(ch.Content))
		b.WriteString("\n\n")
	}
	return []byte(b.String()), nil
}

func writeFB2(book *bookparser.BookData, images []Image) ([]byte, error) {
	imgIndex := imageLookup(images)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">` + "\n")

	b.WriteString("<description><title-info>")
	if book.Metadata.Author != "" {
		first, last := splitAuthor(book.Metadata.Author)
		b.WriteString("<author>")
		if first != "" {
			b.WriteString("<first-name>")
			b.WriteString(escapeXML(first))
			b.WriteString("</first-name>")
		}
		if last != "" {
			b.WriteString("<last-name>")
			b.WriteString(escapeXML(last))
			b.WriteString("</last-name>")
		}
		b.WriteString("</author>")
	}
	b.WriteString("<book-title>")
	b.WriteString(escapeXML(fallbackTitle(book)))
	b.WriteString("</book-title>")
	if book.Metadata.Language != "" {
		b.WriteString("<lang>")
		b.WriteString(escapeXML(book.Metadata.Language))
		b.WriteString("</lang>")
	}
	if strings.TrimSpace(book.Metadata.Description) != "" {
		b.WriteString("<annotation><p>")
		b.WriteString(escapeXML(strings.TrimSpace(book.Metadata.Description)))
		b.WriteString("</p></annotation>")
	}
	if len(book.Metadata.CoverData) > 0 {
		b.WriteString(`<coverpage><image l:href="#cover"/></coverpage>`)
	}
	b.WriteString("</title-info>")
	b.WriteString("<document-info><id>")
	b.WriteString(newID())
	b.WriteString("</id><program-used>NovelHub</program-used><date>")
	b.WriteString(today())
	b.WriteString("</date></document-info>")
	b.WriteString("</description>")

	b.WriteString("<body>")
	for _, ch := range book.Chapters {
		b.WriteString("<section>")
		if strings.TrimSpace(ch.Title) != "" {
			b.WriteString("<title><p>")
			b.WriteString(escapeXML(strings.TrimSpace(ch.Title)))
			b.WriteString("</p></title>")
		}
		content := rebaseChapterLinks(ch.Content, ch.ContentPath, book.Chapters, func(c bookparser.ChapterData) string {
			return fmt.Sprintf("section:%d", c.Index)
		})
		for _, n := range fragmentNodes(content) {
			writeFB2Node(n, &b, imgIndex)
		}
		b.WriteString("</section>")
	}
	b.WriteString("</body>")

	if len(book.Metadata.CoverData) > 0 {
		b.WriteString(`<binary id="cover" content-type="`)
		b.WriteString(mediaTypeForBytes(book.Metadata.CoverData, book.Metadata.CoverType))
		b.WriteString(`">`)
		b.WriteString(base64Encode(book.Metadata.CoverData))
		b.WriteString("</binary>")
	}
	for i, img := range images {
		b.WriteString(`<binary id="fb2img`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`" content-type="`)
		b.WriteString(mediaType(img.Src))
		b.WriteString(`">`)
		b.WriteString(base64Encode(img.Data))
		b.WriteString("</binary>")
	}
	b.WriteString("</FictionBook>")
	return []byte(b.String()), nil
}

func writeFB2Node(n *nethtml.Node, b *strings.Builder, imgIndex map[string]int) {
	switch n.Type {
	case nethtml.TextNode:
		b.WriteString(escapeXML(n.Data))
	case nethtml.ElementNode:
		switch strings.ToLower(n.Data) {
		case "li":
			b.WriteString("<p>")
			writeFB2Children(n, b, imgIndex)
			b.WriteString("</p>")
		case "h1", "h2", "h3", "h4", "h5", "h6":
			b.WriteString("<p><strong>")
			writeFB2Children(n, b, imgIndex)
			b.WriteString("</strong></p>")
		case "p", "blockquote", "cite":
			b.WriteString("<p>")
			writeFB2Children(n, b, imgIndex)
			b.WriteString("</p>")
		case "strong", "b":
			b.WriteString("<strong>")
			writeFB2Children(n, b, imgIndex)
			b.WriteString("</strong>")
		case "em", "i":
			b.WriteString("<emphasis>")
			writeFB2Children(n, b, imgIndex)
			b.WriteString("</emphasis>")
		case "img":
			if idx, ok := imgIndex[base(fileAttr(n, "src"))]; ok {
				b.WriteString(`<image l:href="#fb2img`)
				b.WriteString(strconv.Itoa(idx + 1))
				b.WriteString(`"/>`)
			}
		case "a":
			if href := fileAttr(n, "href"); href != "" && href != "#" {
				b.WriteString(`<a href="`)
				b.WriteString(escapeXML(href))
				b.WriteString(`">`)
				writeFB2Children(n, b, imgIndex)
				b.WriteString("</a>")
			} else {
				writeFB2Children(n, b, imgIndex)
			}
		case "br":
			b.WriteString("\n")
		default:
			writeFB2Children(n, b, imgIndex)
		}
	}
}

func writeFB2Children(n *nethtml.Node, b *strings.Builder, imgIndex map[string]int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		writeFB2Node(c, b, imgIndex)
	}
}
