package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"novelhub/pkg/bookparser"
)

func TestParseMetadataUsesPropertyCover(t *testing.T) {
	path := writeEPUBFixture(t, `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns:dc="http://purl.org/dc/elements/1.1/">
  <metadata>
    <dc:title>Property Cover</dc:title>
    <meta property="cover">cover-image</meta>
  </metadata>
  <manifest>
    <item id="cover-image" href="Images/cover.jpg" media-type="image/jpeg"/>
    <item id="chapter" href="Text/chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="chapter"/></spine>
</package>`)

	meta, err := NewParser().ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata returned error: %v", err)
	}
	if meta.Title != "Property Cover" {
		t.Fatalf("title = %q, want Property Cover", meta.Title)
	}
	if string(meta.CoverData) != "jpeg-cover" || meta.CoverType != "image/jpeg" {
		t.Fatalf("unexpected cover data/type: %q %q", string(meta.CoverData), meta.CoverType)
	}
}

func TestParseMetadataFallsBackToFirstImageCover(t *testing.T) {
	path := writeEPUBFixture(t, `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns:dc="http://purl.org/dc/elements/1.1/">
  <metadata><dc:title>Image Fallback</dc:title></metadata>
  <manifest>
    <item id="front" href="Images/front.png" media-type="image/png"/>
    <item id="chapter" href="Text/chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="chapter"/></spine>
</package>`)

	meta, err := NewParser().ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata returned error: %v", err)
	}
	if string(meta.CoverData) != "png-cover" || meta.CoverType != "image/png" {
		t.Fatalf("unexpected fallback cover data/type: %q %q", string(meta.CoverData), meta.CoverType)
	}
}

func TestParseSpinePreservesNCXHierarchyAndAnchors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ncx_test.epub")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	files := map[string]string{
		"META-INF/container.xml": `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns:dc="http://purl.org/dc/elements/1.1/">
  <metadata><dc:title>NCX Test</dc:title></metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="sp0" href="Text/index_split_000.html" media-type="application/xhtml+xml"/>
    <item id="sp1" href="Text/index_split_001.html" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="sp0"/>
    <itemref idref="sp1"/>
  </spine>
</package>`,
		"OEBPS/toc.ncx": `<?xml version="1.0" encoding="utf-8" ?>
<ncx version="2005-1" xmlns="http://www.daisy.org/z3986/2005/ncx/">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Prologue</text></navLabel>
      <content src="Text/index_split_000.html"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="Text/index_split_001.html"/>
      <navPoint id="np3" playOrder="3">
        <navLabel><text>Part 1</text></navLabel>
        <content src="Text/index_split_001.html#part1"/>
      </navPoint>
      <navPoint id="np4" playOrder="4">
        <navLabel><text>Part 2</text></navLabel>
        <content src="Text/index_split_001.html#part2"/>
      </navPoint>
    </navPoint>
  </navMap>
</ncx>`,
		"OEBPS/Text/index_split_000.html": "<html><body>Prologue</body></html>",
		"OEBPS/Text/index_split_001.html": "<html><body>Chapter 1</body></html>",
	}
	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	_ = zw.Close()

	chapters, err := NewParser().ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine returned error: %v", err)
	}

	if len(chapters) != 4 {
		t.Fatalf("len(chapters) = %d, want 4", len(chapters))
	}

	expected := []struct {
		title string
		path  string
	}{
		{"Prologue", "OEBPS/Text/index_split_000.html"},
		{"Chapter 1", "OEBPS/Text/index_split_001.html"},
		{"Part 1", "OEBPS/Text/index_split_001.html#part1"},
		{"Part 2", "OEBPS/Text/index_split_001.html#part2"},
	}

	for i, exp := range expected {
		if chapters[i].Title != exp.title || chapters[i].ContentPath != exp.path {
			t.Errorf("chapter[%d] = {%q, %q}, want {%q, %q}", i, chapters[i].Title, chapters[i].ContentPath, exp.title, exp.path)
		}
	}
}

func writeEPUBFixture(t *testing.T, opf string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.epub")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()
	files := map[string]string{
		"META-INF/container.xml": `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OPS/content.opf":        opf,
		"OPS/Images/cover.jpg":   "jpeg-cover",
		"OPS/Images/front.png":   "png-cover",
		"OPS/Text/chapter.xhtml": "<html><head><title>Chapter</title></head><body><p>Text</p></body></html>",
	}
	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	return path
}

func TestSaveOriginalMetadataAndFix(t *testing.T) {
	path := writeEPUBFixture(t, `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns:dc="http://purl.org/dc/elements/1.1/">
  <metadata>
    <dc:title>Original Title</dc:title>
    <dc:creator>Original Author</dc:creator>
  </metadata>
  <manifest>
    <item id="chapter" href="Text/chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="chapter"/></spine>
</package>`)

	parser := NewParser()
	update := &bookparser.BookMetadata{
		Title:       "Updated Title",
		Author:      "Updated Author",
		Description: "Updated Description",
		Publisher:   "NovelHub Publishing",
		Language:    "vi",
		Subjects:    []string{"Fantasy", "Adventure"},
		Series:      "Novel Series",
		SeriesIndex: "2.5",
	}

	if err := parser.SaveOriginalMetadataAndFix(path, update); err != nil {
		t.Fatalf("SaveOriginalMetadataAndFix failed: %v", err)
	}

	parsed, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata after update failed: %v", err)
	}

	if parsed.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", parsed.Title)
	}
	if parsed.Author != "Updated Author" {
		t.Errorf("Author = %q, want 'Updated Author'", parsed.Author)
	}
	if parsed.Publisher != "NovelHub Publishing" {
		t.Errorf("Publisher = %q, want 'NovelHub Publishing'", parsed.Publisher)
	}
	if parsed.Language != "vi" {
		t.Errorf("Language = %q, want 'vi'", parsed.Language)
	}
	if parsed.Series != "Novel Series" {
		t.Errorf("Series = %q, want 'Novel Series'", parsed.Series)
	}
	if parsed.SeriesIndex != "2.5" {
		t.Errorf("SeriesIndex = %q, want '2.5'", parsed.SeriesIndex)
	}
}

func TestCheckUserEPUBFiles(t *testing.T) {
	files := []string{
		"../../../Và Rồi, Tháng 9 Không Có Cậu Đã Tới.epub",
		"../../../Tập 07 - Omiya Yuu.epub",
	}
	parser := NewParser()
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			continue
		}
		chapters, err := parser.ParseSpine(f)
		if err != nil {
			t.Errorf("ParseSpine failed for %s: %v", f, err)
			continue
		}
		if len(chapters) == 0 {
			t.Errorf("ParseSpine returned 0 chapters for %s", f)
		}
		t.Logf("File: %s -> %d chapters parsed cleanly", filepath.Base(f), len(chapters))
		for i, ch := range chapters {
			t.Logf("  [%d] %s -> %s", i+1, ch.Title, ch.ContentPath)
		}
	}
}
