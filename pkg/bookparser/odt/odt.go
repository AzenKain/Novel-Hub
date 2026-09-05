package odt

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
	"novelhub/pkg/constants"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	data, err := readZipEntry(filePath, "meta.xml")
	if err != nil {
		return bookparser.MergeMetadataSidecar(filePath, meta), nil
	}
	values := collectODFText(data, map[string]bool{
		"title":         true,
		"creator":       true,
		"description":   true,
		"language":      true,
		"creation-date": true,
		"date":          true,
		"keyword":       true,
	})
	if value := strings.TrimSpace(values["title"]); value != "" {
		meta.Title = value
	}
	meta.Author = strings.TrimSpace(values["creator"])
	meta.Description = strings.TrimSpace(values["description"])
	meta.Language = strings.TrimSpace(values["language"])
	meta.Date = strings.TrimSpace(values["creation-date"])
	if meta.Date == "" {
		meta.Date = strings.TrimSpace(values["date"])
	}
	if keyword := strings.TrimSpace(values["keyword"]); keyword != "" {
		meta.Subjects = splitKeywords(keyword)
	}

	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			meta.CoverType = "image/jpeg"
			if strings.HasSuffix(strings.ToLower(images[0]), ".png") {
				meta.CoverType = "image/png"
			}
		}
	}

	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged, nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	meta, _ := p.ParseMetadata(filePath)
	title := bookparser.TitleFromPath(filePath)
	if meta != nil && meta.Title != "" {
		title = meta.Title
	}
	return []bookparser.ChapterData{{
		Title:       title,
		ContentPath: "content.xml",
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

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	data, err := readZipEntry(filePath, "content.xml")
	if err != nil {
		return "", err
	}
	return contentXMLToHTML(data)
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid odt asset path")
	}
	data, err := readZipEntry(filePath, assetPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open odt: %w", err)
	}
	defer reader.Close()
	var images []string
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "Thumbnails/") || strings.HasPrefix(name, "ObjectReplacements/") {
			continue
		}
		switch strings.ToLower(filepath.Ext(name)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
			images = append(images, name)
		}
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func readZipEntry(filePath string, entryName string) ([]byte, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open odt: %w", err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != entryName {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open odt entry: %w", err)
		}
		defer rc.Close()
		return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
	}
	return nil, fmt.Errorf("odt entry %q not found", entryName)
}

func collectODFText(data []byte, wanted map[string]bool) map[string]string {
	values := make(map[string]string)
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var current string
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return values
		}
		switch t := token.(type) {
		case xml.StartElement:
			if wanted[t.Name.Local] {
				current = t.Name.Local
				text.Reset()
			}
		case xml.CharData:
			if current != "" {
				text.Write([]byte(t))
			}
		case xml.EndElement:
			if current == t.Name.Local {
				values[current] = strings.TrimSpace(text.String())
				current = ""
			}
		}
	}
	return values
}

type odfStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Align     string
}

func parseODFStyles(data []byte) map[string]odfStyle {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	styles := make(map[string]odfStyle)

	var currentStyleName string
	var currentStyle odfStyle
	inStyle := false

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "style" {
				inStyle = true
				currentStyle = odfStyle{}
				for _, attr := range t.Attr {
					if attr.Name.Local == "name" {
						currentStyleName = attr.Value
					}
				}
			} else if inStyle && t.Name.Local == "text-properties" {
				for _, attr := range t.Attr {
					val := strings.ToLower(attr.Value)
					switch attr.Name.Local {
					case "font-weight", "font-weight-asian", "font-weight-complex":
						if val == "bold" || val == "700" || val == "800" || val == "900" {
							currentStyle.Bold = true
						}
					case "font-style", "font-style-asian", "font-style-complex":
						if val == "italic" || val == "oblique" {
							currentStyle.Italic = true
						}
					case "text-underline-style", "text-underline-type":
						if val != "none" && val != "" {
							currentStyle.Underline = true
						}
					case "text-line-through-style", "text-line-through-type":
						if val != "none" && val != "" {
							currentStyle.Strike = true
						}
					}
				}
			} else if inStyle && t.Name.Local == "paragraph-properties" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "text-align" {
						val := strings.ToLower(attr.Value)
						switch val {
						case "center":
							currentStyle.Align = "center"
						case "right", "end":
							currentStyle.Align = "right"
						case "justify":
							currentStyle.Align = "justify"
						case "left", "start":
							currentStyle.Align = "left"
						}
					}
				}
			}
		case xml.EndElement:
			if t.Name.Local == "style" && inStyle {
				if currentStyleName != "" {
					styles[currentStyleName] = currentStyle
				}
				inStyle = false
				currentStyleName = ""
			}
		}
	}
	return styles
}

func contentXMLToHTML(data []byte) (string, error) {
	styles := parseODFStyles(data)
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var out strings.Builder
	var block strings.Builder
	blockTag := ""
	blockAlign := ""
	inBlock := false
	var spanStyleStack []odfStyle

	out.WriteString("<article>")
	flush := func() {
		value := strings.TrimSpace(block.String())
		if value != "" {
			out.WriteByte('<')
			out.WriteString(blockTag)
			if blockAlign != "" {
				out.WriteString(fmt.Sprintf(` align="%s"`, blockAlign))
			}
			out.WriteByte('>')
			out.WriteString(value)
			out.WriteString("</")
			out.WriteString(blockTag)
			out.WriteString(">\n")
		}
		block.Reset()
		blockTag = ""
		blockAlign = ""
		inBlock = false
	}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode odt content: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "h":
				if inBlock {
					flush()
				}
				blockTag = headingTag(t)
				for _, attr := range t.Attr {
					if attr.Name.Local == "style-name" {
						if s, ok := styles[attr.Value]; ok && s.Align != "" {
							blockAlign = s.Align
						}
					}
				}
				inBlock = true
			case "p":
				if inBlock {
					flush()
				}
				blockTag = "p"
				for _, attr := range t.Attr {
					if attr.Name.Local == "style-name" {
						if s, ok := styles[attr.Value]; ok && s.Align != "" {
							blockAlign = s.Align
						}
					}
				}
				inBlock = true
			case "span":
				if inBlock {
					var curStyle odfStyle
					for _, attr := range t.Attr {
						if attr.Name.Local == "style-name" {
							if s, ok := styles[attr.Value]; ok {
								curStyle = s
							}
						}
					}
					spanStyleStack = append(spanStyleStack, curStyle)
					if curStyle.Bold {
						block.WriteString("<b>")
					}
					if curStyle.Italic {
						block.WriteString("<i>")
					}
					if curStyle.Underline {
						block.WriteString("<u>")
					}
					if curStyle.Strike {
						block.WriteString("<s>")
					}
				}
			case "image":
				if inBlock {
					for _, attr := range t.Attr {
						if attr.Name.Local == "href" && isRenderableODFImage(attr.Value) {
							block.WriteString(fmt.Sprintf(`<img src="%s" style="max-width: 100%%; height: auto;" />`, html.EscapeString(attr.Value)))
						}
					}
				}
			case "tab":
				if inBlock {
					block.WriteByte('\t')
				}
			case "line-break":
				if inBlock {
					block.WriteString("<br>")
				}
			case "s":
				if inBlock {
					block.WriteString(strings.Repeat(" ", odfSpaceCount(t)))
				}
			}
		case xml.CharData:
			if inBlock {
				block.WriteString(html.EscapeString(string(t)))
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "span":
				if inBlock && len(spanStyleStack) > 0 {
					top := spanStyleStack[len(spanStyleStack)-1]
					spanStyleStack = spanStyleStack[:len(spanStyleStack)-1]
					if top.Strike {
						block.WriteString("</s>")
					}
					if top.Underline {
						block.WriteString("</u>")
					}
					if top.Italic {
						block.WriteString("</i>")
					}
					if top.Bold {
						block.WriteString("</b>")
					}
				}
			case "h", "p":
				if inBlock {
					flush()
				}
			}
		}
	}
	if inBlock {
		flush()
	}
	out.WriteString("</article>")
	return out.String(), nil
}

func isRenderableODFImage(href string) bool {
	switch strings.ToLower(filepath.Ext(href)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

func headingTag(element xml.StartElement) string {
	for _, attr := range element.Attr {
		if attr.Name.Local != "outline-level" {
			continue
		}
		level, err := strconv.Atoi(attr.Value)
		if err != nil || level < 1 {
			return "h2"
		}
		if level > 6 {
			level = 6
		}
		return "h" + strconv.Itoa(level)
	}
	return "h2"
}

func odfSpaceCount(element xml.StartElement) int {
	for _, attr := range element.Attr {
		if attr.Name.Local != "c" {
			continue
		}
		count, err := strconv.Atoi(attr.Value)
		if err == nil && count > 0 && count < 100 {
			return count
		}
	}
	return 1
}

func splitKeywords(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})
	subjects := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			subjects = append(subjects, field)
		}
	}
	return subjects
}
