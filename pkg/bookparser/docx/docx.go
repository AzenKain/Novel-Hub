package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"html"
	"path/filepath"
	"strconv"
	"strings"

	"novelhub/pkg/bookparser"
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

	var props coreProperties
	if err := xml.NewDecoder(rc).Decode(&props); err != nil {
		return bookparser.MergeMetadataSidecar(filePath, meta), nil
	}

	if props.Title != "" {
		meta.Title = props.Title
	}
	if props.Creator != "" {
		meta.Author = props.Creator
	}
	if props.Description != "" {
		meta.Description = props.Description
	}
	if props.Subject != "" {
		meta.Subjects = []string{props.Subject}
	}

	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		images, err := p.ListImages(filePath)
		if err == nil && len(images) > 0 {
			coverData, err := p.GetAsset(filePath, images[0])
			if err == nil && len(coverData) > 0 {
				merged.CoverData = coverData
				ext := strings.ToLower(filepath.Ext(images[0]))
				if ext == ".png" {
					merged.CoverType = "image/png"
				} else {
					merged.CoverType = "image/jpeg"
				}
			}
		}
	}

	return merged, nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	meta, _ := p.ParseMetadata(filePath)
	title := bookparser.TitleFromPath(filePath)
	if meta != nil && meta.Title != "" {
		title = meta.Title
	}

	paragraphs, err := p.readDocxParagraphs(filePath)
	if err != nil {
		return []bookparser.ChapterData{{
			Title:       title,
			ContentPath: "word/document.xml",
			Index:       0,
		}}, nil
	}

	sections := splitDocxSections(paragraphs)
	if len(sections) <= 1 {
		return []bookparser.ChapterData{{
			Title:       title,
			ContentPath: "word/document.xml",
			Index:       0,
		}}, nil
	}

	var chapters []bookparser.ChapterData
	for i, sec := range sections {
		chapters = append(chapters, bookparser.ChapterData{
			Title:       sec.title,
			ContentPath: fmt.Sprintf("docx-section:%d", i),
			Index:       i,
		})
	}
	return chapters, nil
}

type docxSection struct {
	title string
	start int
	end   int
}

func splitDocxSections(paragraphs []*paragraph) []docxSection {
	var sections []docxSection
	var pending docxSection
	pending.start = 0

	flushPending := func(end int) {
		if end > pending.start {
			sections = append(sections, docxSection{title: pending.title, start: pending.start, end: end})
		}
	}

	isHeading1 := func(p *paragraph) bool {
		lower := strings.ToLower(p.style)
		return lower == "heading1" || lower == "heading 1"
	}

	for i, p := range paragraphs {
		if isHeading1(p) {
			flushPending(i)
			pending = docxSection{title: paragraphPlainText(p), start: i}
		}
	}
	flushPending(len(paragraphs))
	return sections
}

func (p *Parser) readDocxParagraphs(filePath string) ([]*paragraph, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	documentFile := findZipFile(r.File, "word/document.xml")
	if documentFile == nil {
		return nil, fmt.Errorf("docx document file not found")
	}
	rc, err := documentFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	relMap, err := parseRelationships(&r.Reader)
	if err != nil {
		relMap = nil
	}
	return parseDocument(rc, relMap)
}

func paragraphPlainText(p *paragraph) string {
	var pText strings.Builder
	for _, el := range p.elements {
		switch te := el.(type) {
		case *textElement:
			pText.WriteString(te.text)
		case *hyperlinkElement:
			pText.WriteString(te.text)
		case *tabElement:
			pText.WriteByte('\t')
		}
	}
	return bookparser.CleanOfficeTextLine(pText.String())
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

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	documentFile := findZipFile(r.File, "word/document.xml")
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

	start, end := 0, len(paragraphs)
	if sectionIdx, ok := strings.CutPrefix(contentPath, "docx-section:"); ok {
		n, convErr := strconv.Atoi(sectionIdx)
		sections := splitDocxSections(paragraphs)
		if convErr != nil || n < 0 || n >= len(sections) {
			return "", fmt.Errorf("docx section %q out of range", contentPath)
		}
		start, end = sections[n].start, sections[n].end
	}

	var out strings.Builder
	out.WriteString("<article>")
	renderDocxParagraphs(&out, paragraphs[start:end], contentPath)
	out.WriteString("</article>")
	return out.String(), nil
}

func renderDocxParagraphs(out *strings.Builder, paragraphs []*paragraph, contentPath string) {
	for _, p := range paragraphs {
		if p.tableData != nil {
			out.WriteString(renderTable(p.tableData))
			continue
		}
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
				if te.style.strike {
					escaped = "<s>" + escaped + "</s>"
				}
				if te.style.doubleStrike {
					escaped = `<s class="double-strike">` + escaped + `</s>`
				}
				if te.style.superScript {
					escaped = "<sup>" + escaped + "</sup>"
				}
				if te.style.subScript {
					escaped = "<sub>" + escaped + "</sub>"
				}
				if te.style.caps {
					escaped = `<span class="uppercase">` + escaped + `</span>`
				}
				if te.style.smallCaps {
					escaped = `<span class="small-caps">` + escaped + `</span>`
				}
				if te.style.fontColor != "" {
					escaped = fmt.Sprintf(`<span style="color:%s">%s</span>`, te.style.fontColor, escaped)
				}
				if te.style.fontSize != "" {
					escaped = fmt.Sprintf(`<span style="font-size:%s">%s</span>`, te.style.fontSize, escaped)
				}
				if te.style.highlight != "" && te.style.highlight != "none" {
					escaped = fmt.Sprintf(`<mark class="hl-%s">%s</mark>`, te.style.highlight, escaped)
				}
				pContent.WriteString(escaped)
			case *hyperlinkElement:
				escaped := html.EscapeString(te.text)
				if te.style.bold {
					escaped = "<b>" + escaped + "</b>"
				}
				if te.style.italic {
					escaped = "<i>" + escaped + "</i>"
				}
				if te.url != "" {
					pContent.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(te.url), escaped))
				} else {
					pContent.WriteString(escaped)
				}
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
		} else if p.isQuote {
			tag = "blockquote"
		}

		var attrs strings.Builder
		if p.align != "" {
			fmt.Fprintf(&attrs, ` align="%s"`, p.align)
		}
		var styleParts []string

		if tocLevel := getTOCLevel(p.style); tocLevel > 0 {
			indent := (tocLevel - 1) * 20
			styleParts = append(styleParts, fmt.Sprintf("margin-left:%dpt", indent))
			styleParts = append(styleParts, "text-indent:0")
		}

		if p.pStyle.indentLeft != "" {
			styleParts = append(styleParts, "margin-left:"+p.pStyle.indentLeft)
		}
		if p.pStyle.indentRight != "" {
			styleParts = append(styleParts, "margin-right:"+p.pStyle.indentRight)
		}
		if p.pStyle.indentFirst != "" {
			styleParts = append(styleParts, "text-indent:"+p.pStyle.indentFirst)
		}
		if p.pStyle.spaceBefore != "" {
			styleParts = append(styleParts, "margin-top:"+p.pStyle.spaceBefore)
		}
		if p.pStyle.spaceAfter != "" {
			styleParts = append(styleParts, "margin-bottom:"+p.pStyle.spaceAfter)
		}
		if p.pStyle.lineSpacing != "" {
			styleParts = append(styleParts, "line-height:"+p.pStyle.lineSpacing)
		}

		if len(styleParts) > 0 {
			fmt.Fprintf(&attrs, ` style="%s"`, strings.Join(styleParts, ";"))
		}

		if p.isList {
			prefix := "• "
			if !strings.HasPrefix(pText, prefix) {
				pText = prefix + pText
			}
			out.WriteString(fmt.Sprintf("<p%s>%s</p>\n", attrs.String(), pText))
		} else {
			out.WriteString(fmt.Sprintf("<%s%s>%s</%s>\n", tag, attrs.String(), pText, tag))
		}
	}
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	for _, file := range r.File {
		if filepath.ToSlash(file.Name) == assetPath {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return bookparser.ReadAllLimit(rc, 50*1024*1024)
		}
	}
	return nil, fmt.Errorf("asset not found: %s", assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	relMap, err := parseRelationships(&r.Reader)
	if err != nil {
		relMap = nil
	}

	var images []string
	for _, file := range r.File {
		name := filepath.Base(file.Name)
		if isDocxImage(name) {
			path := file.Name
			for _, target := range relMap {
				if strings.HasSuffix(target, name) || strings.Contains(file.Name, name) {
					path = target
					break
				}
			}
			images = append(images, path)
		}
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}
