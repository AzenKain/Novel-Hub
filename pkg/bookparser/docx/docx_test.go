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

func TestDOCXParserWithImagesAndStyling(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styled.docx")

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create docx: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	
	files := map[string]string{
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rIdImage1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.png"/>
</Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r>
        <w:rPr><w:b/></w:rPr>
        <w:t>Header Text</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:rPr><w:i/></w:rPr>
        <w:t>Italic run</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:u w:val="single"/></w:rPr>
        <w:t>Underline run</w:t>
      </w:r>
      <w:r>
        <w:drawing>
          <wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing">
            <a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
              <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">
                <pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
                  <pic:blipFill>
                    <a:blip r:embed="rIdImage1"/>
                  </pic:blipFill>
                </pic:pic>
              </a:graphicData>
            </a:graphic>
          </wp:inline>
        </w:drawing>
      </w:r>
    </w:p>
  </w:body>
</w:document>`,
		"word/media/image1.png": "png-data-here",
	}

	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create entry %s: %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("write entry %s: %v", name, err)
		}
	}
	zw.Close()

	parser := NewParser()
	html, err := parser.GetChapterContent(path, "word/document.xml")
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}

	// Heading tag check
	if !strings.Contains(html, "<h1><b>Header Text</b></h1>") {
		t.Errorf("expected Heading1 structure in HTML, got %q", html)
	}

	// Italic and underline check
	if !strings.Contains(html, "<i>Italic run</i>") || !strings.Contains(html, "<u>Underline run</u>") {
		t.Errorf("expected formatted text runs, got %q", html)
	}

	// Image extraction check
	if !strings.Contains(html, `<img src="media/image1.png" />`) {
		t.Errorf("expected img tag with relative path in HTML, got %q", html)
	}
}

func TestDOCXParserAlignmentAndAdvancedTypography(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "advanced_typography.docx")

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create docx: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	files := map[string]string{
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:jc w:val="center"/></w:pPr>
      <w:r><w:t>Centered Paragraph</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:jc w:val="right"/></w:pPr>
      <w:r><w:t>Right Paragraph</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:jc w:val="both"/></w:pPr>
      <w:r><w:t>Justified Paragraph</w:t></w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:rPr><w:strike/></w:rPr>
        <w:t>Strikethrough text</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:caps/></w:rPr>
        <w:t>Uppercase text</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:smallCaps/></w:rPr>
        <w:t>Smallcaps text</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`,
	}

	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create entry %s: %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("write entry %s: %v", name, err)
		}
	}
	zw.Close()

	parser := NewParser()
	html, err := parser.GetChapterContent(path, "word/document.xml")
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}

	if !strings.Contains(html, `<p align="center">Centered Paragraph</p>`) {
		t.Errorf("expected centered paragraph, got %q", html)
	}
	if !strings.Contains(html, `<p align="right">Right Paragraph</p>`) {
		t.Errorf("expected right-aligned paragraph, got %q", html)
	}
	if !strings.Contains(html, `<p align="justify">Justified Paragraph</p>`) {
		t.Errorf("expected justified paragraph, got %q", html)
	}
	if !strings.Contains(html, `<s>Strikethrough text</s>`) {
		t.Errorf("expected strikethrough text, got %q", html)
	}
	if !strings.Contains(html, `<span class="uppercase">Uppercase text</span>`) {
		t.Errorf("expected uppercase text, got %q", html)
	}
	if !strings.Contains(html, `<span class="small-caps">Smallcaps text</span>`) {
		t.Errorf("expected smallcaps text, got %q", html)
	}
}
