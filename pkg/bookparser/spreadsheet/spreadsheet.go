package spreadsheet

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

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
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xlsx":
		return p.parseXLSXMetadata(filePath)
	case ".ods":
		return p.parseODSMetadata(filePath)
	case ".xls":
		return p.parseXLSMetadata(filePath)
	default:
		return p.defaultMetadata(filePath), nil
	}
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xlsx":
		return p.parseXLSXSpine(filePath)
	case ".ods":
		return p.parseODSSpine(filePath)
	case ".xls":
		return p.parseXLSSpine(filePath)
	default:
		return p.defaultSpine(filePath), nil
	}
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
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xlsx":
		return p.getXLSXSheetContent(filePath, contentPath)
	case ".ods":
		return p.getODSSheetContent(filePath, contentPath)
	case ".xls":
		return p.getXLSContent(filePath, contentPath)
	default:
		return "<article><p>Unsupported spreadsheet format.</p></article>", nil
	}
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid asset path")
	}

	if ext == ".xls" {
		streams, err := readCompoundStreams(filePath)
		if err != nil {
			return nil, err
		}
		var assets []bookparser.EmbeddedImageAsset
		for _, data := range streams {
			assets = bookparser.AppendEmbeddedImageAssets(assets, data)
		}
		return bookparser.FindEmbeddedImageAsset(assets, assetPath)
	}

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		cleanName := strings.TrimLeft(filepath.ToSlash(f.Name), "/")
		if cleanName == assetPath || strings.HasSuffix(cleanName, "/"+assetPath) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return bookparser.ReadAllLimit(rc, constants.MaxArchiveAssetSize)
		}
	}

	return nil, fmt.Errorf("asset %q not found in spreadsheet", assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".xls" {
		streams, err := readCompoundStreams(filePath)
		if err != nil {
			return []string{}, nil
		}
		var assets []bookparser.EmbeddedImageAsset
		for _, data := range streams {
			assets = bookparser.AppendEmbeddedImageAssets(assets, data)
		}
		return bookparser.EmbeddedImageNames(assets), nil
	}

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return []string{}, nil
	}
	defer r.Close()

	var images []string
	for _, f := range r.File {
		lower := strings.ToLower(f.Name)
		if strings.HasPrefix(lower, "xl/media/") || strings.HasPrefix(lower, "pictures/") || strings.HasPrefix(lower, "thumbnails/") {
			ext := filepath.Ext(lower)
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".svg" {
				images = append(images, f.Name)
			}
		}
	}
	sort.Strings(images)
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func (p *Parser) defaultMetadata(filePath string) *bookparser.BookMetadata {
	meta := &bookparser.BookMetadata{
		Title:       bookparser.TitleFromPath(filePath),
		Description: fmt.Sprintf("Spreadsheet document: %s", filepath.Base(filePath)),
	}
	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged
}

func (p *Parser) defaultSpine(filePath string) []bookparser.ChapterData {
	return []bookparser.ChapterData{{
		Title:       bookparser.TitleFromPath(filePath),
		ContentPath: "sheet:0",
		Index:       0,
	}}
}

func (p *Parser) parseXLSXMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return p.defaultMetadata(filePath), nil
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "docProps/core.xml" {
			rc, err := f.Open()
			if err == nil {
				var core struct {
					Title       string `xml:"title"`
					Creator     string `xml:"creator"`
					Description string `xml:"description"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&core) == nil {
					if t := strings.TrimSpace(core.Title); t != "" {
						meta.Title = t
					}
					meta.Author = strings.TrimSpace(core.Creator)
					meta.Description = strings.TrimSpace(core.Description)
				}
				rc.Close()
			}
			break
		}
	}

	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			if strings.HasSuffix(strings.ToLower(images[0]), ".png") {
				meta.CoverType = "image/png"
			} else {
				meta.CoverType = "image/jpeg"
			}
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

func (p *Parser) parseXLSXSpine(filePath string) ([]bookparser.ChapterData, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer r.Close()

	relMap := make(map[string]string)
	for _, f := range r.File {
		if f.Name == "xl/_rels/workbook.xml.rels" {
			if rc, err := f.Open(); err == nil {
				var rels struct {
					Items []struct {
						ID     string `xml:"Id,attr"`
						Target string `xml:"Target,attr"`
					} `xml:"Relationship"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&rels) == nil {
					for _, item := range rels.Items {
						target := item.Target
						if !strings.HasPrefix(target, "xl/") {
							target = "xl/" + strings.TrimPrefix(target, "/")
						}
						relMap[item.ID] = filepath.ToSlash(filepath.Clean(target))
					}
				}
				rc.Close()
			}
			break
		}
	}

	var chapters []bookparser.ChapterData
	for _, f := range r.File {
		if f.Name == "xl/workbook.xml" {
			rc, err := f.Open()
			if err == nil {
				var wb struct {
					Sheets []struct {
						Name string `xml:"name,attr"`
						RID  string `xml:"id,attr"`
					} `xml:"sheets>sheet"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&wb) == nil {
					for i, s := range wb.Sheets {
						targetPath := relMap[s.RID]
						if targetPath == "" {
							targetPath = fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
						}
						chapters = append(chapters, bookparser.ChapterData{
							Title:       s.Name,
							ContentPath: targetPath,
							Index:       i,
						})
					}
				}
				rc.Close()
			}
			break
		}
	}

	if len(chapters) == 0 {
		var sheetFiles []string
		for _, f := range r.File {
			if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
				sheetFiles = append(sheetFiles, f.Name)
			}
		}
		sort.Strings(sheetFiles)
		for i, sf := range sheetFiles {
			chapters = append(chapters, bookparser.ChapterData{
				Title:       fmt.Sprintf("Sheet %d", i+1),
				ContentPath: sf,
				Index:       i,
			})
		}
	}

	if len(chapters) == 0 {
		return p.defaultSpine(filePath), nil
	}
	return chapters, nil
}

func (p *Parser) getXLSXSheetContent(filePath, contentPath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer r.Close()

	var sharedStrings []string
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			if rc, err := f.Open(); err == nil {
				decoder := xml.NewDecoder(rc)
				var inText bool
				var sb strings.Builder
				for {
					tok, err := decoder.Token()
					if err != nil {
						break
					}
					switch t := tok.(type) {
					case xml.StartElement:
						if t.Name.Local == "t" {
							inText = true
							sb.Reset()
						}
					case xml.EndElement:
						if t.Name.Local == "t" {
							inText = false
						} else if t.Name.Local == "si" {
							sharedStrings = append(sharedStrings, sb.String())
							sb.Reset()
						}
					case xml.CharData:
						if inText {
							sb.Write(t)
						}
					}
				}
				rc.Close()
			}
			break
		}
	}

	var sheetFile *zip.File
	for _, f := range r.File {
		if f.Name == contentPath {
			sheetFile = f
			break
		}
	}
	if sheetFile == nil {
		return "", fmt.Errorf("sheet file %q not found", contentPath)
	}

	rc, err := sheetFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var rows [][]string
	var currentRow []string
	var currentCellVal strings.Builder
	var cellType string
	var inVal bool

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "row" {
				currentRow = nil
			} else if t.Name.Local == "c" {
				cellType = ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "t" {
						cellType = attr.Value
					}
				}
				currentCellVal.Reset()
			} else if t.Name.Local == "v" || t.Name.Local == "t" {
				inVal = true
			}
		case xml.EndElement:
			if t.Name.Local == "row" {
				if len(currentRow) > 0 {
					rows = append(rows, currentRow)
				}
			} else if t.Name.Local == "c" {
				val := currentCellVal.String()
				if cellType == "s" {
					idx, err := strconv.Atoi(val)
					if err == nil && idx >= 0 && idx < len(sharedStrings) {
						val = sharedStrings[idx]
					}
				}
				currentRow = append(currentRow, val)
			} else if t.Name.Local == "v" || t.Name.Local == "t" {
				inVal = false
			}
		case xml.CharData:
			if inVal {
				currentCellVal.Write(t)
			}
		}
	}

	return renderSpreadsheetTableHTML(rows), nil
}

func (p *Parser) parseODSMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return p.defaultMetadata(filePath), nil
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "meta.xml" {
			rc, err := f.Open()
			if err == nil {
				var metaDoc struct {
					Title       string `xml:"meta>title"`
					Creator     string `xml:"meta>initial-creator"`
					Description string `xml:"meta>description"`
				}
				if xml.NewDecoder(io.LimitReader(rc, constants.MaxArchiveAssetSize)).Decode(&metaDoc) == nil {
					if t := strings.TrimSpace(metaDoc.Title); t != "" {
						meta.Title = t
					}
					meta.Author = strings.TrimSpace(metaDoc.Creator)
					meta.Description = strings.TrimSpace(metaDoc.Description)
				}
				rc.Close()
			}
			break
		}
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

func (p *Parser) parseODSSpine(filePath string) ([]bookparser.ChapterData, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open ods: %w", err)
	}
	defer r.Close()

	var contentFile *zip.File
	for _, f := range r.File {
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return p.defaultSpine(filePath), nil
	}

	rc, err := contentFile.Open()
	if err != nil {
		return p.defaultSpine(filePath), nil
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var chapters []bookparser.ChapterData
	tableIdx := 0

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "table" {
			tableName := ""
			for _, attr := range se.Attr {
				if attr.Name.Local == "name" {
					tableName = attr.Value
				}
			}
			if tableName == "" {
				tableName = fmt.Sprintf("Sheet %d", tableIdx+1)
			}
			chapters = append(chapters, bookparser.ChapterData{
				Title:       tableName,
				ContentPath: fmt.Sprintf("table:%d", tableIdx),
				Index:       tableIdx,
			})
			tableIdx++
		}
	}

	if len(chapters) == 0 {
		return p.defaultSpine(filePath), nil
	}
	return chapters, nil
}

func (p *Parser) getODSSheetContent(filePath, contentPath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open ods: %w", err)
	}
	defer r.Close()

	var contentFile *zip.File
	for _, f := range r.File {
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return "", fmt.Errorf("content.xml not found")
	}

	targetIdx := 0
	if strings.HasPrefix(contentPath, "table:") {
		targetIdx, _ = strconv.Atoi(strings.TrimPrefix(contentPath, "table:"))
	}

	rc, err := contentFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	currentTable := -1
	inTargetTable := false

	var rows [][]string
	var currentRow []string
	var currentCellVal strings.Builder
	var repeatCols int = 1

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "table" {
				currentTable++
				inTargetTable = (currentTable == targetIdx)
			}
			if inTargetTable {
				if t.Name.Local == "table-row" {
					currentRow = nil
				} else if t.Name.Local == "table-cell" {
					currentCellVal.Reset()
					repeatCols = 1
					for _, attr := range t.Attr {
						if attr.Name.Local == "number-columns-repeated" {
							if n, err := strconv.Atoi(attr.Value); err == nil && n > 0 && n < 100 {
								repeatCols = n
							}
						}
					}
				}
			}
		case xml.EndElement:
			if t.Name.Local == "table" && inTargetTable {
				inTargetTable = false
				break
			}
			if inTargetTable {
				if t.Name.Local == "table-row" {
					if len(currentRow) > 0 {
						rows = append(rows, currentRow)
					}
				} else if t.Name.Local == "table-cell" {
					val := strings.TrimSpace(currentCellVal.String())
					for r := 0; r < repeatCols; r++ {
						currentRow = append(currentRow, val)
					}
				}
			}
		case xml.CharData:
			if inTargetTable {
				currentCellVal.Write(t)
			}
		}
	}

	return renderSpreadsheetTableHTML(rows), nil
}

func (p *Parser) parseXLSMetadata(filePath string) (*bookparser.BookMetadata, error) {
	meta := &bookparser.BookMetadata{
		Title:       bookparser.TitleFromPath(filePath),
		Description: fmt.Sprintf("Excel Spreadsheet: %s", filepath.Base(filePath)),
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

func (p *Parser) parseXLSSpine(filePath string) ([]bookparser.ChapterData, error) {
	return []bookparser.ChapterData{{
		Title:       bookparser.TitleFromPath(filePath),
		ContentPath: "sheet:0",
		Index:       0,
	}}, nil
}

func (p *Parser) getXLSContent(filePath, contentPath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader, err := mscfb.New(file)
	if err != nil {
		return "", err
	}

	var wbStream []byte
	for _, entry := range reader.File {
		if entry.Name == "Workbook" || entry.Name == "Book" {
			wbStream, err = bookparser.ReadAllLimit(entry, constants.MaxArchiveAssetSize)
			if err != nil {
				return "", err
			}
			break
		}
	}

	if len(wbStream) == 0 {
		return "<article><p>No workbook stream found in XLS file.</p></article>", nil
	}

	rows := parseBIFF8Workbook(wbStream)
	return renderSpreadsheetTableHTML(rows), nil
}

func parseBIFF8Workbook(data []byte) [][]string {
	var sst []string
	type cellPos struct {
		row, col int
		val      string
	}
	var cells []cellPos

	pos := 0
	for pos+4 <= len(data) {
		recType := binary.LittleEndian.Uint16(data[pos : pos+2])
		recLen := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
		pos += 4

		if pos+int(recLen) > len(data) {
			break
		}
		rec := data[pos : pos+int(recLen)]
		pos += int(recLen)

		switch recType {
		case 0x00FC:
			sst = parseBIFF8SST(rec)
		case 0x00FD:
			if len(rec) >= 10 {
				row := int(binary.LittleEndian.Uint16(rec[0:2]))
				col := int(binary.LittleEndian.Uint16(rec[2:4]))
				sstIdx := int(binary.LittleEndian.Uint32(rec[6:10]))
				val := ""
				if sstIdx >= 0 && sstIdx < len(sst) {
					val = sst[sstIdx]
				}
				cells = append(cells, cellPos{row: row, col: col, val: val})
			}
		case 0x0204:
			if len(rec) >= 8 {
				row := int(binary.LittleEndian.Uint16(rec[0:2]))
				col := int(binary.LittleEndian.Uint16(rec[2:4]))
				strLen := int(binary.LittleEndian.Uint16(rec[6:8]))
				if len(rec) >= 8+strLen {
					val := string(rec[8 : 8+strLen])
					cells = append(cells, cellPos{row: row, col: col, val: val})
				}
			}
		case 0x0203:
			if len(rec) >= 14 {
				row := int(binary.LittleEndian.Uint16(rec[0:2]))
				col := int(binary.LittleEndian.Uint16(rec[2:4]))
				bits := binary.LittleEndian.Uint64(rec[6:14])
				val := fmt.Sprintf("%g", math.Float64frombits(bits))
				cells = append(cells, cellPos{row: row, col: col, val: val})
			}
		case 0x027E:
			if len(rec) >= 10 {
				row := int(binary.LittleEndian.Uint16(rec[0:2]))
				col := int(binary.LittleEndian.Uint16(rec[2:4]))
				val := fmt.Sprintf("%g", decodeRK(binary.LittleEndian.Uint32(rec[6:10])))
				cells = append(cells, cellPos{row: row, col: col, val: val})
			}
		}
	}

	if len(cells) == 0 {
		return nil
	}

	maxRow, maxCol := 0, 0
	for _, c := range cells {
		if c.row > maxRow {
			maxRow = c.row
		}
		if c.col > maxCol {
			maxCol = c.col
		}
	}

	if maxRow > 5000 {
		maxRow = 5000
	}
	if maxCol > 100 {
		maxCol = 100
	}

	grid := make([][]string, maxRow+1)
	for r := range grid {
		grid[r] = make([]string, maxCol+1)
	}
	for _, c := range cells {
		if c.row <= maxRow && c.col <= maxCol {
			grid[c.row][c.col] = c.val
		}
	}

	var result [][]string
	for _, row := range grid {
		hasVal := false
		for _, v := range row {
			if strings.TrimSpace(v) != "" {
				hasVal = true
				break
			}
		}
		if hasVal {
			result = append(result, row)
		}
	}
	return result
}

func parseBIFF8SST(data []byte) []string {
	if len(data) < 8 {
		return nil
	}
	uniqueStrings := binary.LittleEndian.Uint32(data[4:8])
	var stringsList []string
	pos := 8

	for i := uint32(0); i < uniqueStrings && pos < len(data); i++ {
		if pos+3 > len(data) {
			break
		}
		cch := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		flags := data[pos+2]
		pos += 3

		isUnicode := (flags & 0x01) != 0
		hasRichText := (flags & 0x08) != 0
		hasExtString := (flags & 0x04) != 0

		cRun := 0
		if hasRichText && pos+2 <= len(data) {
			cRun = int(binary.LittleEndian.Uint16(data[pos : pos+2]))
			pos += 2
		}
		cbExt := uint32(0)
		if hasExtString && pos+4 <= len(data) {
			cbExt = binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += 4
		}

		var strVal string
		if isUnicode {
			byteLen := cch * 2
			if pos+byteLen <= len(data) {
				u16 := make([]uint16, cch)
				for j := 0; j < cch; j++ {
					u16[j] = binary.LittleEndian.Uint16(data[pos+j*2 : pos+j*2+2])
				}
				strVal = string(utf16.Decode(u16))
				pos += byteLen
			}
		} else {
			if pos+cch <= len(data) {
				strVal = string(data[pos : pos+cch])
				pos += cch
			}
		}

		pos += cRun * 4
		pos += int(cbExt)

		stringsList = append(stringsList, strings.TrimSpace(strVal))
	}
	return stringsList
}

func decodeRK(rk uint32) float64 {
	var num float64
	if (rk & 0x02) != 0 {
		num = float64(int32(rk) >> 2)
	} else {
		var buf bytes.Buffer
		_ = binary.Write(&buf, binary.LittleEndian, uint32(0))
		_ = binary.Write(&buf, binary.LittleEndian, rk&0xFFFFFFFC)
		_ = binary.Read(&buf, binary.LittleEndian, &num)
	}
	if (rk & 0x01) != 0 {
		num /= 100.0
	}
	return num
}

func renderSpreadsheetTableHTML(rows [][]string) string {
	if len(rows) == 0 {
		return "<article><p>No spreadsheet data found.</p></article>"
	}

	var out strings.Builder
	out.WriteString(`<article class="novelhub-spreadsheet">`)
	out.WriteString(`<div class="novelhub-table-wrapper" style="overflow-x: auto; max-width: 100%; margin: 1em 0;">`)
	out.WriteString(`<table class="novelhub-table" style="width: 100%; border-collapse: collapse; text-align: left; font-size: 0.9em; line-height: 1.5;">`)

	renderLimit := len(rows)
	if renderLimit > 1000 {
		renderLimit = 1000
	}

	out.WriteString("<thead><tr>")
	for _, cell := range rows[0] {
		out.WriteString(`<th style="border: 1px solid rgba(128,128,128,0.3); padding: 6px 10px; font-weight: bold; background: rgba(128,128,128,0.08);">`)
		out.WriteString(html.EscapeString(strings.TrimSpace(cell)))
		out.WriteString("</th>")
	}
	out.WriteString("</tr></thead>")

	out.WriteString("<tbody>")
	for i := 1; i < renderLimit; i++ {
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
	out.WriteString("</table></div>")

	out.WriteString("</article>")
	return out.String()
}

func readCompoundStreams(filePath string) (map[string][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open compound file: %w", err)
	}
	defer file.Close()

	reader, err := mscfb.New(file)
	if err != nil {
		return nil, fmt.Errorf("open compound reader: %w", err)
	}
	streams := make(map[string][]byte)
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
