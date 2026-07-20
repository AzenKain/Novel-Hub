package external

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"novelhub/pkg/bookparser"
)

type Parser struct {
	format string
}

var errConverterUnavailable = errors.New("converter unavailable")

func NewParser(format string) *Parser {
	return &Parser{format: bookparser.NormalizeFormat(format)}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	return bookparser.MergeMetadataSidecar(filePath, &bookparser.BookMetadata{
		Title: bookparser.TitleFromPath(filePath),
	}), nil
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
	text, err := p.extractText(filePath)
	if err != nil {
		if errors.Is(err, errConverterUnavailable) {
			return converterHintHTML(p.format), nil
		}
		return "", err
	}
	if p.format == "pdf" {
		return bookparser.PreformattedTextToHTML(text), nil
	}
	return bookparser.PlainTextToHTML(text), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assets, err := readExternalImageAssets(filePath)
	if err != nil {
		return nil, err
	}
	return bookparser.FindEmbeddedImageAsset(assets, assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	assets, err := readExternalImageAssets(filePath)
	if err != nil {
		return nil, err
	}
	return bookparser.EmbeddedImageNames(assets), nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func readExternalImageAssets(filePath string) ([]bookparser.EmbeddedImageAsset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read embedded assets: %w", err)
	}
	return bookparser.ExtractEmbeddedImageAssets(data), nil
}

func (p *Parser) extractText(filePath string) (string, error) {
	format := p.format
	if format == "" {
		format = bookparser.FormatFromPath(filePath)
	}
	switch format {
	case "pdf":
		if text, err := pdftotext(filePath); err == nil {
			return text, nil
		}
		return ebookConvertToText(filePath)
	case "mobi", "azw", "azw3":
		return ebookConvertToText(filePath)
	case "doc":
		if text, err := antiword(filePath); err == nil {
			return text, nil
		}
		if text, err := catdoc(filePath); err == nil {
			return text, nil
		}
		return libreOfficeToText(filePath)
	default:
		return "", fmt.Errorf("external parser does not support %q", format)
	}
}

func pdftotext(filePath string) (string, error) {
	return runTextCommand(45*time.Second, "pdftotext", "-layout", filePath, "-")
}

func antiword(filePath string) (string, error) {
	return runTextCommand(45*time.Second, "antiword", filePath)
}

func catdoc(filePath string) (string, error) {
	return runTextCommand(45*time.Second, "catdoc", filePath)
}

func ebookConvertToText(filePath string) (string, error) {
	if _, err := exec.LookPath("ebook-convert"); err != nil {
		return "", fmt.Errorf("%w: ebook-convert", errConverterUnavailable)
	}
	tmpDir, err := os.MkdirTemp("", "novelhub-convert-*")
	if err != nil {
		return "", fmt.Errorf("create convert temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outPath := filepath.Join(tmpDir, "book.txt")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ebook-convert", filePath, outPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return "", fmt.Errorf("ebook-convert timed out: %w", ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("ebook-convert failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("read ebook-convert output: %w", err)
	}
	return string(data), nil
}

func libreOfficeToText(filePath string) (string, error) {
	bin := ""
	for _, candidate := range []string{"soffice", "libreoffice"} {
		if _, err := exec.LookPath(candidate); err == nil {
			bin = candidate
			break
		}
	}
	if bin == "" {
		return "", fmt.Errorf("%w: soffice/libreoffice", errConverterUnavailable)
	}
	tmpDir, err := os.MkdirTemp("", "novelhub-office-*")
	if err != nil {
		return "", fmt.Errorf("create office temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--headless", "--convert-to", "txt:Text", "--outdir", tmpDir, filePath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return "", fmt.Errorf("office conversion timed out: %w", ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("office conversion failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	outPath := filepath.Join(tmpDir, strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))+".txt")
	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("read office output: %w", err)
	}
	return string(data), nil
}

func runTextCommand(timeout time.Duration, name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", fmt.Errorf("%w: %s", errConverterUnavailable, name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return "", fmt.Errorf("%s timed out: %w", name, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func converterHintHTML(format string) string {
	format = strings.ToUpper(format)
	return `<article><h2>` + format + ` reader needs a converter</h2><p>This file is attached, but NovelHub could not find a local converter for this format.</p><p>Install one of the supported tools and try opening the book again: <code>ebook-convert</code> for MOBI/AZW/AZW3/PDF, <code>pdftotext</code> for PDF, or <code>antiword</code>/<code>catdoc</code>/<code>soffice</code> for DOC.</p></article>`
}
