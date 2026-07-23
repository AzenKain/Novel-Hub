package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
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

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	documentFile := findZipFile(r.File, "word/document.xml")
	if documentFile == nil {
		return "", fmt.Errorf("docx document.xml not found")
	}
	rc, err := documentFile.Open()
	if err != nil {
		return "", fmt.Errorf("open docx document: %w", err)
	}
	defer rc.Close()

	text, err := extractDocumentText(rc)
	if err != nil {
		return "", err
	}
	return bookparser.PlainTextToHTML(text), nil
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
	decoder := xml.NewDecoder(r)
	var out strings.Builder
	var paragraph strings.Builder
	inParagraph := false
	isListParagraph := false

	flush := func() {
		value := bookparser.CleanOfficeTextLine(paragraph.String())
		if value != "" {
			if isListParagraph && !strings.HasPrefix(value, "•") {
				value = "• " + value
			}
			if out.Len() > 0 {
				out.WriteString("\n\n")
			}
			out.WriteString(value)
		}
		paragraph.Reset()
		isListParagraph = false
	}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode docx document: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
				isListParagraph = false
				paragraph.Reset()
			case "numPr":
				if inParagraph {
					isListParagraph = true
				}
			case "t":
				var text string
				if err := decoder.DecodeElement(&text, &t); err != nil {
					return "", fmt.Errorf("decode docx text: %w", err)
				}
				if inParagraph {
					paragraph.WriteString(text)
				}
			case "tab":
				if inParagraph {
					paragraph.WriteByte('\t')
				}
			case "br", "cr":
				if inParagraph {
					paragraph.WriteByte('\n')
				}
			}
		case xml.EndElement:
			if t.Name.Local == "p" && inParagraph {
				flush()
				inParagraph = false
			}
		}
	}
	if inParagraph || paragraph.Len() > 0 {
		flush()
	}
	return out.String(), nil
}

func clean(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
