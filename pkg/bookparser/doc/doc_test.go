package doc

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestExtractWordTextFromPieceTable(t *testing.T) {
	text := "Hello DOC"
	word := make([]byte, 0x300)
	binary.LittleEndian.PutUint32(word[0x01a2:0x01a6], 0)
	encoded := utf16.Encode([]rune(text))
	textOffset := 0x0200
	for i, value := range encoded {
		binary.LittleEndian.PutUint16(word[textOffset+i*2:textOffset+i*2+2], value)
	}

	pieceTable := make([]byte, 16)
	binary.LittleEndian.PutUint32(pieceTable[0:4], 0)
	binary.LittleEndian.PutUint32(pieceTable[4:8], uint32(len([]rune(text))))
	binary.LittleEndian.PutUint32(pieceTable[10:14], uint32(textOffset))
	table := make([]byte, 5+len(pieceTable))
	table[0] = 0x02
	binary.LittleEndian.PutUint32(table[1:5], uint32(len(pieceTable)))
	copy(table[5:], pieceTable)
	binary.LittleEndian.PutUint32(word[0x01a6:0x01aa], uint32(len(table)))

	got, err := extractWordTextFromStreams(word, table)
	if err != nil {
		t.Fatalf("extractWordTextFromStreams failed: %v", err)
	}
	if !strings.Contains(got, text) {
		t.Fatalf("text = %q, want it to contain %q", got, text)
	}
}

func TestCleanWordTextNormalizesMalformedOfficeBullets(t *testing.T) {
	input := "Intro\r□? Maecenas non lorem\r\uf0b7 Nulla facilisi\rNormal line"
	got := cleanWordText(input)
	if strings.Contains(got, "□?") || strings.Contains(got, "\uf0b7") {
		t.Fatalf("expected malformed bullet glyphs to be normalized, got %q", got)
	}
	if !strings.Contains(got, "• Maecenas non lorem") || !strings.Contains(got, "• Nulla facilisi") {
		t.Fatalf("expected bullet lines, got %q", got)
	}
	if !strings.Contains(got, "Intro\n\n• Maecenas") {
		t.Fatalf("expected paragraph spacing, got %q", got)
	}
}
