package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDOCXParser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.docx")
	if err := writeMinimalDOCX(path); err != nil {
		t.Fatalf("write docx: %v", err)
	}

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.Title != "Doc Title" {
		t.Fatalf("title = %q, want Doc Title", meta.Title)
	}

	html, err := parser.GetChapterContent(path, "word/document.xml")
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}
	if !strings.Contains(html, "First paragraph") || !strings.Contains(html, "Second paragraph") {
		t.Fatalf("expected paragraphs in html, got %s", html)
	}

	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(images) != 1 || images[0] != "word/media/image1.png" {
		t.Fatalf("unexpected images: %#v", images)
	}
	asset, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset: %v", err)
	}
	if string(asset) != "png-data" {
		t.Fatalf("unexpected asset: %q", string(asset))
	}
}

func TestExtractDocumentTextNormalizesLists(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:numPr><w:ilvl w:val="0"/></w:numPr></w:pPr><w:r><w:t>Maecenas non lorem</w:t></w:r></w:p>
    <w:p><w:r><w:t>□? Nulla facilisi.</w:t></w:r></w:p>
  </w:body>
</w:document>`
	text, err := extractDocumentText(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("extractDocumentText failed: %v", err)
	}
	if strings.Contains(text, "□?") {
		t.Fatalf("expected malformed bullet to be removed, got %q", text)
	}
	if !strings.Contains(text, "• Maecenas non lorem") || !strings.Contains(text, "• Nulla facilisi.") {
		t.Fatalf("expected normalized bullet paragraphs, got %q", text)
	}
}

func writeMinimalDOCX(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	files := map[string]string{
		"docProps/core.xml": `<?xml version="1.0" encoding="UTF-8"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/">
  <dc:title>Doc Title</dc:title>
  <dc:creator>Doc Author</dc:creator>
</cp:coreProperties>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>First paragraph</w:t></w:r></w:p>
    <w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p>
  </w:body>
</w:document>`,
		"word/media/image1.png": "png-data",
	}
	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}
