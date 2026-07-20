package htmlfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserReadsHTMLMetadataAndContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.html")
	err := os.WriteFile(path, []byte(`<!doctype html><html><head><title>HTML Book</title><meta name="description" content="Short description"></head><body><h1>Hello</h1><img src="cover.jpg"></body></html>`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "HTML Book" || meta.Description != "Short description" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}

	html, err := parser.GetChapterContent(path, "book.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<h1>Hello</h1>") {
		t.Fatalf("expected html body, got %q", html)
	}

	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || images[0] != "cover.jpg" {
		t.Fatalf("unexpected images: %#v", images)
	}
}
