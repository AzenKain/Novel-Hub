package archivebook

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/docx"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/bookparser/fb2"
	"novelhub/pkg/bookparser/htmlfile"
	"novelhub/pkg/bookparser/plain"
	"novelhub/pkg/bookparser/rtf"
)

type Parser struct {
	format string
}

type entryData struct {
	name   string
	format string
	data   []byte
}

func NewParser(format string) *Parser {
	return &Parser{format: bookparser.NormalizeFormat(format)}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	entry, err := readPrimaryEntry(filePath)
	if err != nil {
		return nil, err
	}
	meta, err := withEntryParser(entry, func(parser bookparser.Parser, path string) (*bookparser.BookMetadata, error) {
		return parser.ParseMetadata(path)
	})
	if err != nil {
		return nil, err
	}
	if meta.Title == "" || strings.HasPrefix(meta.Title, "novelhub-archive-") {
		meta.Title = bookparser.TitleFromPath(entry.name)
	}
	return bookparser.MergeMetadataSidecar(filePath, meta), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	entry, err := readPrimaryEntry(filePath)
	if err != nil {
		return nil, err
	}
	meta, _ := p.ParseMetadata(filePath)
	title := bookparser.TitleFromPath(entry.name)
	if meta != nil && meta.Title != "" {
		title = meta.Title
	}
	chapters, err := withEntryParser(entry, func(parser bookparser.Parser, path string) ([]bookparser.ChapterData, error) {
		return parser.ParseSpine(path)
	})
	if err != nil || len(chapters) == 0 {
		return []bookparser.ChapterData{{
			Title:       title,
			ContentPath: entry.name,
			Index:       0,
		}}, nil
	}
	for index := range chapters {
		chapters[index].ContentPath = entry.name + "/" + strings.TrimLeft(filepath.ToSlash(chapters[index].ContentPath), "/")
		chapters[index].Index = index
	}
	return chapters, nil
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
	entry, err := readPrimaryEntry(filePath)
	if err != nil {
		return "", err
	}
	innerContentPath := ""
	if strings.HasPrefix(filepath.ToSlash(contentPath), entry.name+"/") {
		innerContentPath = strings.TrimPrefix(filepath.ToSlash(contentPath), entry.name+"/")
	}
	return withEntryParser(entry, func(parser bookparser.Parser, path string) (string, error) {
		if innerContentPath != "" {
			return parser.GetChapterContent(path, innerContentPath)
		}
		book, err := parser.ParseBook(path)
		if err != nil {
			return "", err
		}
		if len(book.Chapters) == 0 {
			return `<article><p>No readable text was found in this archive.</p></article>`, nil
		}
		var out strings.Builder
		out.WriteString("<article>")
		for _, chapter := range book.Chapters {
			content := chapter.Content
			if content == "" {
				content, _ = parser.GetChapterContent(path, chapter.ContentPath)
			}
			content = strings.TrimSpace(content)
			content = strings.TrimPrefix(content, "<article>")
			content = strings.TrimSuffix(content, "</article>")
			out.WriteString(content)
		}
		out.WriteString("</article>")
		return out.String(), nil
	})
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid archive asset path")
	}
	if data, err := readNamedEntry(filePath, assetPath); err == nil {
		return data, nil
	}
	entry, err := readPrimaryEntry(filePath)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(assetPath, entry.name+"/") {
		assetPath = strings.TrimPrefix(assetPath, entry.name+"/")
	}
	return withEntryParser(entry, func(parser bookparser.Parser, path string) ([]byte, error) {
		return parser.GetAsset(path, assetPath)
	})
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()
	var images []string
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(file.Name)
		if isImage(name) {
			images = append(images, name)
		}
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func readPrimaryEntry(filePath string) (*entryData, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()
	for _, wanted := range []string{"fb2", "epub", "kepub.epub", "html", "htm", "md", "markdown", "txt", "rtf", "docx"} {
		for _, file := range reader.File {
			if file.FileInfo().IsDir() || entryFormat(file.Name) != wanted {
				continue
			}
			data, err := readZipFile(file)
			if err != nil {
				return nil, err
			}
			return &entryData{name: filepath.ToSlash(file.Name), format: wanted, data: data}, nil
		}
	}
	return nil, fmt.Errorf("archive does not contain a readable book file")
}

func readNamedEntry(filePath string, name string) ([]byte, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != name {
			continue
		}
		return readZipFile(file)
	}
	return nil, fmt.Errorf("archive asset %q not found", name)
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open archive entry: %w", err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func withEntryParser[T any](entry *entryData, fn func(bookparser.Parser, string) (T, error)) (T, error) {
	var zero T
	parser := parserForEntry(entry.format)
	if parser == nil {
		return zero, fmt.Errorf("archive entry %q is not readable", entry.name)
	}
	temp, err := os.CreateTemp("", "novelhub-archive-*."+entry.format)
	if err != nil {
		return zero, fmt.Errorf("create temp archive entry: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(entry.data); err != nil {
		_ = temp.Close()
		return zero, fmt.Errorf("write temp archive entry: %w", err)
	}
	if err := temp.Close(); err != nil {
		return zero, fmt.Errorf("close temp archive entry: %w", err)
	}
	return fn(parser, tempPath)
}

func parserForEntry(format string) bookparser.Parser {
	switch format {
	case "epub", "kepub.epub":
		return epub.NewParser()
	case "fb2":
		return fb2.NewParser()
	case "html", "htm":
		return htmlfile.NewParser()
	case "md", "markdown", "txt":
		return plain.NewParser()
	case "rtf":
		return rtf.NewParser()
	case "docx":
		return docx.NewParser()
	default:
		return nil
	}
}

func entryFormat(name string) string {
	return bookparser.FormatFromPath(name)
}

func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".avif", ".svg":
		return true
	default:
		return false
	}
}
