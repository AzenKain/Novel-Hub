package plain

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
)

type Parser struct{}

var (
	markdownImageRegex = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	htmlImageRegex     = regexp.MustCompile(`(?is)<img\b[^>]*\bsrc=["']([^"']+)["'][^>]*>`)
)

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read text metadata: %w", err)
	}
	text := string(data)
	title := ""
	if bookparser.FormatFromPath(filePath) == "md" {
		title = bookparser.FirstMarkdownHeading(text)
	}
	if title == "" {
		title = bookparser.TitleFromPath(filePath)
	}
	meta := &bookparser.BookMetadata{
		Title:       title,
		Description: firstTextPreview(text),
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
	return []bookparser.ChapterData{{
		Title:       bookparser.TitleFromPath(filePath),
		ContentPath: "document",
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
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read text content: %w", err)
	}
	text := string(data)
	if bookparser.FormatFromPath(filePath) == "md" {
		return bookparser.MarkdownToHTML(text), nil
	}
	return bookparser.PlainTextToHTML(text), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid text asset path")
	}
	baseDir := filepath.Dir(filePath)
	target := filepath.Join(baseDir, filepath.FromSlash(assetPath))
	rel, err := filepath.Rel(baseDir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("text asset path escapes document directory")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read text asset: %w", err)
	}
	return data, nil
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	if format := bookparser.FormatFromPath(filePath); format != "md" && format != "markdown" {
		return []string{}, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read markdown images: %w", err)
	}
	seen := map[string]struct{}{}
	var images []string
	add := func(value string) {
		value = strings.TrimSpace(strings.Trim(value, `"'`))
		if value == "" || !isLocalPlainAsset(value) {
			return
		}
		value = filepath.ToSlash(value)
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		images = append(images, value)
	}
	for _, match := range markdownImageRegex.FindAllStringSubmatch(string(data), -1) {
		if len(match) > 1 {
			add(match[1])
		}
	}
	for _, match := range htmlImageRegex.FindAllStringSubmatch(string(data), -1) {
		if len(match) > 1 {
			add(match[1])
		}
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func isLocalPlainAsset(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower != "" &&
		!strings.HasPrefix(lower, "http://") &&
		!strings.HasPrefix(lower, "https://") &&
		!strings.HasPrefix(lower, "data:") &&
		!strings.HasPrefix(lower, "#")
}

func firstTextPreview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return strings.TrimSpace(text[:500]) + "..."
	}
	return text
}
