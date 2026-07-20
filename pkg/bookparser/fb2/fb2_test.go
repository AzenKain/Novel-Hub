package fb2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserReadsFB2MetadataContentAndAssets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.fb2")
	content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>Ada</first-name><last-name>Lovelace</last-name></author>
      <book-title>Analytical Tales</book-title>
      <annotation><p>A <strong>machine</strong> story.</p></annotation>
      <lang>en</lang>
      <sequence name="Engines" number="2"/>
      <coverpage><image l:href="#cover.png"/></coverpage>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Chapter One</p></title>
      <p>Hello <emphasis>reader</emphasis>.</p>
      <image l:href="#cover.png"/>
    </section>
  </body>
  <binary id="cover.png" content-type="image/png">iVBORw0KGgo=</binary>
</FictionBook>`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}
	if meta.Title != "Analytical Tales" || meta.Author != "Ada Lovelace" || meta.Series != "Engines" || meta.SeriesIndex != "2" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if len(meta.CoverData) == 0 || meta.CoverType != "image/png" {
		t.Fatalf("expected cover data, got type=%q len=%d", meta.CoverType, len(meta.CoverData))
	}

	spine, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine failed: %v", err)
	}
	if len(spine) != 1 || spine[0].Title != "Chapter One" {
		t.Fatalf("unexpected spine: %+v", spine)
	}
	html, err := parser.GetChapterContent(path, spine[0].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent failed: %v", err)
	}
	if !strings.Contains(html, "<em>reader</em>") || !strings.Contains(html, `src="images/cover.png"`) {
		t.Fatalf("unexpected html: %s", html)
	}
	asset, err := parser.GetAsset(path, "images/cover.png")
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if len(asset) == 0 {
		t.Fatal("expected decoded asset")
	}
}

func TestFB2CrossSectionLinksAndParagraphAnchors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.fb2")
	content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns:l="http://www.w3.org/1999/xlink">
  <body>
    <section id="s_1">
      <title><p>Chapter One</p></title>
      <p>Go to <a l:href="#s_2">Chapter 2</a>.</p>
      <p>Jump to <a l:href="#p_2">last paragraph</a>.</p>
    </section>
    <section id="s_2">
      <title><p>Chapter Two</p></title>
      <p id="p_2">Destination paragraph.</p>
    </section>
  </body>
</FictionBook>`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	spine, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine failed: %v", err)
	}
	if len(spine) != 2 {
		t.Fatalf("unexpected spine length: %d", len(spine))
	}

	chapterOne, err := parser.GetChapterContent(path, spine[0].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent chapter one failed: %v", err)
	}
	if !strings.Contains(chapterOne, `href="section:1#s_2"`) {
		t.Fatalf("expected cross-section href, got %s", chapterOne)
	}
	if !strings.Contains(chapterOne, `href="section:1#p_2"`) {
		t.Fatalf("expected cross-paragraph href, got %s", chapterOne)
	}

	chapterTwo, err := parser.GetChapterContent(path, spine[1].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent chapter two failed: %v", err)
	}
	if !strings.Contains(chapterTwo, `<section id="s_2">`) || !strings.Contains(chapterTwo, `<p id="p_2">`) {
		t.Fatalf("expected destination ids to be preserved, got %s", chapterTwo)
	}
}
