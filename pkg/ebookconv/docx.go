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

func writeDOCX(book *bookparser.BookData) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := writeZipString(zw, "[Content_Types].xml", docxContentTypes()); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "_rels/.rels", docxRootRels()); err != nil {
		return nil, err
	}
	if err := writeZipString(zw, "word/document.xml", docxDocument(book)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func docxContentTypes() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` + "\n" +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` + "\n" +
		`<Default Extension="xml" ContentType="application/xml"/>` + "\n" +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` + "\n" +
		`</Types>` + "\n"
}

func docxRootRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + "\n" +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` + "\n" +
		`</Relationships>` + "\n"
}

func docxDocument(book *bookparser.BookData) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + "\n")
	b.WriteString(`<w:body>` + "\n")

	title := fallbackTitle(book)
	if strings.TrimSpace(book.Metadata.Author) != "" {
		title += " — " + strings.TrimSpace(book.Metadata.Author)
	}
	b.WriteString(docxParagraph(title, "48", true, true))

	for _, ch := range book.Chapters {
		if strings.TrimSpace(ch.Title) != "" {
			b.WriteString(docxParagraph(strings.TrimSpace(ch.Title), "36", true, false))
		}
		for _, block := range splitParagraphs(ch.Content) {
			tag, inner, _ := strings.Cut(block, "|")
			text := renderToText(inner)
			if strings.TrimSpace(text) == "" {
				continue
			}
			bold := strings.HasPrefix(tag, "h")
			b.WriteString(docxParagraph(inner, "24", bold, false))
		}
	}
	b.WriteString(`<w:sectPr/></w:body></w:document>` + "\n")
	return b.String()
}

// renderToText flattens an inline HTML snippet to plain text (for empty
// detection); the DOCX run builder walks the snippet for bold/italic.
func renderToText(inner string) string {
	var sb strings.Builder
	for _, n := range fragmentNodes(inner) {
		writeText(n, &sb)
	}
	return sb.String()
}

func docxParagraph(inner string, sz string, bold bool, italic bool) string {
	var runs strings.Builder
	writeRuns(fragmentNodes(inner), &runs, bold, italic)
	if runs.Len() == 0 {
		return ""
	}
	return `<w:p><w:pPr><w:rPr><w:sz w:val="` + sz + `"/><w:szCs w:val="` + sz + `"/></w:rPr></w:pPr>` + runs.String() + `</w:p>` + "\n"
}

func writeRuns(nodes []*nethtml.Node, runs *strings.Builder, bold bool, italic bool) {
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
				writeRuns(childrenOf(n), runs, true, italic)
			case "i":
				writeRuns(childrenOf(n), runs, bold, true)
			case "img":
				// images are dropped from DOCX output (text-level export)
			default:
				writeRuns(childrenOf(n), runs, bold, italic)
			}
		}
	}
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