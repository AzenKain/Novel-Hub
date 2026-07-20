package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTMLDocumentToReaderHTMLFallsBackToFragments(t *testing.T) {
	raw := `<html><body><div></div></body></html><html><body><p>Readable KF8 fragment text lives here.</p></body></html>`
	html := htmlDocumentToReaderHTML(raw)
	if strings.Contains(html, "cannot decode yet") {
		t.Fatalf("unexpected unsupported html: %s", html)
	}
	if !strings.Contains(html, "Readable KF8 fragment text lives here.") {
		t.Fatalf("expected readable fragment, got %s", html)
	}
}

func TestHTMLDocumentToReaderHTMLStripsUnsupportedKindleImages(t *testing.T) {
	raw := `<html><body><p>Readable text</p><a href="kindle:pos:fid:0002:off:0000000000">TOC item</a><img src="kindle:embed:0008?mime=image/jpeg" alt="cover"><img src="kindle:flow:0005?mime=image/svg+xml"><p>More text</p></body></html>`
	html := htmlDocumentToReaderHTML(raw)
	if strings.Contains(html, "kindle:") {
		t.Fatalf("expected unsupported kindle image references to be removed, got %s", html)
	}
	if strings.Contains(html, "<img") {
		t.Fatalf("expected unsupported image tag to be removed, got %s", html)
	}
	if strings.Contains(html, "href=") {
		t.Fatalf("expected unsupported kindle link href to be removed, got %s", html)
	}
	if !strings.Contains(html, "Readable text") || !strings.Contains(html, "TOC item") || !strings.Contains(html, "More text") {
		t.Fatalf("expected text content to remain, got %s", html)
	}
}

func TestHTMLDocumentToReaderHTMLStripsBrokenAttributeText(t *testing.T) {
	raw := `<html><body><h1>Quyển Thứ Nhất</h1><p>Tôi yêu em.</p> ize="4">-- Rabindranath Tagore<p>HỒ SƠ TÂM LÝ PHẠM TỘI</p></body></html>`
	html := htmlDocumentToReaderHTML(raw)
	if strings.Contains(html, `ize="4">`) {
		t.Fatalf("expected broken attribute fragment to be removed, got %s", html)
	}
	if !strings.Contains(html, "Rabindranath Tagore") {
		t.Fatalf("expected readable text to remain, got %s", html)
	}
}

func TestHTMLDocumentToReaderHTMLKeepsValidTagAttributes(t *testing.T) {
	raw := `<html><body><p><font size="4">Tôi chữa trị cho em.</font></p></body></html>`
	html := normalizeMobiSectionHTML(htmlDocumentToReaderHTML(raw))
	if !strings.Contains(html, `<font size="4">`) {
		t.Fatalf("expected valid font tag to remain, got %s", html)
	}
	if !strings.Contains(html, "Tôi chữa trị cho em.") {
		t.Fatalf("expected readable text to remain, got %s", html)
	}
}

func TestHTMLDocumentToReaderHTMLNormalizesParagraphPresentationAttrs(t *testing.T) {
	raw := `<html><body><p height="1em" width="0pt" align="center"><font size="4"><b>Chương 1</b></font></p></body></html>`
	html := normalizeMobiSectionHTML(htmlDocumentToReaderHTML(raw))
	if !strings.Contains(html, `height="1em"`) || !strings.Contains(html, `width="0pt"`) {
		t.Fatalf("expected legacy paragraph attrs to remain, got %s", html)
	}
	if !strings.Contains(html, `<font size="4"><b>Chương 1</b></font>`) {
		t.Fatalf("expected centered paragraph formatting to remain, got %s", html)
	}
}

func TestHTMLDocumentToReaderHTMLStripsEscapedEmptyAnchorText(t *testing.T) {
	raw := `<html><body><p>chạy từ &lt;a id="x.10355"&gt;&lt;/a&gt;Cách Thử đến Phương Mai</p></body></html>`
	html := normalizeMobiSectionHTML(htmlDocumentToReaderHTML(raw))
	if strings.Contains(html, `&lt;a id="x.10355"&gt;&lt;/a&gt;`) {
		t.Fatalf("expected escaped empty anchor fragment to be removed, got %s", html)
	}
	if !strings.Contains(html, "Cách Thử đến Phương Mai") {
		t.Fatalf("expected readable text to remain, got %s", html)
	}
}

func TestSplitMobiSectionsSkipsNavigationAndSplitsChapters(t *testing.T) {
	readerHTML := htmlDocumentToReaderHTML(`<html><body>
		<h2>Table of Contents</h2>
		<ul><li><a href="kindle:pos:fid:0002:off:0">Chapter One</a></li></ul>
		<h1>Chapter One</h1>
		<p>This is the first chapter with enough readable words to keep the section.</p>
		<h1>Chapter Two</h1>
		<p>This is the second chapter with enough readable words to keep the section.</p>
	</body></html>`)

	sections := splitMobiSections(readerHTML, "Fallback")
	if len(sections) != 2 {
		t.Fatalf("sections = %d, want 2: %#v", len(sections), sections)
	}
	if sections[0].Title != "Chapter One" || sections[1].Title != "Chapter Two" {
		t.Fatalf("unexpected section titles: %#v", sections)
	}
	if strings.Contains(sections[0].Content, "Chapter Two") {
		t.Fatalf("first section contains second section content: %s", sections[0].Content)
	}
}

func TestSplitMobiSectionsUsesFileposNavigationWhenHeadingsAreMissing(t *testing.T) {
	body := `<p>Mục lục</p>` +
		`<a filepos=0000000000><b>Chapter One</b></a>` +
		`<a filepos=1111111111><b>Chapter Two</b></a>` +
		`<p>` + strings.Repeat("toc filler ", 180) + `</p>` +
		`<p><b>Chapter One</b></p><p>This is the first chapter with enough readable words to keep the section.</p>` +
		`<p><b>Chapter Two</b></p><p>This is the second chapter with enough readable words to keep the section.</p>`
	firstOffset := strings.LastIndex(body, `<p><b>Chapter One`)
	secondOffset := strings.LastIndex(body, `<p><b>Chapter Two`)
	body = strings.Replace(body, "filepos=0000000000", fmt.Sprintf("filepos=%010d", firstOffset), 1)
	body = strings.Replace(body, "filepos=1111111111", fmt.Sprintf("filepos=%010d", secondOffset), 1)

	sections := splitMobiSections(`<article class="novelhub-mobi">`+body+`</article>`, "Fallback")
	if len(sections) != 2 {
		t.Fatalf("sections = %d, want 2: %#v", len(sections), sections)
	}
	if sections[0].Title != "Chapter One" || sections[1].Title != "Chapter Two" {
		t.Fatalf("unexpected section titles: %#v", sections)
	}
	if strings.Contains(sections[0].Content, "Mục lục") || strings.Contains(sections[0].Content, "Chapter Two") {
		t.Fatalf("filepos section was not cut correctly: %s", sections[0].Content)
	}
}

func TestMobiListsAndLoadsEmbeddedImageAssets(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0xff, 0xd9}
	path := writeMobiFixture(t, [][]byte{
		mobiHeaderRecord(2),
		[]byte("<html><body>text</body></html>"),
		jpeg,
	})

	parser := NewParser()
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages returned error: %v", err)
	}
	if len(images) != 1 || images[0] != "images/kindle-0001.jpg" {
		t.Fatalf("images = %#v, want kindle image", images)
	}

	data, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset by listed name returned error: %v", err)
	}
	if !bytes.Equal(data, jpeg) {
		t.Fatalf("listed asset bytes = %x, want %x", data, jpeg)
	}

	data, err = parser.GetAsset(path, "kindle:embed:0001?mime=image/jpeg")
	if err != nil {
		t.Fatalf("GetAsset by kindle ref returned error: %v", err)
	}
	if !bytes.Equal(data, jpeg) {
		t.Fatalf("kindle asset bytes = %x, want %x", data, jpeg)
	}

	data, err = parser.GetAsset(path, "images/record-0002.jpg")
	if err != nil {
		t.Fatalf("GetAsset by record alias returned error: %v", err)
	}
	if !bytes.Equal(data, jpeg) {
		t.Fatalf("record asset bytes = %x, want %x", data, jpeg)
	}
}

func TestMobiListsRecordAssetsWithoutImageIndex(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00}
	path := writeMobiFixture(t, [][]byte{
		mobiHeaderRecord(0xffffffff),
		[]byte("text"),
		png,
	})

	parser := NewParser()
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages returned error: %v", err)
	}
	if len(images) != 1 || images[0] != "images/record-0002.png" {
		t.Fatalf("images = %#v, want record image", images)
	}
}

func mobiHeaderRecord(firstImageIndex uint32) []byte {
	record := make([]byte, 16+0x60)
	mobiStart := 16
	copy(record[mobiStart:], []byte("MOBI"))
	binary.BigEndian.PutUint32(record[mobiStart+4:mobiStart+8], 0x60)
	binary.BigEndian.PutUint32(record[mobiStart+0x5c:mobiStart+0x60], firstImageIndex)
	return record
}

func writeMobiFixture(t *testing.T, records [][]byte) string {
	t.Helper()
	header := make([]byte, 78)
	copy(header[:32], []byte("Asset Fixture"))
	binary.BigEndian.PutUint16(header[76:78], uint16(len(records)))
	tableEnd := len(header) + len(records)*8
	data := make([]byte, tableEnd)
	copy(data, header)

	offset := tableEnd
	for index, record := range records {
		binary.BigEndian.PutUint32(data[78+index*8:82+index*8], uint32(offset))
		offset += len(record)
	}
	for _, record := range records {
		data = append(data, record...)
	}

	path := filepath.Join(t.TempDir(), "fixture.mobi")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}
