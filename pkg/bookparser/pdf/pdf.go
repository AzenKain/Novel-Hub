package pdf

import (
	"fmt"
	"os"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{
		Title: bookparser.TitleFromPath(filePath),
	}

	assets, err := readPDFImageAssets(filePath)
	if err == nil && len(assets) > 0 {
		meta.CoverData = assets[0].Data
		if strings.HasSuffix(assets[0].Name, ".png") {
			meta.CoverType = "image/png"
		} else {
			meta.CoverType = "image/jpeg"
		}
	} else {
		svgCover := defaultcover.GenerateSVG(meta.Title, meta.Author)
		meta.CoverData = []byte(svgCover)
		meta.CoverType = "image/svg+xml"
	}

	return bookparser.MergeMetadataSidecar(filePath, meta), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	return []bookparser.ChapterData{{
		Title:       bookparser.TitleFromPath(filePath),
		ContentPath: bookparser.RawFileContentPath,
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
	return &bookparser.BookData{Metadata: *meta, Chapters: chapters}, nil
}

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	return "", fmt.Errorf("pdf content is rendered through the raw-file reader")
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assets, err := readPDFImageAssets(filePath)
	if err != nil {
		return nil, err
	}
	return bookparser.FindEmbeddedImageAsset(assets, assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	assets, err := readPDFImageAssets(filePath)
	if err != nil {
		return nil, err
	}
	return bookparser.EmbeddedImageNames(assets), nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func readPDFImageAssets(filePath string) ([]bookparser.EmbeddedImageAsset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read pdf assets: %w", err)
	}
	return bookparser.ExtractEmbeddedImageAssets(data), nil
}
