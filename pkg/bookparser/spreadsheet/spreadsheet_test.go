package spreadsheet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpreadsheetParserWithRealFiles(t *testing.T) {
	parser := NewParser()

	xlsxPath := filepath.Join("..", "..", "..", "sample_file", "file_example_XLSX_5000.xlsx")
	if _, err := os.Stat(xlsxPath); err == nil {
		meta, err := parser.ParseMetadata(xlsxPath)
		if err != nil {
			t.Fatalf("XLSX ParseMetadata: %v", err)
		}
		if meta.Title == "" {
			t.Errorf("expected XLSX title")
		}

		spine, err := parser.ParseSpine(xlsxPath)
		if err != nil || len(spine) == 0 {
			t.Fatalf("XLSX ParseSpine: %v (len %d)", err, len(spine))
		}

		sheet1, err := parser.GetChapterContent(xlsxPath, spine[0].ContentPath)
		if err != nil {
			t.Fatalf("XLSX GetChapterContent: %v", err)
		}
		if !strings.Contains(sheet1, "<table") {
			t.Errorf("expected table in sheet content, got: %s", sheet1)
		}
	}

	odsPath := filepath.Join("..", "..", "..", "sample_file", "file_example_ODS_5000.ods")
	if _, err := os.Stat(odsPath); err == nil {
		meta, err := parser.ParseMetadata(odsPath)
		if err != nil {
			t.Fatalf("ODS ParseMetadata: %v", err)
		}
		if meta.Title == "" {
			t.Errorf("expected ODS title")
		}

		spine, err := parser.ParseSpine(odsPath)
		if err != nil || len(spine) == 0 {
			t.Fatalf("ODS ParseSpine: %v (len %d)", err, len(spine))
		}

		sheet1, err := parser.GetChapterContent(odsPath, spine[0].ContentPath)
		if err != nil {
			t.Fatalf("ODS GetChapterContent: %v", err)
		}
		if !strings.Contains(sheet1, "<table") {
			t.Errorf("expected table in ods content, got: %s", sheet1)
		}
	}

	xlsPath := filepath.Join("..", "..", "..", "sample_file", "file_example_XLS_5000.xls")
	if _, err := os.Stat(xlsPath); err == nil {
		meta, err := parser.ParseMetadata(xlsPath)
		if err != nil {
			t.Fatalf("XLS ParseMetadata: %v", err)
		}
		if meta.Title == "" {
			t.Errorf("expected XLS title")
		}

		spine, err := parser.ParseSpine(xlsPath)
		if err != nil || len(spine) == 0 {
			t.Fatalf("XLS ParseSpine: %v (len %d)", err, len(spine))
		}

		sheet1, err := parser.GetChapterContent(xlsPath, spine[0].ContentPath)
		if err != nil {
			t.Fatalf("XLS GetChapterContent: %v", err)
		}
		if !strings.Contains(sheet1, "<table") {
			t.Errorf("expected table in xls content, got: %s", sheet1)
		}
	}
}
