package ebookconv

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/comic"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/bookparser/fb2"
	"novelhub/pkg/bookparser/plain"
)

func testRegistry(t *testing.T) bookparser.Registry {
	t.Helper()
	reg := bookparser.NewRegistry()
	reg.Register(epub.NewParser(), "epub")
	reg.Register(plain.NewParser(), "txt")
	reg.Register(fb2.NewParser(), "fb2")
	reg.Register(comic.NewParser("cbz"), "cbz")
	reg.Register(comic.NewParser("cbr"), "cbr")
	return reg
}

// --- fixtures ---------------------------------------------------------------

func writeZip(t *testing.T, name string, entries map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return path
}

func writeSourceEPUB(t *testing.T, withImage bool) string {
	t.Helper()
	chn1 := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter One</title></head><body><h1>Chapter One</h1><p>First prose paragraph.</p><p>Second prose with <strong>bold</strong> emphasis.</p>`
	if withImage {
		chn1 += `<p><img src="../Images/cover.png" alt="cover"/></p>`
	}
	chn1 += `</body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf": []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Source Title</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>en</dc:language></metadata><manifest><item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/><item id="c1" href="Text/chapter_1.xhtml" media-type="application/xhtml+xml"/><item id="c2" href="Text/chapter_2.xhtml" media-type="application/xhtml+xml"/><item id="img1" href="Images/cover.png" media-type="image/png"/></manifest><spine toc="ncx"><itemref idref="c1"/><itemref idref="c2"/></spine></package>`),
		"OEBPS/toc.ncx": []byte(`<?xml version="1.0"?><ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap><navPoint id="n1" playOrder="1"><navLabel><text>Chapter One</text></navLabel><content src="Text/chapter_1.xhtml"/></navPoint><navPoint id="n2" playOrder="2"><navLabel><text>Chapter Two</text></navLabel><content src="Text/chapter_2.xhtml"/></navPoint></navMap></ncx>`),
		"OEBPS/Text/chapter_1.xhtml": []byte(chn1),
		"OEBPS/Text/chapter_2.xhtml": []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter Two</title></head><body><h1>Chapter Two</h1><p>Second chapter standalone prose.</p></body></html>`),
	}
	if withImage {
		entries["OEBPS/Images/cover.png"] = []byte("\x89PNG\r\n\x1a\nfake-png-bytes")
	}
	path := writeZip(t, "fixture.epub", entries)
	return path
}

func writeSourceTXT(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plain.txt")
	content := "Plain Book Title\n\nChapter One\n\n" +
		"This is the first chapter text.\nAnother line of it.\n\n" +
		"Chapter Two\n\nThis is the second chapter text.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	return path
}

func writeSourceFB2(t *testing.T) string {
	t.Helper()
	content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
<description><title-info>
<author><first-name>Jane</first-name><last-name>Doe</last-name></author>
<book-title>FB2 Source</book-title><lang>en</lang>
</title-info></description>
<body>
<section><title><p>Alpha Section</p></title><p>Alpha body prose.</p></section>
<section><title><p>Beta Section</p></title><p>Beta body prose.</p></section>
</body>
</FictionBook>`
	path := filepath.Join(t.TempDir(), "source.fb2")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fb2: %v", err)
	}
	return path
}

func writeSourceCBZ(t *testing.T) string {
	t.Helper()
	entries := map[string][]byte{
		"images/1.jpg": []byte("jpeg-bytes-1"),
		"images/2.png": []byte("png-bytes-2"),
	}
	path := writeZip(t, "fixture.cbz", entries)
	return path
}

func writeSourceCBR(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake.cbr")
	if err := os.WriteFile(path, []byte("fake-rar"), 0o644); err != nil {
		t.Fatalf("write cbr: %v", err)
	}
	return path
}

// --- zip helpers ------------------------------------------------------------

func zipEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	out := make(map[string][]byte)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		rc.Close()
		out[f.Name] = buf.Bytes()
	}
	return out
}

func convert(t *testing.T, reg bookparser.Registry, format, path, target string) []byte {
	t.Helper()
	out, err := Convert(reg, format, path, target)
	if err != nil {
		t.Fatalf("Convert(%s → %s): %v", format, target, err)
	}
	return out
}

// --- tests ------------------------------------------------------------------

func TestUnsupportedTargets(t *testing.T) {
	reg := testRegistry(t)
	path := writeSourceTXT(t)
	for _, target := range []string{"azw3", "unknown"} {
		if _, err := Convert(reg, "txt", path, target); err == nil {
			t.Errorf("Convert → %s: expected error, got nil", target)
		} else if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("Convert → %s: error %q does not mention not supported", target, err)
		}
	}
}

func TestSameFormatRejected(t *testing.T) {
	reg := testRegistry(t)
	path := writeSourceEPUB(t, false)
	if _, err := Convert(reg, "epub", path, "epub"); err == nil {
		t.Fatal("Convert epub → epub: expected error")
	}
}

func TestTXTtoEPUB(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "txt", writeSourceTXT(t), "epub")
	entries := zipEntries(t, out)

	if string(entries["mimetype"]) != "application/epub+zip" {
		t.Errorf("mimetype = %q", entries["mimetype"])
	}
	if string(entries["META-INF/container.xml"]) == "" {
		t.Error("missing container.xml")
	}
	opf := string(entries["OEBPS/content.opf"])
	if !strings.Contains(opf, "<dc:title>plain</dc:title>") {
		t.Errorf("opf title wrong: %s", opf)
	}
	// A .txt source is one chapter holding the whole document.
	chapter := string(entries["OEBPS/chapter_1.xhtml"])
	if !strings.Contains(chapter, "This is the first chapter text.") {
		t.Errorf("chapter text missing: %s", chapter)
	}
	if !strings.Contains(chapter, "This is the second chapter text.") {
		t.Errorf("second chapter text missing: %s", chapter)
	}
}

func TestEPUBtoTXT(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "epub", writeSourceEPUB(t, false), "txt")
	text := string(out)
	if !strings.Contains(text, "Source Title") {
		t.Errorf("missing title: %s", text)
	}
	if !strings.Contains(text, "First prose paragraph.") {
		t.Errorf("missing chapter one prose: %s", text)
	}
	if !strings.Contains(text, "Second chapter standalone prose.") {
		t.Errorf("missing chapter two prose: %s", text)
	}
	if strings.Contains(text, "<p>") || strings.Contains(text, "<html") {
		t.Errorf("txt output contains HTML tags: %s", text)
	}
}

func TestEPUBtoFB2(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "epub", writeSourceEPUB(t, true), "fb2")
	var doc struct {
		XMLName xml.Name `xml:"FictionBook"`
		Title   string   `xml:"description>title-info>book-title"`
		Authors []struct {
			FirstName string `xml:"first-name"`
			LastName  string `xml:"last-name"`
		} `xml:"description>title-info>author"`
		Sections []struct {
			Title string `xml:"title>p"`
		} `xml:"body>section"`
		Binaries []struct {
			ID          string `xml:"id,attr"`
			ContentType string `xml:"content-type,attr"`
			Data        string `xml:",chardata"`
		} `xml:"binary"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("fb2 not valid XML: %v\n%s", err, out)
	}
	if doc.Title != "Source Title" {
		t.Errorf("fb2 title = %q", doc.Title)
	}
	if len(doc.Authors) != 1 || doc.Authors[0].FirstName != "Jane" || doc.Authors[0].LastName != "Doe" {
		t.Errorf("fb2 authors = %+v", doc.Authors)
	}
	if len(doc.Sections) != 2 {
		t.Fatalf("fb2 sections = %d, want 2", len(doc.Sections))
	}
	if doc.Sections[0].Title != "Chapter One" {
		t.Errorf("section[0].title = %q", doc.Sections[0].Title)
	}
	// Cover + the manifest image → prose must be present in the sections.
	body := string(out)
	for _, want := range []string{"First prose paragraph.", "<strong>bold</strong>", "Second chapter standalone prose."} {
		if !strings.Contains(body, want) {
			t.Errorf("fb2 prose missing %q", want)
		}
	}
	if len(doc.Binaries) < 1 {
		t.Errorf("fb2 binaries = %d, want >= 1", len(doc.Binaries))
	}
	for _, b := range doc.Binaries {
		if b.ID == "" || b.ContentType == "" || b.Data == "" {
			t.Errorf("malformed binary: %+v", b)
		}
	}
}

func TestEPUBtoDOCX(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "epub", writeSourceEPUB(t, false), "docx")
	entries := zipEntries(t, out)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		if len(entries[name]) == 0 {
			t.Errorf("docx missing %s", name)
		}
	}
	doc := string(entries["word/document.xml"])
	if !strings.Contains(doc, "Source Title") {
		t.Errorf("docx missing title: %s", doc)
	}
	if !strings.Contains(doc, "First prose paragraph.") {
		t.Errorf("docx missing prose: %s", doc)
	}
	if !strings.Contains(doc, "Second chapter standalone prose.") {
		t.Errorf("docx missing chapter two: %s", doc)
	}
}

func TestCBZtoEPUB(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "cbz", writeSourceCBZ(t), "epub")
	entries := zipEntries(t, out)
	opf := string(entries["OEBPS/content.opf"])
	if !strings.Contains(opf, `media-type="image/jpeg"`) || !strings.Contains(opf, `media-type="image/png"`) {
		t.Errorf("opf missing image items: %s", opf)
	}
	chapter := string(entries["OEBPS/chapter_1.xhtml"])
	if !strings.Contains(chapter, `src="images/img_1.jpg"`) {
		t.Errorf("chapter image not rebased: %s", chapter)
	}
	if !strings.Contains(chapter, `src="images/img_2.png"`) {
		t.Errorf("chapter second image not rebased: %s", chapter)
	}
	if string(entries["OEBPS/images/img_1.jpg"]) != "jpeg-bytes-1" {
		t.Errorf("image 1 content wrong")
	}
	if string(entries["OEBPS/images/img_2.png"]) != "png-bytes-2" {
		t.Errorf("image 2 content wrong")
	}
}

func TestEPUBtoCBZ(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "epub", writeSourceEPUB(t, true), "cbz")
	entries := zipEntries(t, out)
	if len(entries) != 1 {
		t.Fatalf("cbz entries = %d, want 1: %v", len(entries), entries)
	}
	for name, data := range entries {
		if string(data) != "\x89PNG\r\n\x1a\nfake-png-bytes" {
			t.Errorf("cbz image %s content wrong", name)
		}
	}
}

func TestCBZWithoutImagesRejected(t *testing.T) {
	reg := testRegistry(t)
	if _, err := Convert(reg, "epub", writeSourceEPUB(t, false), "cbz"); err == nil {
		t.Fatal("epub without images → cbz: expected error")
	}
}

func TestFB2toEPUB(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "fb2", writeSourceFB2(t), "epub")
	entries := zipEntries(t, out)
	joined := string(entries["OEBPS/chapter_1.xhtml"]) + string(entries["OEBPS/chapter_2.xhtml"])
	if !strings.Contains(joined, "Alpha body prose.") || !strings.Contains(joined, "Beta body prose.") {
		t.Errorf("fb2 prose missing after conversion: %s", joined)
	}
	if !strings.Contains(string(entries["OEBPS/content.opf"]), "<dc:title>FB2 Source</dc:title>") {
		t.Errorf("fb2 title not carried into epub opf")
	}
}

func TestRoundTripFB2toEPUBtoFB2(t *testing.T) {
	reg := testRegistry(t)
	epubData := convert(t, reg, "fb2", writeSourceFB2(t), "epub")
	epubPath := filepath.Join(t.TempDir(), "roundtrip.epub")
	if err := os.WriteFile(epubPath, epubData, 0o644); err != nil {
		t.Fatalf("write roundtrip epub: %v", err)
	}
	back := convert(t, reg, "epub", epubPath, "fb2")
	var doc struct {
		Sections []struct {
			Title string `xml:"title>p"`
			Body  string `xml:"p"`
		} `xml:"body>section"`
	}
	if err := xml.Unmarshal(back, &doc); err != nil {
		t.Fatalf("roundtrip fb2 invalid: %v", err)
	}
	if len(doc.Sections) != 2 {
		t.Fatalf("roundtrip sections = %d, want 2", len(doc.Sections))
	}
	if doc.Sections[0].Title != "Alpha Section" || doc.Sections[1].Title != "Beta Section" {
		t.Errorf("roundtrip section titles = %q, %q", doc.Sections[0].Title, doc.Sections[1].Title)
	}
	if !strings.Contains(doc.Sections[0].Body, "Alpha body prose.") {
		t.Errorf("roundtrip section[0] body = %q", doc.Sections[0].Body)
	}
}

func TestCBRUnsupportedReader(t *testing.T) {
	reg := testRegistry(t)
	path := writeSourceCBR(t)
	if _, err := Convert(reg, "cbr", path, "epub"); err == nil {
		t.Fatal("cbr → epub: expected SimulatedReader error (no RAR support in test env)")
	}
}

func TestMimetypeIsFirstAndStored(t *testing.T) {
	reg := testRegistry(t)
	out := convert(t, reg, "txt", writeSourceTXT(t), "epub")
	r, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	if len(r.File) == 0 || r.File[0].Name != "mimetype" {
		t.Fatalf("first entry = %q, want mimetype", firstEntryName(r))
	}
	if r.File[0].Method != zip.Store {
		t.Errorf("mimetype method = %d, want Store(%d)", r.File[0].Method, zip.Store)
	}
}

func firstEntryName(r *zip.Reader) string {
	if len(r.File) == 0 {
		return ""
	}
	return r.File[0].Name
}

func TestFragmentTextAndSplit(t *testing.T) {
	frag := `<article class="x"><h1>Heading</h1><p>Para one.</p><p>Para <strong>two</strong>.<br/>Next line.</p></article>`
	text := fragmentText(frag)
	for _, want := range []string{"Heading", "Para one.", "Para two.", "Next line."} {
		if !strings.Contains(text, want) {
			t.Errorf("fragmentText missing %q", want)
		}
	}
	if strings.Contains(text, "<") || strings.Contains(text, ">") {
		t.Errorf("fragmentText leaked tags: %q", text)
	}
	blocks := splitParagraphs(frag)
	if len(blocks) < 3 {
		t.Fatalf("splitParagraphs = %d, want >= 3", len(blocks))
	}
}

func TestImageLookupAndFilename(t *testing.T) {
	images := []Image{
		{Src: "OEBPS/Images/cover.png", Name: imageFilename("OEBPS/Images/cover.png", 1), Data: []byte("x")},
		{Src: "images/2.JPG", Name: imageFilename("images/2.JPG", 2), Data: []byte("y")},
	}
	idx := imageLookup(images)
	if idx["cover.png"] != 0 {
		t.Errorf("lookup cover.png = %d, want 0", idx["cover.png"])
	}
	if idx["2.jpg"] != 1 {
		t.Errorf("lookup 2.jpg = %d, want 1", idx["2.jpg"])
	}
	if images[0].Name != "img_1.png" || images[1].Name != "img_2.jpg" {
		t.Errorf("image names = %q, %q", images[0].Name, images[1].Name)
	}
}

func TestSplitAuthor(t *testing.T) {
	cases := []struct{ in, first, last string }{
		{"Jane Doe", "Jane", "Doe"},
		{"Jane Doe Jr.", "Jane", "Doe Jr."},
		{"Single", "", "Single"},
		{"", "", ""},
	}
	for _, c := range cases {
		f, l := splitAuthor(c.in)
		if f != c.first || l != c.last {
			t.Errorf("splitAuthor(%q) = %q, %q; want %q, %q", c.in, f, l, c.first, c.last)
		}
	}
}

func TestIsTargetSupported(t *testing.T) {
	for _, ok := range []struct {
		target string
		want   bool
	}{
		{"epub", true}, {"FB2", true}, {"TXT", true}, {"cbz", true}, {"Docx", true},
		{"kepub.epub", true}, {"pdf", true}, {"mobi", true}, {"azw", true}, {"", false},
	} {
		if got := IsTargetSupported(ok.target); got != ok.want {
			t.Errorf("IsTargetSupported(%q) = %v, want %v", ok.target, got, ok.want)
		}
	}
}