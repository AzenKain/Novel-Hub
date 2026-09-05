package ebookconv

import (
	"archive/zip"
	"bytes"
	"errors"
	"strconv"
	"strings"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

func writeDOCX(book *bookparser.BookData, images []Image) ([]byte, error) {
	imgIndex := imageLookup(images)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := writeZipString(zw, "[Content_Types].xml", docxContentTypes(images)); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "_rels/.rels", docxRootRels()); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "word/document.xml", docxDocument(book, imgIndex)); err != nil {
		return nil, err
	}
	if len(images) > 0 {
		if err := writeZipString(zw, "word/_rels/document.xml.rels", docxDocRels(images)); err != nil {
			return nil, err
		}
		for i, img := range images {
			w, err := zw.Create("word/media/image" + strconv.Itoa(i+1) + imageExt(img.Src))
			if err != nil {
				return nil, err
			}
			if _, err := w.Write(img.Data); err != nil {
				return nil, err
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func docxContentTypes(images []Image) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` + "\n")
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` + "\n")
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>` + "\n")
	seen := map[string]bool{}
	for _, img := range images {
		ext := strings.TrimPrefix(imageExt(img.Src), ".")
		if ext == "jpeg" {
			ext = "jpg"
		}
		if seen[ext] {
			continue
		}
		seen[ext] = true
		switch ext {
		case "png":
			b.WriteString(`<Default Extension="png" ContentType="image/png"/>` + "\n")
		case "gif":
			b.WriteString(`<Default Extension="gif" ContentType="image/gif"/>` + "\n")
		case "webp":
			b.WriteString(`<Default Extension="webp" ContentType="image/webp"/>` + "\n")
		case "bmp":
			b.WriteString(`<Default Extension="bmp" ContentType="image/bmp"/>` + "\n")
		case "svg":
			b.WriteString(`<Default Extension="svg" ContentType="image/svg+xml"/>` + "\n")
		default:
			b.WriteString(`<Default Extension="jpg" ContentType="image/jpeg"/>` + "\n")
		}
	}
	b.WriteString(`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` + "\n")
	b.WriteString(`</Types>` + "\n")
	return b.String()
}

func docxRootRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + "\n" +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` + "\n" +
		`</Relationships>` + "\n"
}

func docxDocRels(images []Image) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + "\n")
	for i, img := range images {
		b.WriteString(`<Relationship Id="rIdImg`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(imageExt(img.Src))
		b.WriteString(`"/>`)
		b.WriteString("\n")
	}
	b.WriteString(`</Relationships>` + "\n")
	return b.String()
}

func docxDocument(book *bookparser.BookData, imgIndex map[string]int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` + "\n")
	b.WriteString(`<w:body>` + "\n")

	title := fallbackTitle(book)
	if strings.TrimSpace(book.Metadata.Author) != "" {
		title += " — " + strings.TrimSpace(book.Metadata.Author)
	}
	b.WriteString(docxParagraph(title, "48", true, true, imgIndex))

	for _, ch := range book.Chapters {
		if strings.TrimSpace(ch.Title) != "" {
			b.WriteString(docxParagraph(strings.TrimSpace(ch.Title), "36", true, false, imgIndex))
		}
		for _, block := range splitParagraphs(ch.Content) {
			tag, inner, _ := strings.Cut(block, "|")
			text := renderToText(inner)
			if strings.TrimSpace(text) == "" && !strings.Contains(inner, "<img") {
				continue
			}
			bold := strings.HasPrefix(tag, "h")
			b.WriteString(docxParagraph(inner, "24", bold, false, imgIndex))
		}
	}
	b.WriteString(`<w:sectPr/></w:body></w:document>` + "\n")
	return b.String()
}

func renderToText(inner string) string {
	var sb strings.Builder
	for _, n := range fragmentNodes(inner) {
		writeText(n, &sb)
	}
	return sb.String()
}

func docxParagraph(inner string, sz string, bold bool, italic bool, imgIndex map[string]int) string {
	var runs strings.Builder
	writeRuns(fragmentNodes(inner), &runs, bold, italic, imgIndex)
	if runs.Len() == 0 {
		return ""
	}
	return `<w:p><w:pPr><w:rPr><w:sz w:val="` + sz + `"/><w:szCs w:val="` + sz + `"/></w:rPr></w:pPr>` + runs.String() + `</w:p>` + "\n"
}

func writeRuns(nodes []*nethtml.Node, runs *strings.Builder, bold bool, italic bool, imgIndex map[string]int) {
	for _, n := range nodes {
		switch n.Type {
		case nethtml.TextNode:
			if strings.TrimSpace(n.Data) != "" {
				runs.WriteString(`<w:r>`)
				if bold || italic {
					runs.WriteString(`<w:rPr>`)
					if bold {
						runs.WriteString(`<w:b/>`)
					}
					if italic {
						runs.WriteString(`<w:i/>`)
					}
					runs.WriteString(`</w:rPr>`)
				}
				runs.WriteString(`<w:t xml:space="preserve">`)
				runs.WriteString(escapeXML(n.Data))
				runs.WriteString(`</w:t></w:r>`)
			}
		case nethtml.ElementNode:
			switch strings.ToLower(n.Data) {
			case "b":
				writeRuns(childrenOf(n), runs, true, italic, imgIndex)
			case "i":
				writeRuns(childrenOf(n), runs, bold, true, imgIndex)
			case "img":
				runs.WriteString(docxDrawing(imgIndex, fileAttr(n, "src")))
			default:
				writeRuns(childrenOf(n), runs, bold, italic, imgIndex)
			}
		}
	}
}

func docxDrawing(imgIndex map[string]int, src string) string {
	idx, ok := imgIndex[base(src)]
	if !ok {
		return ""
	}
	return `<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="5486400" cy="7315200"/>` +
		`<wp:effectExtent l="0" t="0" r="0" b="0"/>` +
		`<wp:docPr id="` + strconv.Itoa(idx+1) + `" name="Picture ` + strconv.Itoa(idx+1) + `"/>` +
		`<wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:nvPicPr><pic:cNvPr id="` + strconv.Itoa(idx+1) + `" name="Picture ` + strconv.Itoa(idx+1) + `"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="rIdImg` + strconv.Itoa(idx+1) + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="5486400" cy="7315200"/></a:xfrm>` +
		`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>` +
		`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
}

func childrenOf(n *nethtml.Node) []*nethtml.Node {
	var out []*nethtml.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, c)
	}
	return out
}

func writeCBZ(images []Image) ([]byte, error) {
	if len(images) == 0 {
		return nil, errors.New("no images available to export as CBZ")
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i, img := range images {
		name := pad4(i+1) + imageExt(img.Src)
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(img.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pad4(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}
