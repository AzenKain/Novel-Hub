package rtf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserExtractsRTFText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.rtf")
	content := `{\rtf1\ansi{\fonttbl{\f0 Arial;}}Hello \b world\b0\par Unicode: \u10019?\par Hex: caf\'e9}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	html, err := NewParser().GetChapterContent(path, "document")
	if err != nil {
		t.Fatalf("GetChapterContent failed: %v", err)
	}
	for _, want := range []string{"Hello <b>world</b>", "Unicode: ✣", "Hex: café"} {
		if !strings.Contains(html, want) {
			t.Fatalf("html does not contain %q: %s", want, html)
		}
	}
}

func TestParserListsAndReadsRTFPictAssets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.rtf")
	content := `{\rtf1{\pict\pngblip
89504E470D0A1A0A0000000D49484452
}}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}
	if len(images) != 1 || images[0] != "images/pict-001.png" {
		t.Fatalf("unexpected images: %#v", images)
	}
	data, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("unexpected asset data: %x", data)
	}
}
