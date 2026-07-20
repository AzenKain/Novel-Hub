package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
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
