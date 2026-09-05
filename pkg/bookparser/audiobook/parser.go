package audiobook

import (
	"fmt"
	"os"
	"strings"

	"github.com/dhowden/tag"
	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
	"novelhub/pkg/jsonx"
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
		merged := bookparser.MergeMetadataSidecar(filePath, &bookparser.BookMetadata{
			Title: bookparser.TitleFromPath(filePath),
		})
		return withCoverFallback(merged), nil
	}

	meta := &bookparser.BookMetadata{
		Title:  m.Title(),
		Author: m.Artist(),
		Series: m.Album(),
	}

	if raw := m.Raw(); raw != nil {
		for k, v := range raw {
			if strings.Contains(strings.ToLower(k), "asin") {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					meta.MetadataJSON, _ = jsonx.MarshalString(map[string]string{"asin": strings.TrimSpace(s)})
					break
				}
			}
		}
	}

	if meta.Title == "" {
		meta.Title = bookparser.TitleFromPath(filePath)
	}

	pic := m.Picture()
	if pic != nil {
		meta.CoverData = pic.Data
		meta.CoverType = pic.MIMEType
	}

	return withCoverFallback(bookparser.MergeMetadataSidecar(filePath, meta)), nil
}

func withCoverFallback(meta *bookparser.BookMetadata) *bookparser.BookMetadata {
	if len(meta.CoverData) == 0 {
		meta.CoverData = defaultcover.GenerateSVG(meta.Title, meta.Author)
		meta.IsDefaultCover = true
		meta.CoverType = "image/svg+xml"
	}
	return meta
}

func (p *AudiobookParser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
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
