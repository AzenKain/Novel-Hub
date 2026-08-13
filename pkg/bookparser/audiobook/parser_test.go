package audiobook

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func flacMetadataBlock(blockType byte, last bool, payload []byte) []byte {
	header := blockType
	if last {
		header |= 0x80
	}
	out := make([]byte, 4, 4+len(payload))
	out[0] = header
	binary.BigEndian.PutUint32(out[1:5], uint32(len(payload)))
	return append(out, payload...)
}

func streamInfoBlock() []byte {
	payload := make([]byte, 34)
	binary.BigEndian.PutUint16(payload[0:2], 4096)
	binary.BigEndian.PutUint16(payload[2:4], 4096)
	payload[18] = 0x0A
	payload[19] = 0x84
	payload[20] = 0x40
	payload[21] = 0xF0
	binary.BigEndian.PutUint32(payload[22:26], 44100)
	return flacMetadataBlock(0, true, payload)
}

func pictureBlock(data []byte) []byte {
	var p bytes.Buffer
	mime := "image/png"
	binary.Write(&p, binary.BigEndian, uint32(3))
	binary.Write(&p, binary.BigEndian, uint32(len(mime)))
	p.WriteString(mime)
	binary.Write(&p, binary.BigEndian, uint32(0))
	binary.Write(&p, binary.BigEndian, uint32(0))
	binary.Write(&p, binary.BigEndian, uint32(0))
	binary.Write(&p, binary.BigEndian, uint32(0))
	binary.Write(&p, binary.BigEndian, uint32(0))
	binary.Write(&p, binary.BigEndian, uint32(len(data)))
	p.Write(data)
	return flacMetadataBlock(6, false, p.Bytes())
}

func writeFLAC(t *testing.T, cover []byte) string {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("fLaC")
	if cover != nil {
		b.Write(pictureBlock(cover))
	}
	b.Write(streamInfoBlock())
	path := filepath.Join(t.TempDir(), "book.flac")
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatalf("write flac: %v", err)
	}
	return path
}

func TestAudiobookCoverFallback(t *testing.T) {
	path := writeFLAC(t, nil)
	meta, err := New().ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.CoverType != "image/svg+xml" {
		t.Errorf("cover type = %q, want image/svg+xml", meta.CoverType)
	}
	if len(meta.CoverData) == 0 || !strings.Contains(string(meta.CoverData), "<svg") {
		t.Errorf("fallback cover data = %q, want svg", meta.CoverData)
	}
}

func TestAudiobookKeepsEmbeddedCover(t *testing.T) {
	pic := []byte("\x89PNG\r\n\x1a\nfake-png-bytes")
	path := writeFLAC(t, pic)
	meta, err := New().ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.CoverType != "image/png" {
		t.Errorf("cover type = %q, want image/png", meta.CoverType)
	}
	if !bytes.Equal(meta.CoverData, pic) {
		t.Errorf("cover data = %v, want embedded picture", meta.CoverData)
	}
}

func TestAudiobookUnreadableFileFallsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.m4a")
	if err := os.WriteFile(path, []byte("not audio at all"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	meta, err := New().ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.Title == "" {
		t.Error("title empty for unreadable file")
	}
	if meta.CoverType != "image/svg+xml" || len(meta.CoverData) == 0 {
		t.Errorf("expected svg fallback cover, got type=%q data=%d bytes", meta.CoverType, len(meta.CoverData))
	}
}