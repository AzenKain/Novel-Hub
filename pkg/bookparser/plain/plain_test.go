package plain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownParser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.md")
	if err := os.WriteFile(filepath.Join(dir, "cover.png"), []byte("image-data"), 0600); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := os.WriteFile(path, []byte("# My Title\n\n![Cover](cover.png)\n\nHello **world**"), 0600); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.Title != "My Title" {
		t.Fatalf("title = %q, want My Title", meta.Title)
	}
	html, err := parser.GetChapterContent(path, "document")
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}
	if !strings.Contains(html, "<h1>My Title</h1>") {
		t.Fatalf("expected markdown heading in html, got %s", html)
	}
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(images) != 1 || images[0] != "cover.png" {
		t.Fatalf("unexpected images: %#v", images)
	}
	asset, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset: %v", err)
	}
	if string(asset) != "image-data" {
		t.Fatalf("unexpected asset: %q", string(asset))
	}
}
