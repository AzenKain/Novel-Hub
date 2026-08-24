package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
)

type Relationship struct {
	Id     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type Relationships struct {
	XMLName xml.Name       `xml:"Relationships"`
	Items   []Relationship `xml:"Relationship"`
}

type runStyle struct {
	bold         bool
	italic       bool
	underline    bool
	strike       bool
	doubleStrike bool
	caps         bool
	smallCaps    bool
	superScript  bool
	subScript    bool
	fontSize     string
	fontColor    string
	highlight    string
}

type paragraphStyle struct {
	indentLeft  string
	indentRight string
	indentFirst string
	spaceBefore string
	spaceAfter  string
	lineSpacing string
}

type paragraphElement interface {
	isElement()
}

type textElement struct {
	text  string
	style runStyle
}

func (*textElement) isElement() {}

type imageElement struct {
	path string
}

func (*imageElement) isElement() {}

type brElement struct{}

func (*brElement) isElement() {}

type tabElement struct{}

func (*tabElement) isElement() {}

type hyperlinkElement struct {
	url   string
	style runStyle
	text  string
}

func (*hyperlinkElement) isElement() {}

type tableCell struct {
	elements []paragraphElement
	colSpan  int
	rowSpan  int
}

type tableRow struct {
	cells []tableCell
}

type tableElement struct {
	rows []tableRow
}

func (*tableElement) isElement() {}

type paragraph struct {
	style     string
	align     string
	isList    bool
	isQuote   bool
	pStyle    paragraphStyle
	elements  []paragraphElement
	tableData *tableElement
}

var malformedOfficeBulletPrefix = regexp.MustCompile(`^(?:[□]\s*\?\s*|[ðï]\s*\?\s*)+`)

func parseRelationships(r *zip.Reader) (map[string]string, error) {
	relsFile := findZipFile(r.File, "word/_rels/document.xml.rels")
	if relsFile == nil {
		return nil, nil
	}
	rc, err := relsFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var rels Relationships
	if err := xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&rels); err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, rel := range rels.Items {
		target := rel.Target
		if !strings.HasPrefix(target, "/") {
			target = "word/" + target
		} else {
			target = strings.TrimPrefix(target, "/")
		}
		target = filepath.ToSlash(filepath.Clean(target))
		m[rel.Id] = target
	}
	return m, nil
}

func cleanParagraph(p *paragraph) {
	var firstTextEl *textElement
	for _, el := range p.elements {
		if te, ok := el.(*textElement); ok {
			firstTextEl = te
			break
		}
	}

	if firstTextEl != nil {
		text := firstTextEl.text
		trimmedText := strings.TrimSpace(text)
		trimmedText = strings.NewReplacer(
			"", "•",
			"", "•",
			"", "•",
			"", "•",
			"", "•",
			"•", "•",
			"", "•",
			"", "•",
			"□?", "• ",
			"□ ?", "• ",
			"?", "• ",
			"ð?", "• ",
			"ï?", "• ",
		).Replace(trimmedText)

		if loc := malformedOfficeBulletPrefix.FindStringIndex(trimmedText); loc != nil && loc[0] == 0 {
			trimmedText = "• " + trimmedText[loc[1]:]
		}

		if strings.HasPrefix(trimmedText, "•") {
			p.isList = true
			trimmedText = "• " + strings.TrimSpace(strings.TrimPrefix(trimmedText, "•"))
		}

		firstTextEl.text = trimmedText
	}

	if p.isList && firstTextEl != nil {
		if !strings.HasPrefix(firstTextEl.text, "•") {
			firstTextEl.text = "• " + strings.TrimSpace(firstTextEl.text)
		}
	}
}

func parseDocument(r io.Reader, relMap map[string]string) ([]*paragraph, error) {
	decoder := xml.NewDecoder(r)
	var paragraphs []*paragraph

	var currentParagraph *paragraph
	var currentStyle runStyle
	inRun := false
	inHyperlink := false
	var hyperlinkStyle runStyle
	var hyperlinkText strings.Builder
	var hyperlinkURL string

	var runText strings.Builder

	flushRunText := func() {
		if inRun && runText.Len() > 0 {
			currentParagraph.elements = append(currentParagraph.elements, &textElement{
				text:  runText.String(),
				style: currentStyle,
			})
			runText.Reset()
		}
	}

	flushHyperlinkText := func() {
		if inHyperlink && hyperlinkText.Len() > 0 {
			currentParagraph.elements = append(currentParagraph.elements, &hyperlinkElement{
				url:   hyperlinkURL,
				style: hyperlinkStyle,
				text:  hyperlinkText.String(),
			})
			hyperlinkText.Reset()
		}
	}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode docx document: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				currentParagraph = &paragraph{}
			case "pStyle":
				if currentParagraph != nil {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							currentParagraph.style = attr.Value
							if strings.Contains(strings.ToLower(attr.Value), "quote") ||
								strings.Contains(strings.ToLower(attr.Value), "block") {
								currentParagraph.isQuote = true
							}
						}
					}
				}
			case "ind":
				if currentParagraph != nil {
					for _, attr := range t.Attr {
						switch attr.Name.Local {
						case "left", "start":
							if v, err := strconv.Atoi(attr.Value); err == nil {
								currentParagraph.pStyle.indentLeft = fmt.Sprintf("%dpt", v/20)
							}
						case "right", "end":
							if v, err := strconv.Atoi(attr.Value); err == nil {
								currentParagraph.pStyle.indentRight = fmt.Sprintf("%dpt", v/20)
							}
						case "firstLine", "hanging":
							if v, err := strconv.Atoi(attr.Value); err == nil {
								currentParagraph.pStyle.indentFirst = fmt.Sprintf("%dpt", v/20)
							}
						}
					}
				}
			case "spacing":
				if inRun {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							if currentParagraph != nil {
								currentParagraph.pStyle.lineSpacing = attr.Value
							}
						}
					}
				} else if currentParagraph != nil {
					for _, attr := range t.Attr {
						switch attr.Name.Local {
						case "before":
							if v, err := strconv.Atoi(attr.Value); err == nil && v > 0 {
								currentParagraph.pStyle.spaceBefore = fmt.Sprintf("%dpt", v/20)
							}
						case "after":
							if v, err := strconv.Atoi(attr.Value); err == nil && v > 0 {
								currentParagraph.pStyle.spaceAfter = fmt.Sprintf("%dpt", v/20)
							}
						case "line":
							if v, err := strconv.Atoi(attr.Value); err == nil && v > 0 {
								currentParagraph.pStyle.lineSpacing = fmt.Sprintf("%.1f", float64(v)/240.0)
							}
						}
					}
				}
			case "jc":
				if currentParagraph != nil {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							switch strings.ToLower(attr.Value) {
							case "center":
								currentParagraph.align = "center"
							case "right", "end":
								currentParagraph.align = "right"
							case "both", "distribute":
								currentParagraph.align = "justify"
							case "left", "start":
								currentParagraph.align = "left"
							}
						}
					}
				}
			case "numPr":
				if currentParagraph != nil && !strings.HasPrefix(strings.ToLower(currentParagraph.style), "heading") {
					currentParagraph.isList = true
				}
			case "r":
				inRun = true
				currentStyle = runStyle{}
				runText.Reset()
			case "b":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						currentStyle.bold = true
					}
				}
			case "i":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						currentStyle.italic = true
					}
				}
			case "u":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "none" && val != "0" {
						currentStyle.underline = true
					}
				}
			case "strike":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						currentStyle.strike = true
					}
				}
			case "dstrike":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						currentStyle.doubleStrike = true
					}
				}
			case "caps":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						currentStyle.caps = true
					}
				}
			case "smallCaps":
				if inRun {
					val := "true"
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							val = attr.Value
						}
					}
					if val != "false" && val != "0" {
						currentStyle.smallCaps = true
					}
				}
			case "vertAlign":
				if inRun {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							switch strings.ToLower(attr.Value) {
							case "superscript":
								currentStyle.superScript = true
							case "subscript":
								currentStyle.subScript = true
							}
						}
					}
				}
			case "sz", "szCs":
				if inRun {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							if v, err := strconv.Atoi(attr.Value); err == nil && v > 0 {
								currentStyle.fontSize = fmt.Sprintf("%.0fpt", float64(v)/2.0)
							}
						}
					}
				}
			case "color":
				if inRun {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" && len(attr.Value) == 6 {
							currentStyle.fontColor = "#" + strings.ToLower(attr.Value)
						}
					}
				}
			case "highlight":
				if inRun {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							currentStyle.highlight = strings.ToLower(attr.Value)
						}
					}
				}
			case "highlightRef":
				if inRun {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							currentStyle.highlight = strings.ToLower(attr.Value)
						}
					}
				}
			case "hyperlink":
				flushRunText()
				inHyperlink = true
				hyperlinkStyle = runStyle{}
				hyperlinkText.Reset()
				hyperlinkURL = ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "id" && relMap != nil {
						if url, ok := relMap[attr.Value]; ok {
							hyperlinkURL = url
						}
					}
				}
			case "tbl":
				if currentParagraph != nil {
					flushRunText()
				}
				table, err := parseTable(decoder, relMap)
				if err == nil && table != nil && len(table.rows) > 0 {
					tablePara := &paragraph{tableData: table}
					paragraphs = append(paragraphs, tablePara)
				}
			case "tr":
			case "tc":
			case "t":
				var text string
				if err := decoder.DecodeElement(&text, &t); err != nil {
					return nil, fmt.Errorf("decode docx text: %w", err)
				}
				if inHyperlink {
					hyperlinkText.WriteString(text)
				} else if inRun && currentParagraph != nil {
					runText.WriteString(text)
				}
			case "tab":
				if inRun && currentParagraph != nil {
					flushRunText()
					currentParagraph.elements = append(currentParagraph.elements, &tabElement{})
				}
			case "br", "cr":
				if inRun && currentParagraph != nil {
					flushRunText()
					currentParagraph.elements = append(currentParagraph.elements, &brElement{})
				}
			case "blip":
				if currentParagraph != nil {
					flushRunText()
					var embedId string
					for _, attr := range t.Attr {
						if attr.Name.Local == "embed" {
							embedId = attr.Value
						}
					}
					if embedId != "" && relMap != nil {
						if imgPath, ok := relMap[embedId]; ok {
							currentParagraph.elements = append(currentParagraph.elements, &imageElement{
								path: imgPath,
							})
						}
					}
				}
			case "imagedata":
				if currentParagraph != nil {
					flushRunText()
					var id string
					for _, attr := range t.Attr {
						if attr.Name.Local == "id" {
							id = attr.Value
						}
					}
					if id != "" && relMap != nil {
						if imgPath, ok := relMap[id]; ok {
							currentParagraph.elements = append(currentParagraph.elements, &imageElement{
								path: imgPath,
							})
						}
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "r":
				flushRunText()
				inRun = false
			case "hyperlink":
				if inHyperlink {
					flushHyperlinkText()
					inHyperlink = false
				}
			case "p":
				if currentParagraph != nil {
					cleanParagraph(currentParagraph)
					paragraphs = append(paragraphs, currentParagraph)
					currentParagraph = nil
				}
			}
		}
	}
	return paragraphs, nil
}

func clean(value string) string {
	var out strings.Builder
	for _, r := range value {
		switch {
		case r == 0 || r == 1 || r == 2 || r == 3 || r == 4 || r == 5 || r == 6 || r == 7 || r == 8 || r == 19 || r == 20 || r == 21:
			continue
		case r == '\r' || r == '\n' || r == '\v' || r == '\f':
			continue
		case r == '\t' || r == ' ':
			out.WriteRune(r)
		default:
			if (r >= 0 && r <= 8) || (r >= 11 && r <= 12) || (r >= 14 && r <= 31) {
				continue
			}
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}

func getTOCLevel(style string) int {
	lower := strings.ToLower(style)
	if !strings.Contains(lower, "toc") {
		return 0
	}
	for i := 1; i <= 9; i++ {
		suffix := fmt.Sprintf("%d", i)
		if strings.HasSuffix(lower, suffix) || strings.HasSuffix(lower, " "+suffix) {
			return i
		}
	}
	if strings.Contains(lower, "toc") {
		return 1
	}
	return 0
}

func extractDocumentText(r io.Reader) (string, error) {
	paragraphs, err := parseDocument(r, nil)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, p := range paragraphs {
		var pText strings.Builder
		for _, el := range p.elements {
			switch te := el.(type) {
			case *textElement:
				pText.WriteString(te.text)
			case *tabElement:
				pText.WriteByte('\t')
			case *brElement:
				pText.WriteByte('\n')
			}
		}
		line := bookparser.CleanOfficeTextLine(pText.String())
		if line != "" {
			out.WriteString(line)
			out.WriteString("\n")
		}
	}
	return out.String(), nil
}

func isDocxImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".tif", ".tiff":
		return true
	default:
		return false
	}
}

func findZipFile(files []*zip.File, name string) *zip.File {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return nil
}
