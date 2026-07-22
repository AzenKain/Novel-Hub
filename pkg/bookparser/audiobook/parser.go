package audiobook

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
	"novelhub/pkg/bookparser"
)

type AudiobookParser struct{}

func New() *AudiobookParser {
	return &AudiobookParser{}
}

func (p *AudiobookParser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	m, err := tag.ReadFrom(file)
	if err != nil {
		return &bookparser.BookMetadata{
			Title: bookparser.TitleFromPath(filePath),
		}, nil
	}

	meta := &bookparser.BookMetadata{
		Title:  m.Title(),
		Author: m.Artist(),
		Series: m.Album(),
	}

	if meta.Title == "" {
		meta.Title = bookparser.TitleFromPath(filePath)
	}

	pic := m.Picture()
	if pic != nil {
		meta.CoverData = pic.Data
		meta.CoverType = pic.MIMEType
	}

	return bookparser.MergeMetadataSidecar(filePath, meta), nil
}

func (p *AudiobookParser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	// For basic mp3/m4b, we return a single chapter pointing to the file itself.
	// Extracting actual chapters from m4b requires a specialized mp4 parser which is beyond basic ID3.
	return []bookparser.ChapterData{
		{
			Title:       "Audiobook",
			Content:     "Audiobook Content",
			ContentPath: bookparser.RawFileContentPath,
			Index:       0,
		},
	}, nil
}

func (p *AudiobookParser) ParseBook(filePath string) (*bookparser.BookData, error) {
	meta, err := p.ParseMetadata(filePath)
	if err != nil {
		return nil, err
	}
	spine, err := p.ParseSpine(filePath)
	if err != nil {
		return nil, err
	}
	return &bookparser.BookData{
		Metadata: *meta,
		Chapters: spine,
	}, nil
}

func (p *AudiobookParser) GetChapterContent(filePath, contentPath string) (string, error) {
	return "", nil
}

func (p *AudiobookParser) GetAsset(filePath, assetPath string) ([]byte, error) {
	if assetPath == bookparser.RawFileContentPath {
		return os.ReadFile(filePath)
	}
	return nil, os.ErrNotExist
}

func (p *AudiobookParser) ListImages(filePath string) ([]string, error) {
	return nil, nil
}

func (p *AudiobookParser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return nil
}
