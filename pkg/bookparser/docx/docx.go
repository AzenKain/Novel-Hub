package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
)

type Parser struct{}

type coreProperties struct {
	Title       string `xml:"title"`
	Creator     string `xml:"creator"`
	Description string `xml:"description"`
	Subject     string `xml:"subject"`
	Language    string `xml:"language"`
	Created     string `xml:"created"`
	Modified    string `xml:"modified"`
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	coreFile := findZipFile(r.File, "docProps/core.xml")
	if coreFile == nil {
		return bookparser.MergeMetadataSidecar(filePath, meta), nil
	}
	rc, err := coreFile.Open()
	if err != nil {
		return bookparser.MergeMetadataSidecar(filePath, meta), nil
	}
	defer rc.Close()

	var core coreProperties
	if err := xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&core); err != nil {
		return bookparser.MergeMetadataSidecar(filePath, meta), nil
	}
	if value := clean(core.Title); value != "" {
		meta.Title = value
	}
	meta.Author = clean(core.Creator)
	meta.Description = clean(core.Description)
	meta.Language = clean(core.Language)
	meta.Date = clean(core.Created)
	if meta.Date == "" {
		meta.Date = clean(core.Modified)
	}
	if subject := clean(core.Subject); subject != "" {
		meta.Subjects = []string{subject}
	}
	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			ext := strings.ToLower(filepath.Ext(images[0]))
			if ext == ".png" {
				meta.CoverType = "image/png"
			} else {
				meta.CoverType = "image/jpeg"
			}
		}
	}
	return bookparser.MergeMetadataSidecar(filePath, meta), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	meta, _ := p.ParseMetadata(filePath)
	title := bookparser.TitleFromPath(filePath)
	if meta != nil && meta.Title != "" {
		title = meta.Title
	}
	return []bookparser.ChapterData{{
		Title:       title,
		ContentPath: "word/document.xml",
		Index:       0,
	}}, nil
}

func (p *Parser) ParseBook(filePath string) (*bookparser.BookData, error) {
	meta, err := p.ParseMetadata(filePath)
	if err != nil {
		return nil, err
	}
	chapters, err := p.ParseSpine(filePath)
	if err != nil {
		return nil, err
	}
	for i := range chapters {
		content, err := p.GetChapterContent(filePath, chapters[i].ContentPath)
		if err == nil {
			chapters[i].Content = content
		}
	}
	return &bookparser.BookData{Metadata: *meta, Chapters: chapters}, nil
}

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
	bold      bool
	italic    bool
	underline bool
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

type paragraph struct {
	style    string
	isList   bool
	elements []paragraphElement
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
			"\uf0b7", "•",
			"\uf0a7", "•",
			"\uf0d8", "•",
			"\uf0fc", "•",
			"\uf06c", "•",
			"\u2022", "•",
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
						}
					}
				}
			case "numPr":
				if currentParagraph != nil {
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
			case "t":
				var text string
				if err := decoder.DecodeElement(&text, &t); err != nil {
					return nil, fmt.Errorf("decode docx text: %w", err)
				}
				if inRun && currentParagraph != nil {
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

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	documentFile := findZipFile(r.File, contentPath)
	if documentFile == nil {
		documentFile = findZipFile(r.File, "word/document.xml")
	}
	if documentFile == nil {
		return "", fmt.Errorf("docx document file not found")
	}
	rc, err := documentFile.Open()
	if err != nil {
		return "", fmt.Errorf("open docx document: %w", err)
	}
	defer rc.Close()

	relMap, err := parseRelationships(&r.Reader)
	if err != nil {
		relMap = nil
	}

	paragraphs, err := parseDocument(rc, relMap)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	out.WriteString("<article>")
	for _, p := range paragraphs {
		var pContent strings.Builder
		for _, el := range p.elements {
			switch te := el.(type) {
			case *textElement:
				escaped := html.EscapeString(te.text)
				if te.style.bold {
					escaped = "<b>" + escaped + "</b>"
				}
				if te.style.italic {
					escaped = "<i>" + escaped + "</i>"
				}
				if te.style.underline {
					escaped = "<u>" + escaped + "</u>"
				}
				pContent.WriteString(escaped)
			case *imageElement:
				baseDir := filepath.Dir(contentPath)
				relPath := te.path
				if baseDir != "." && baseDir != "" {
					prefix := filepath.ToSlash(baseDir) + "/"
					if after, ok := strings.CutPrefix(relPath, prefix); ok {
						relPath = after
					}
				}
				pContent.WriteString(`<img src="`)
				pContent.WriteString(html.EscapeString(relPath))
				pContent.WriteString(`" />`)
			case *brElement:
				pContent.WriteString("<br>")
			case *tabElement:
				pContent.WriteByte('\t')
			}
		}

		pText := pContent.String()
		if pText == "" {
			continue
		}

		tag := "p"
		if strings.HasPrefix(p.style, "Heading") {
			level := strings.TrimPrefix(p.style, "Heading")
			if level == "1" || level == "2" || level == "3" || level == "4" || level == "5" || level == "6" {
				tag = "h" + level
			} else {
				tag = "h2"
			}
		}

		out.WriteString("<")
		out.WriteString(tag)
		out.WriteString(">")
		out.WriteString(pText)
		out.WriteString("</")
		out.WriteString(tag)
		out.WriteString(">\n")
	}
	out.WriteString("</article>")
	return out.String(), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid docx asset path")
	}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()
	file := findZipFile(r.File, assetPath)
	if file == nil {
		return nil, fmt.Errorf("docx asset %q not found", assetPath)
	}
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open docx asset: %w", err)
	}
	defer rc.Close()
	return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()
	var images []string
	for _, file := range r.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "word/media/") && isDocxImage(name) {
			images = append(images, name)
		}
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
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
		val := bookparser.CleanOfficeTextLine(pText.String())
		if p.isList && val != "" && !strings.HasPrefix(val, "•") {
			val = "• " + val
		}
		if val != "" {
			if out.Len() > 0 {
				out.WriteString("\n\n")
			}
			out.WriteString(val)
		}
	}
	return out.String(), nil
}

func clean(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
