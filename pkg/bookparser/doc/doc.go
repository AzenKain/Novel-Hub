package doc

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
	"golang.org/x/text/encoding/charmap"

	"novelhub/pkg/bookparser"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	text, err := extractDocText(filePath)
	if err != nil {
		return bookparser.MergeMetadataSidecar(filePath, &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}), nil
	}
	return bookparser.MergeMetadataSidecar(filePath, &bookparser.BookMetadata{
		Title:       bookparser.TitleFromPath(filePath),
		Description: preview(text),
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
	text, err := extractDocText(filePath)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return `<article><p>No readable text was found in this DOC file.</p></article>`, nil
	}
	return bookparser.PlainTextToHTML(text), nil
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
		data, err := io.ReadAll(entry)
		if err != nil {
			continue
		}
		streams[entry.Name] = data
	}
	return streams, nil
}

func extractDocText(filePath string) (string, error) {
	streams, err := readDocStreams(filePath)
	if err != nil {
		return "", err
	}
	word := streams["WordDocument"]
	if len(word) == 0 {
		return "", fmt.Errorf("WordDocument stream not found")
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
	text, err := extractWordTextFromStreams(word, table)
	if err != nil {
		return "", err
	}
	return cleanWordText(text), nil
}

func extractWordTextFromStreams(word []byte, table []byte) (string, error) {
	if len(word) < 0x01aa {
		return "", fmt.Errorf("WordDocument stream too small")
	}
	fcClx := binary.LittleEndian.Uint32(word[0x01a2:0x01a6])
	lcbClx := binary.LittleEndian.Uint32(word[0x01a6:0x01aa])
	if len(table) > 0 && lcbClx > 0 && int(fcClx+lcbClx) <= len(table) {
		if text, err := extractPieceTableText(word, table[fcClx:fcClx+lcbClx]); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
	}
	return extractSimpleTextRange(word)
}

func extractPieceTableText(word []byte, clx []byte) (string, error) {
	pieceTable, err := findPieceTable(clx)
	if err != nil {
		return "", err
	}
	if len(pieceTable) < 16 || (len(pieceTable)-4)%12 != 0 {
		return "", fmt.Errorf("invalid DOC piece table")
	}
	pieceCount := (len(pieceTable) - 4) / 12
	cpOffset := 0
	pcdOffset := (pieceCount + 1) * 4
	var out strings.Builder
	for i := 0; i < pieceCount; i++ {
		cpStart := binary.LittleEndian.Uint32(pieceTable[cpOffset+i*4 : cpOffset+i*4+4])
		cpEnd := binary.LittleEndian.Uint32(pieceTable[cpOffset+(i+1)*4 : cpOffset+(i+1)*4+4])
		if cpEnd <= cpStart {
			continue
		}
		pcd := pieceTable[pcdOffset+i*8 : pcdOffset+i*8+8]
		fcRaw := binary.LittleEndian.Uint32(pcd[2:6])
		charCount := int(cpEnd - cpStart)
		if fcRaw&0x40000000 != 0 {
			offset := int((fcRaw &^ 0x40000000) / 2)
			if offset < 0 || offset >= len(word) {
				continue
			}
			end := offset + charCount
			if end > len(word) {
				end = len(word)
			}
			out.WriteString(decodeWindows1252(word[offset:end]))
		} else {
			offset := int(fcRaw)
			if offset < 0 || offset >= len(word) {
				continue
			}
			end := offset + charCount*2
			if end > len(word) {
				end = len(word)
			}
			out.WriteString(decodeUTF16LE(word[offset:end]))
		}
		out.WriteString("\n")
	}
	return out.String(), nil
}

func findPieceTable(clx []byte) ([]byte, error) {
	for i := 0; i < len(clx); {
		switch clx[i] {
		case 0x01:
			if i+3 > len(clx) {
				return nil, fmt.Errorf("truncated DOC Prc block")
			}
			size := int(binary.LittleEndian.Uint16(clx[i+1 : i+3]))
			i += 3 + size
		case 0x02:
			if i+5 > len(clx) {
				return nil, fmt.Errorf("truncated DOC piece table header")
			}
			size := int(binary.LittleEndian.Uint32(clx[i+1 : i+5]))
			start := i + 5
			end := start + size
			if size < 0 || end > len(clx) {
				return nil, fmt.Errorf("truncated DOC piece table")
			}
			return clx[start:end], nil
		default:
			i++
		}
	}
	return nil, fmt.Errorf("DOC piece table not found")
}

func extractSimpleTextRange(word []byte) (string, error) {
	if len(word) < 0x20 {
		return "", fmt.Errorf("WordDocument stream too small")
	}
	fcMin := int(binary.LittleEndian.Uint32(word[0x18:0x1c]))
	fcMac := int(binary.LittleEndian.Uint32(word[0x1c:0x20]))
	if fcMin < 0 || fcMac <= fcMin || fcMac > len(word) {
		return decodeBestEffort(word), nil
	}
	return decodeBestEffort(word[fcMin:fcMac]), nil
}

func decodeBestEffort(data []byte) string {
	utf16Text := decodeUTF16LE(data)
	ansiText := decodeWindows1252(data)
	if readableScore(utf16Text) > readableScore(ansiText) {
		return utf16Text
	}
	return ansiText
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 == 1 {
		data = data[:len(data)-1]
	}
	values := make([]uint16, len(data)/2)
	for i := range values {
		values[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
	}
	return string(utf16.Decode(values))
}

func decodeWindows1252(data []byte) string {
	var out strings.Builder
	for _, b := range data {
		out.WriteRune(charmap.Windows1252.DecodeByte(b))
	}
	return out.String()
}

func cleanWordText(value string) string {
	var out strings.Builder
	previousNewline := false
	for _, r := range value {
		switch r {
		case 0, '\u0001', '\u0002', '\u0003', '\u0004', '\u0005', '\u0006', '\u0007', '\u0008', '\u0013', '\u0014', '\u0015':
			continue
		case '\r', '\n', '\v', '\f':
			if !previousNewline {
				out.WriteByte('\n')
				previousNewline = true
			}
		case '\t':
			out.WriteByte('\t')
			previousNewline = false
		default:
			if unicode.IsControl(r) {
				continue
			}
			out.WriteRune(r)
			previousNewline = false
		}
	}
	lines := strings.Split(out.String(), "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = bookparser.CleanOfficeTextLine(line)
		if line == "" {
			if !blank {
				cleaned = append(cleaned, "")
			}
			blank = true
			continue
		}
		cleaned = append(cleaned, line)
		blank = false
	}
	paragraphs := make([]string, 0, len(cleaned))
	for _, line := range cleaned {
		if line == "" {
			continue
		}
		paragraphs = append(paragraphs, line)
	}
	return strings.TrimSpace(strings.Join(paragraphs, "\n\n"))
}

func readableScore(value string) int {
	score := 0
	for _, r := range value {
		if r == '\n' || r == '\t' || unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsPunct(r) || unicode.IsSpace(r) {
			score++
		}
	}
	return score
}

func preview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return strings.TrimSpace(text[:500]) + "..."
	}
	return text
}
