package doc

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/richardlehane/mscfb"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
	"novelhub/pkg/constants"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	text, _ := extractDocText(filePath)
	meta := &bookparser.BookMetadata{
		Title:       bookparser.TitleFromPath(filePath),
		Description: preview(text),
	}
	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			meta.CoverType = "image/jpeg"
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
	streams, err := readDocStreams(filePath)
	if err != nil {
		return "", err
	}
	word, table := selectTableStream(streams)
	if len(word) == 0 {
		return "", fmt.Errorf("WordDocument stream not found")
	}

	pieces, err := extractPieceTable(word, table)
	if err != nil || len(pieces) == 0 {
		text, err2 := extractDocText(filePath)
		if err2 != nil {
			return "", err2
		}
		if strings.TrimSpace(text) == "" {
			return `<article><p>No readable text was found in this DOC file.</p></article>`, nil
		}
		return bookparser.PlainTextToHTML(text), nil
	}

	docStreams := docStreams{word: word, table: table}
	charMap := buildCharFormatMap(word, docStreams)
	paraMap := buildParaFormatMap(word, docStreams)
	cpBase, cpScale := detectCPScale(word, charMap, paraMap)

	totalRunes := 0
	for _, pc := range pieces {
		totalRunes += len([]rune(pc.text))
	}
	formats := make([]charFormat, totalRunes)
	paraFormats := make([]paraFormat, len(pieces))

	pos := 0
	for pi, pc := range pieces {
		if pf, ok := paraMap[keyCP(cpBase, cpScale, pc.cpStart)]; ok {
			paraFormats[pi] = pf
		}
		cp := pc.cpStart
		for range pc.text {
			if f, ok := charMap[keyCP(cpBase, cpScale, cp)]; ok {
				formats[pos] = f
			}
			cp++
			pos++
		}
	}

	return formatsToHTML(pieces, formats, paraFormats), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assets, err := readDocImageAssets(filePath)
	if err != nil {
		return nil, err
	}
	return bookparser.FindEmbeddedImageAsset(assets, assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	assets, err := readDocImageAssets(filePath)
	if err != nil {
		return nil, err
	}
	return bookparser.EmbeddedImageNames(assets), nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func selectTableStream(streams map[string][]byte) ([]byte, []byte) {
	word := streams["WordDocument"]
	if len(word) == 0 {
		return nil, nil
	}
	tableName := "0Table"
	if len(word) > 0x0c && binary.LittleEndian.Uint16(word[0x0a:0x0c])&0x0200 != 0 {
		tableName = "1Table"
	}
	table := streams[tableName]
	if len(table) == 0 {
		if tableName == "0Table" {
			table = streams["1Table"]
		} else {
			table = streams["0Table"]
		}
	}
	return word, table
}

func readDocImageAssets(filePath string) ([]bookparser.EmbeddedImageAsset, error) {
	streams, err := readDocStreams(filePath)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(streams))
	for name := range streams {
		names = append(names, name)
	}
	sort.Strings(names)
	var assets []bookparser.EmbeddedImageAsset
	for _, name := range names {
		assets = bookparser.AppendEmbeddedImageAssets(assets, streams[name])
	}
	return assets, nil
}

func readDocStreams(filePath string) (map[string][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open doc: %w", err)
	}
	defer file.Close()

	reader, err := mscfb.New(file)
	if err != nil {
		return nil, fmt.Errorf("open doc compound file: %w", err)
	}
	streams := make(map[string][]byte, len(reader.File))
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || entry.Size <= 0 {
			continue
		}
		data, err := bookparser.ReadAllLimit(entry, constants.MaxArchiveAssetSize)
		if err != nil {
			continue
		}
		streams[entry.Name] = data
	}
	return streams, nil
}

func preview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return strings.TrimSpace(text[:500]) + "..."
	}
	return text
}
