package doc

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"

	"novelhub/pkg/bookparser"
)

func extractPieceTable(word, table []byte) ([]textPiece, error) {
	if len(word) < 0x01aa {
		return nil, fmt.Errorf("WordDocument stream too small")
	}
	fcClx := binary.LittleEndian.Uint32(word[0x01a2:0x01a6])
	lcbClx := binary.LittleEndian.Uint32(word[0x01a6:0x01aa])
	if len(table) == 0 || lcbClx == 0 || int(fcClx+lcbClx) > len(table) {
		return nil, fmt.Errorf("no CLX data")
	}
	clx := table[fcClx : fcClx+lcbClx]
	pieceTable, err := findPieceTable(clx)
	if err != nil {
		return nil, err
	}
	if len(pieceTable) < 16 || (len(pieceTable)-4)%12 != 0 {
		return nil, fmt.Errorf("invalid DOC piece table")
	}
	pieceCount := (len(pieceTable) - 4) / 12
	var pieces []textPiece
	for i := 0; i < pieceCount; i++ {
		cpStart := int(binary.LittleEndian.Uint32(pieceTable[i*4 : i*4+4]))
		cpEnd := int(binary.LittleEndian.Uint32(pieceTable[(i+1)*4 : (i+1)*4+4]))
		if cpEnd <= cpStart {
			continue
		}
		pcd := pieceTable[(pieceCount+1)*4+i*8 : (pieceCount+1)*4+i*8+8]
		fcRaw := binary.LittleEndian.Uint32(pcd[2:6])
		charCount := cpEnd - cpStart
		var text string
		if fcRaw&0x40000000 != 0 {
			offset := int((fcRaw &^ 0x40000000) / 2)
			end := offset + charCount
			if offset >= 0 && end <= len(word) {
				text = decodeWindows1252(word[offset:end])
			}
		} else {
			offset := int(fcRaw)
			end := offset + charCount*2
			if offset >= 0 && end <= len(word) {
				text = decodeUTF16LE(word[offset:end])
			}
		}
		if text != "" {
			pieces = append(pieces, textPiece{text: text, cpStart: cpStart})
		}
	}
	return pieces, nil
}

func extractDocText(filePath string) (string, error) {
	streams, err := readDocStreams(filePath)
	if err != nil {
		return "", err
	}
	word, table := selectTableStream(streams)
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
	var out strings.Builder
	for i := 0; i < pieceCount; i++ {
		cpStart := binary.LittleEndian.Uint32(pieceTable[i*4 : i*4+4])
		cpEnd := binary.LittleEndian.Uint32(pieceTable[(i+1)*4 : (i+1)*4+4])
		if cpEnd <= cpStart {
			continue
		}
		pcd := pieceTable[(pieceCount+1)*4+i*8 : (pieceCount+1)*4+i*8+8]
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
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 19, 20, 21:
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
	var paragraphs []string
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
