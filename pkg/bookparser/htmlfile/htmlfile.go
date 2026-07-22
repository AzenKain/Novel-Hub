package htmlfile

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read html metadata: %w", err)
	}
	title, description := htmlMetadata(data)
	if title == "" {
		title = bookparser.TitleFromPath(filePath)
	}
	return bookparser.MergeMetadataSidecar(filePath, &bookparser.BookMetadata{
		Title:       title,
		Description: description,
	}), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	meta, _ := p.ParseMetadata(filePath)
	title := bookparser.TitleFromPath(filePath)
	if meta != nil && meta.Title != "" {
		title = meta.Title
	}
	return []bookparser.ChapterData{{
		Title:       title,
		ContentPath: filepath.Base(filePath),
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
		return "", fmt.Errorf("read html content: %w", err)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return `<article><p>No readable text was found in this file.</p></article>`, nil
	}
	return content, nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid html asset path")
	}
	baseDir := filepath.Dir(filePath)
	target := filepath.Join(baseDir, filepath.FromSlash(assetPath))
	rel, err := filepath.Rel(baseDir, target)
	if err != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return nil, fmt.Errorf("html asset path escapes document directory")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read html asset: %w", err)
	}
	return data, nil
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read html images: %w", err)
	}
	doc, err := nethtml.Parse(bytes.NewReader(data))
	if err != nil {
		return []string{}, nil
	}
	images := make([]string, 0)
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode && strings.EqualFold(node.Data, "img") {
			if src := attrValue(node, "src"); src != "" && isLocalRef(src) {
				images = append(images, src)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func htmlMetadata(data []byte) (string, string) {
	doc, err := nethtml.Parse(bytes.NewReader(data))
	if err != nil {
		return "", ""
	}
	title := ""
	description := ""
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode {
			switch strings.ToLower(node.Data) {
			case "title":
				if title == "" {
					title = bookparser.CleanChapterTitle(textContent(node))
				}
			case "meta":
				name := strings.ToLower(attrValue(node, "name"))
				property := strings.ToLower(attrValue(node, "property"))
				if description == "" && (name == "description" || property == "og:description") {
					description = strings.TrimSpace(attrValue(node, "content"))
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return title, description
}

func attrValue(node *nethtml.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func textContent(node *nethtml.Node) string {
	var out strings.Builder
	var walk func(*nethtml.Node)
	walk = func(current *nethtml.Node) {
		if current.Type == nethtml.TextNode {
			_, _ = io.WriteString(&out, current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return out.String()
}

func isLocalRef(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value != "" &&
		!strings.HasPrefix(value, "http://") &&
		!strings.HasPrefix(value, "https://") &&
		!strings.HasPrefix(value, "data:") &&
		!strings.HasPrefix(value, "#")
}
