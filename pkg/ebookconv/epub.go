package ebookconv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

func writeEPUB(book *bookparser.BookData, images []Image) ([]byte, error) {
	return writeEPUBWithKobo(book, images, false)
}

func writeEPUBWithKobo(book *bookparser.BookData, images []Image, kobo bool) ([]byte, error) {
	imgIndex := imageLookup(images)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// mimetype must be the first entry, uncompressed.
	hdr := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mw, err := zw.CreateHeader(hdr)
	if err != nil {
		return nil, err
	}
	if _, err := mw.Write([]byte("application/epub+zip")); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "META-INF/container.xml", containerXML()); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "OEBPS/content.opf", opfXML(book, images)); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "OEBPS/toc.ncx", ncxXML(book)); err != nil {
		return nil, err
	}
	for i, ch := range book.Chapters {
		body := rebaseChapterLinks(ch.Content, ch.ContentPath, book.Chapters, func(c bookparser.ChapterData) string {
			return "chapter_" + strconv.Itoa(c.Index+1) + ".xhtml"
		})
		body = rebaseImages(body, imgIndex)
		if kobo {
			body = kepubify(body)
		}
		xhtml := chapterXHTML(book, ch, body)
		if err := writeZipString(zw, "OEBPS/chapter_"+strconv.Itoa(i+1)+".xhtml", xhtml); err != nil {
			return nil, err
		}
	}
	for i, img := range images {
		w, err := zw.Create("OEBPS/images/" + img.Name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(img.Data); err != nil {
			return nil, err
		}
		_ = i
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeZipString(zw *zip.Writer, name string, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(content))
	return err
}

func containerXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` + "\n" +
		`  <rootfiles>` + "\n" +
		`    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>` + "\n" +
		`  </rootfiles>` + "\n" +
		`</container>` + "\n"
}

func opfXML(book *bookparser.BookData, images []Image) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">` + "\n")
	b.WriteString(`  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">` + "\n")
	b.WriteString(`    <dc:identifier id="uid">`)
	b.WriteString(escapeXML(newID()))
	b.WriteString(`</dc:identifier>`)
	b.WriteString("\n")
	b.WriteString(`    <dc:title>`)
	b.WriteString(escapeXML(fallbackTitle(book)))
	b.WriteString(`</dc:title>`)
	b.WriteString("\n")
	if strings.TrimSpace(book.Metadata.Author) != "" {
		b.WriteString(`    <dc:creator>`)
		b.WriteString(escapeXML(strings.TrimSpace(book.Metadata.Author)))
		b.WriteString(`</dc:creator>`)
		b.WriteString("\n")
	}
	lang := book.Metadata.Language
	if lang == "" {
		lang = "en"
	}
	b.WriteString(`    <dc:language>`)
	b.WriteString(escapeXML(lang))
	b.WriteString(`</dc:language>`)
	b.WriteString("\n")
	if strings.TrimSpace(book.Metadata.Description) != "" {
		b.WriteString(`    <dc:description>`)
		b.WriteString(escapeXML(strings.TrimSpace(book.Metadata.Description)))
		b.WriteString(`</dc:description>`)
		b.WriteString("\n")
	}
	b.WriteString(`    <meta name="generator" content="NovelHub"/>` + "\n")
	b.WriteString(`  </metadata>` + "\n")

	b.WriteString(`  <manifest>` + "\n")
	b.WriteString(`    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>` + "\n")
	for i := range book.Chapters {
		b.WriteString(`    <item id="chapter_`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`" href="chapter_`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`.xhtml" media-type="application/xhtml+xml"/>`)
		b.WriteString("\n")
	}
	for i, img := range images {
		b.WriteString(`    <item id="img_`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`" href="images/`)
		b.WriteString(escapeXML(img.Name))
		b.WriteString(`" media-type="`)
		b.WriteString(mediaType(img.Src))
		b.WriteString(`"/>`)
		b.WriteString("\n")
	}
	b.WriteString(`  </manifest>` + "\n")

	b.WriteString(`  <spine toc="ncx">` + "\n")
	for i := range book.Chapters {
		b.WriteString(`    <itemref idref="chapter_`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`"/>`)
		b.WriteString("\n")
	}
	b.WriteString(`  </spine>` + "\n")
	b.WriteString(`</package>` + "\n")
	return b.String()
}

func ncxXML(book *bookparser.BookData) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">` + "\n")
	b.WriteString(`  <head><meta name="dtb:uid" content="`)
	b.WriteString(escapeXML(newID()))
	b.WriteString(`"/></head>`)
	b.WriteString("\n")
	b.WriteString(`  <docTitle><text>`)
	b.WriteString(escapeXML(fallbackTitle(book)))
	b.WriteString(`</text></docTitle>`)
	b.WriteString("\n")
	b.WriteString(`  <navMap>` + "\n")
	for i, ch := range book.Chapters {
		title := strings.TrimSpace(ch.Title)
		if title == "" {
			title = fmt.Sprintf("Chapter %d", i+1)
		}
		b.WriteString(`    <navPoint id="nav_`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`" playOrder="`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`">`)
		b.WriteString("\n")
		b.WriteString(`      <navLabel><text>`)
		b.WriteString(escapeXML(title))
		b.WriteString(`</text></navLabel>`)
		b.WriteString("\n")
		b.WriteString(`      <content src="chapter_`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`.xhtml"/>`)
		b.WriteString("\n")
		b.WriteString(`    </navPoint>` + "\n")
	}
	b.WriteString(`  </navMap>` + "\n")
	b.WriteString(`</ncx>` + "\n")
	return b.String()
}

func chapterXHTML(book *bookparser.BookData, ch bookparser.ChapterData, body string) string {
	title := strings.TrimSpace(ch.Title)
	if title == "" {
		title = fallbackTitle(book)
	}
	lang := book.Metadata.Language
	if lang == "" {
		lang = "en"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<!DOCTYPE html>` + "\n" +
		`<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="` + escapeXML(lang) + `" lang="` + escapeXML(lang) + `">` + "\n" +
		`<head><meta charset="utf-8"/><title>` + escapeXML(title) + `</title>` + "\n" +
		`<style>body{margin:1em;font-family:serif;line-height:1.5}img{max-width:100%;height:auto}</style>` + "\n" +
		`</head>` + "\n" +
		`<body>` + "\n" +
		body + "\n" +
		`</body>` + "\n" +
		`</html>` + "\n"
}

// rebaseImages rewrites every <img src="..."> whose basename matches a source
// image so it points at the output location under images/.
func rebaseImages(content string, imgIndex map[string]int) string {
	nodes := fragmentNodes(content)
	if len(nodes) == 0 {
		return content
	}
	changed := false
	var rewrite func(n *nethtml.Node)
	rewrite = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			switch n.Data {
			case "img":
				for i, a := range n.Attr {
					if a.Key == "src" {
						if idx, ok := imgIndex[base(a.Val)]; ok {
							n.Attr[i].Val = "images/" + "img_" + strconv.Itoa(idx+1) + imageExt(a.Val)
							changed = true
						}
					}
				}
			case "image":
				for i, a := range n.Attr {
					if (a.Namespace == "xlink" && a.Key == "href") || a.Key == "xlink:href" {
						if idx, ok := imgIndex[base(a.Val)]; ok {
							n.Attr[i].Val = "images/" + "img_" + strconv.Itoa(idx+1) + imageExt(a.Val)
							changed = true
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rewrite(c)
		}
	}
	for _, n := range nodes {
		rewrite(n)
	}
	if !changed {
		return content
	}
	var b strings.Builder
	for _, n := range nodes {
		_ = nethtml.Render(&b, n)
	}
	return b.String()
}
