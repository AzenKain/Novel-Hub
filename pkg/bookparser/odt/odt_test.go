package odt

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserReadsODTContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.odt")
	writeODT(t, path, map[string]string{
		"meta.xml":    `<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:dc="http://purl.org/dc/elements/1.1/"><office:meta><dc:title>ODT Book</dc:title><dc:creator>Writer</dc:creator></office:meta></office:document-meta>`,
		"content.xml": `<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"><office:body><office:text><text:h text:outline-level="1">Chapter</text:h><text:p>Hello <text:s text:c="2"/>ODT</text:p></office:text></office:body></office:document-content>`,
	})

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "ODT Book" || meta.Author != "Writer" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}

	html, err := parser.GetChapterContent(path, "content.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<h1>Chapter</h1>") || !strings.Contains(html, "Hello   ODT") {
		t.Fatalf("unexpected html: %s", html)
	}
}

func writeODT(t *testing.T, path string, entries map[string]string) {
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
