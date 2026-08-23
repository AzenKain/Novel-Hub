package csv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVParser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")

	content := `Name,Age,Role
Alice,28,Engineer
Bob,32,Designer
"Charlie, Jr.",45,Manager`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.Title != "data" {
		t.Fatalf("title = %q, want data", meta.Title)
	}

	spine, err := parser.ParseSpine(path)
	if err != nil || len(spine) != 1 {
		t.Fatalf("ParseSpine: %v (len: %d)", err, len(spine))
	}

	htmlContent, err := parser.GetChapterContent(path, spine[0].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}

	if !strings.Contains(htmlContent, "Alice") || !strings.Contains(htmlContent, "Charlie, Jr.") {
		t.Errorf("expected cell values in table, got: %s", htmlContent)
	}
	if !strings.Contains(htmlContent, "<table") || !strings.Contains(htmlContent, "Name</th>") {
		t.Errorf("expected table structure with headers, got: %s", htmlContent)
	}
}
