package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	title := bookparser.TitleFromPath(filePath)
	meta := &bookparser.BookMetadata{
		Title:       title,
		Description: fmt.Sprintf("Data spreadsheet / CSV document: %s", filepath.Base(filePath)),
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
	if meta != nil && strings.TrimSpace(meta.Title) != "" {
		title = meta.Title
	}
	return []bookparser.ChapterData{{
		Title:       title,
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
		return "", fmt.Errorf("read csv content: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	isTSV := ext == ".tsv"

	reader := csv.NewReader(bytes.NewReader(data))
	if isTSV {
		reader.Comma = '\t'
	}
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			lineStr := string(data)
			lines := strings.Split(lineStr, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if isTSV {
					rows = append(rows, strings.Split(line, "\t"))
				} else {
					rows = append(rows, strings.Split(line, ","))
				}
			}
			break
		}
		isEmpty := true
		for _, cell := range record {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			rows = append(rows, record)
		}
	}

	if len(rows) == 0 {
		return "<article><p>No data rows found in spreadsheet.</p></article>", nil
	}

	var out strings.Builder
	out.WriteString(`<article class="novelhub-spreadsheet">`)
	out.WriteString(`<div class="novelhub-table-wrapper" style="overflow-x: auto; max-width: 100%; margin: 1em 0;">`)
	out.WriteString(`<table class="novelhub-table" style="width: 100%; border-collapse: collapse; text-align: left; font-size: 0.9em; line-height: 1.5;">`)

	out.WriteString("<thead><tr>")
	for _, cell := range rows[0] {
		out.WriteString(`<th style="border: 1px solid rgba(128,128,128,0.3); padding: 6px 10px; font-weight: bold; background: rgba(128,128,128,0.08);">`)
		out.WriteString(html.EscapeString(strings.TrimSpace(cell)))
		out.WriteString("</th>")
	}
	out.WriteString("</tr></thead>")

	out.WriteString("<tbody>")
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		bgStyle := ""
		if i%2 == 0 {
			bgStyle = ` style="background: rgba(128,128,128,0.04);"`
		}
		out.WriteString(fmt.Sprintf("<tr%s>", bgStyle))
		for _, cell := range row {
			out.WriteString(`<td style="border: 1px solid rgba(128,128,128,0.2); padding: 5px 10px;">`)
			out.WriteString(html.EscapeString(strings.TrimSpace(cell)))
			out.WriteString("</td>")
		}
		out.WriteString("</tr>")
	}
	out.WriteString("</tbody>")
	out.WriteString("</table></div></article>")

	return out.String(), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	return nil, fmt.Errorf("no assets in csv file")
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	return []string{}, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}
