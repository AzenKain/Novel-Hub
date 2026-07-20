package archivebook

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserReadsZippedMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.zip")
	writeZip(t, path, map[string]string{
		"book.md": "# Zipped Title\n\nHello from archive.",
	})

	parser := NewParser("zip")
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "Zipped Title" {
		t.Fatalf("unexpected title %q", meta.Title)
	}

	html, err := parser.GetChapterContent(path, "book.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<h1>Zipped Title</h1>") {
		t.Fatalf("unexpected html: %s", html)
	}
}

func TestParserReadsFBZ(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.fbz")
	writeZip(t, path, map[string]string{
		"book.fb2": `<?xml version="1.0"?><FictionBook><description><title-info><book-title>FBZ Book</book-title></title-info></description><body><section><title><p>One</p></title><p>Hello one.</p></section><section><title><p>Two</p></title><p>Hello two.</p></section></body></FictionBook>`,
	})

	parser := NewParser("fbz")
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "FBZ Book" {
		t.Fatalf("unexpected title %q", meta.Title)
	}
	chapters, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 {
		t.Fatalf("chapters = %d, want 2: %#v", len(chapters), chapters)
	}
	html, err := parser.GetChapterContent(path, chapters[1].ContentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hello two.") || strings.Contains(html, "Hello one.") {
		t.Fatalf("unexpected html: %s", html)
	}
}

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	defer writer.Close()
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}
