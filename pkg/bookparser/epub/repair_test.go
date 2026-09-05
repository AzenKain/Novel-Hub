package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairXHTMLEntities(t *testing.T) {
	input := `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>A&nbsp;B &copy; C & unknown &madeup;</p></body></html>`
	fixed, count := repairXHTMLEntities(input)
	if count != 4 {
		t.Fatalf("count = %d, want 4; fixed=%s", count, fixed)
	}
	if strings.Contains(fixed, "&nbsp;") {
		t.Fatalf("fixed still contains &nbsp;: %s", fixed)
	}
	if !strings.Contains(fixed, "A&#160;B") {
		t.Fatalf("fixed missing numeric nbsp replacement: %s", fixed)
	}
	if !strings.Contains(fixed, "&amp; unknown") {
		t.Fatalf("fixed missing bare ampersand escape: %s", fixed)
	}
	if !strings.Contains(fixed, "&amp;madeup;") {
		t.Fatalf("fixed missing unknown entity escape: %s", fixed)
	}
	if err := validateXMLWellFormed(fixed); err != nil {
		t.Fatalf("fixed XML is not well-formed: %v\n%s", err, fixed)
	}
}

func TestIsChapterTitleMissing(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "No heading at all",
			content: `<html><body><p>Hello world</p></body></html>`,
			want:    true,
		},
		{
			name:    "Heading at the beginning",
			content: `<html><body><h2>Chapter Title</h2><p>Hello world</p></body></html>`,
			want:    false,
		},
		{
			name:    "Heading at the beginning with spaces and comments",
			content: `<html><body>  <!-- comment -->  <h2>Chapter Title</h2><p>Hello world</p></body></html>`,
			want:    false,
		},
		{
			name:    "Heading at the end (notes case)",
			content: `<html><body><p>Hello world</p><h2>Ghi chú</h2></body></html>`,
			want:    true,
		},
		{
			name:    "Heading at the end with HTML entities",
			content: `<html><body><p>&nbsp;Hello&nbsp;world&nbsp;</p><h2>Ghi chú</h2></body></html>`,
			want:    true,
		},
		{
			name:    "Fragment - no heading",
			content: `<p>Hello world</p>`,
			want:    true,
		},
		{
			name:    "Fragment - heading at beginning",
			content: `<h2>Chapter Title</h2><p>Hello world</p>`,
			want:    false,
		},
		{
			name:    "Fragment - heading at end",
			content: `<p>Hello world</p><h2>Notes</h2>`,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isChapterTitleMissing(tt.content)
			if got != tt.want {
				t.Errorf("isChapterTitleMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateXMLWellFormed(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "Well-formed XML",
			content: `<root><child>text</child></root>`,
			wantErr: false,
		},
		{
			name:    "Malformed - mismatched tag",
			content: `<root><child>text</root></child>`,
			wantErr: true,
		},
		{
			name:    "Malformed - missing close tag",
			content: `<root><child>text</child>`,
			wantErr: true,
		},
		{
			name:    "Malformed - XML syntax error (navMap closed by navPoint)",
			content: `<ncx><navMap><navPoint></navMap></navPoint></ncx>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateXMLWellFormed(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateXMLWellFormed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepairFixForIssue(t *testing.T) {
	tests := []struct {
		code      string
		wantFixID string
		wantOk    bool
	}{
		{"MANIFEST_ORPHAN_DOCUMENT", "REMOVE_MISSING_MANIFEST_ITEMS", true},
		{"NCX_DUMMY_DUPLICATE_LINK", "FIX_TOC_NCX", true},
		{"NCX_TARGET_NOT_IN_SPINE", "FIX_TOC_NCX", true},
		{"TOC_NAV_NCX_MISMATCH", "FIX_TOC_NCX", true},
		{"MANIFEST_FILE_MISSING", "REMOVE_MISSING_MANIFEST_ITEMS", true},
		{"CHAPTER_TITLE_MISSING", "BUILD_CHAPTER_TITLES", true},
		{"UNKNOWN_CODE", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			fixID, ok := RepairFixForIssue(tt.code)
			if ok != tt.wantOk || fixID != tt.wantFixID {
				t.Errorf("repairFixForIssue(%q) = (%q, %v), want (%q, %v)", tt.code, fixID, ok, tt.wantFixID, tt.wantOk)
			}
		})
	}
}

func TestValidateAndRepairEPUB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "epub-doctor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	corruptEPUBPath := filepath.Join(tempDir, "corrupted.epub")
	repairedEPUBPath := filepath.Join(tempDir, "repaired.epub")

	f, err := os.Create(corruptEPUBPath)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	zw := zip.NewWriter(f)

	w1, _ := zw.Create("META-INF/container.xml")
	_, _ = w1.Write([]byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	w2, _ := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Deflate})
	_, _ = w2.Write([]byte("application/epub+zip"))

	w3, _ := zw.Create("OEBPS/content.opf")
	_, _ = w3.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Corrupted Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">test-12345</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="text/html" />
    <item id="missing_item" href="missing.xhtml" media-type="application/xhtml+xml" />
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1" />
    <itemref idref="missing_item" />
  </spine>
</package>`))

	w4, _ := zw.Create("OEBPS/ch1.xhtml")
	_, _ = w4.Write([]byte(`<html>
<head><title>Chapter 1</title></head>
<body>
  <p>First line<br>Second line &copy; Kain & Antigravity</p>
  <img src="missing_image.jpg">
  <a href="dead_link.xhtml">Dead link text</a>
</body>
</html>`))

	w5, _ := zw.Create("OEBPS/extra_style.css")
	_, _ = w5.Write([]byte(`body { font-family: sans-serif; }`))

	_ = zw.Close()
	_ = f.Close()

	report, err := ValidateEPUB(corruptEPUBPath)
	if err != nil {
		t.Fatalf("ValidateEPUB failed: %v", err)
	}

	if report.Valid {
		t.Fatal("expected report to be invalid")
	}
	if report.Errors == 0 {
		t.Fatalf("expected errors in corrupted epub, got 0. Issues: %#v", report.Issues)
	}

	opts := DefaultRepairOptions()
	result, err := RepairEPUB(corruptEPUBPath, repairedEPUBPath, opts)
	if err != nil {
		t.Fatalf("RepairEPUB failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected repair to succeed, logs: %v", result.Logs)
	}

	postReport, err := ValidateEPUB(repairedEPUBPath)
	if err != nil {
		t.Fatalf("post-repair ValidateEPUB failed: %v", err)
	}

	if !postReport.Valid || postReport.Errors > 0 {
		t.Fatalf("expected repaired EPUB to be 100%% valid with 0 errors, got: %d errors, issues: %#v", postReport.Errors, postReport.Issues)
	}

	rRepaired, err := zip.OpenReader(repairedEPUBPath)
	if err != nil {
		t.Fatalf("failed to open repaired epub: %v", err)
	}
	defer rRepaired.Close()

	if len(rRepaired.File) == 0 || rRepaired.File[0].Name != "mimetype" {
		t.Fatal("repaired epub does not have mimetype as first entry")
	}
	if rRepaired.File[0].Method != zip.Store {
		t.Fatal("repaired epub mimetype is compressed")
	}
}
