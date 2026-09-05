package ebookconv

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/image/font/sfnt"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/comic"
	"novelhub/pkg/bookparser/epub"
	"novelhub/pkg/bookparser/fb2"
	"novelhub/pkg/bookparser/mobi"
	"novelhub/pkg/bookparser/pdf"
	"novelhub/pkg/bookparser/plain"
)

func regressionRegistry(t *testing.T) bookparser.Registry {
	t.Helper()
	reg := bookparser.NewRegistry()
	reg.Register(epub.NewParser(), "epub", "kepub.epub")
	reg.Register(plain.NewParser(), "txt")
	reg.Register(fb2.NewParser(), "fb2")
	reg.Register(comic.NewParser("cbz"), "cbz")
	reg.Register(comic.NewParser("cbr"), "cbr")
	reg.Register(mobi.NewParser(), "mobi", "azw")
	reg.Register(pdf.NewParser(), "pdf")
	return reg
}

func zipNames(t *testing.T, data []byte) []string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

func joinContent(t *testing.T, parser bookparser.Parser, path string, chapters []bookparser.ChapterData) string {
	t.Helper()
	var b strings.Builder
	for _, ch := range chapters {
		content, err := parser.GetChapterContent(path, ch.ContentPath)
		if err != nil {
			t.Fatalf("GetChapterContent(%q): %v", ch.ContentPath, err)
		}
		b.WriteString(stripTags(content))
		b.WriteString("\n")
	}
	return b.String()
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeSourceEPUBRealBookShape(t *testing.T) string {
	t.Helper()
	manifest := `<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Real Shape</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>vi</dc:language></metadata><manifest><item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/><item id="c1" href="Text/chapter_1.xhtml" media-type="application/xhtml+xml"/><item id="img1" href="Images/Chương 1 - Ảnh bìa.jpg" media-type=""/><item id="img2" href="Images/Ảnh minh họa.png" media-type=""/></manifest><spine toc="ncx"><itemref idref="c1"/></spine></package>`
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head><body><h1>One</h1><p>Chương một có hình minh họa.</p><p><img src="Images/Ch%C6%B0%C6%A1ng%201%20-%20%E1%BA%A2nh%20b%C3%ACa.jpg?v=1#frag" alt="bia"/></p><p><img src="images/%E1%BA%A2nh%20minh%20h%E1%BB%8Da.png" alt="minhhoa"/></p></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml":              []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":                   []byte(manifest),
		"OEBPS/toc.ncx":                       []byte(`<?xml version="1.0"?><ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap><navPoint id="n1" playOrder="1"><navLabel><text>One</text></navLabel><content src="Text/chapter_1.xhtml"/></navPoint></navMap></ncx>`),
		"OEBPS/Text/chapter_1.xhtml":          []byte(chapter),
		"OEBPS/Images/Chương 1 - Ảnh bìa.jpg": []byte("\xff\xd8\xff\xe0fake-jpeg"),
		"OEBPS/Images/Ảnh minh họa.png":       []byte("\x89PNG\r\n\x1a\nfake-png"),
	}
	return writeZip(t, "real.epub", entries)
}

// TestRealBookShapeImagesSurvive is the regression for "fb2/kepub/docx mất toàn bộ ảnh" on the production book: manifest media-type="" + encoded refs.
func TestRealBookShapeImagesSurvive(t *testing.T) {
	reg := regressionRegistry(t)
	src := writeSourceEPUBRealBookShape(t)
	for _, target := range []string{"fb2", "kepub.epub", "docx", "cbz"} {
		t.Run(target, func(t *testing.T) {
			out := convert(t, reg, "epub", src, target)
			if target == "fb2" {
				text := string(out)
				if !strings.Contains(text, `<image l:href="#fb2img`) {
					t.Errorf("fb2 body lost image refs:\n%.600s", text)
				}
				if !strings.Contains(text, "fb2img1") || !strings.Contains(text, "fb2img2") {
					t.Errorf("fb2 did not embed both binaries: %.600s", text)
				}
				return
			}
			r, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
			if err != nil {
				t.Fatalf("%s not a zip: %v", target, err)
			}
			var images int
			for _, f := range r.File {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				buf := new(bytes.Buffer)
				_, _ = buf.ReadFrom(rc)
				_ = rc.Close()
				b := buf.Bytes()
				if bytes.HasPrefix(b, []byte{0xff, 0xd8}) || bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G'}) {
					images++
				}
			}
			if images != 2 {
				t.Errorf("%s embedded %d images, want 2", target, images)
			}
			if target == "docx" {
				doc := zipEntry(t, r, "word/document.xml")
				if !strings.Contains(doc, `<w:drawing>`) {
					t.Error("docx body has no drawings")
				}
				if !strings.Contains(doc, `r:embed="rIdImg`) {
					t.Error("docx drawings do not reference image rels")
				}
			}
		})
	}
}

func zipEntry(t *testing.T, r *zip.Reader, name string) string {
	t.Helper()
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		defer rc.Close()
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(rc)
		return buf.String()
	}
	return ""
}

// TestMobiRoundTripChapters is the regression for "mobi ko ra toc, content đầu tiên lại là chương 3": round-trips a 2-chapter epub through the mobi writer and re-parses with the mobi reader.
func TestMobiRoundTripChapters(t *testing.T) {
	reg := regressionRegistry(t)
	src := writeSourceEPUB(t, false)
	for _, target := range []string{"mobi", "azw"} {
		t.Run(target, func(t *testing.T) {
			out := convert(t, reg, "epub", src, target)
			path := filepath.Join(t.TempDir(), "out."+target)
			if err := os.WriteFile(path, out, 0o644); err != nil {
				t.Fatal(err)
			}
			parser := mobi.NewParser()
			chapters, err := parser.ParseSpine(path)
			if err != nil {
				t.Fatalf("ParseSpine: %v", err)
			}
			if len(chapters) < 2 {
				t.Fatalf("chapters = %d, want >= 2 (TOC lost)", len(chapters))
			}
			all := joinContent(t, parser, path, chapters)
			for _, want := range []string{"First prose paragraph.", "Second chapter standalone prose."} {
				if !strings.Contains(all, want) {
					t.Errorf("%s round-trip lost %q (text truncated)", target, want)
				}
			}
		})
	}
}

// TestMobiRoundTripImages proves image records survive the mobi writer and are readable back through the reader's GetAsset.
func TestMobiRoundTripImages(t *testing.T) {
	reg := regressionRegistry(t)
	src := writeSourceEPUB(t, true)
	out := convert(t, reg, "epub", src, "mobi")
	path := filepath.Join(t.TempDir(), "out.mobi")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	parser := mobi.NewParser()
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(images) == 0 {
		t.Fatal("mobi output has no image records")
	}
	data, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset(%q): %v", images[0], err)
	}
	if len(data) == 0 {
		t.Error("mobi image asset is empty")
	}
}

// TestKepubRoundTripPreservesText proves the kepub writer emits a valid epub whose prose survives a round-trip through the epub reader.
func TestKepubRoundTripPreservesText(t *testing.T) {
	reg := regressionRegistry(t)
	src := writeSourceEPUB(t, true)
	out := convert(t, reg, "epub", src, "kepub.epub")
	path := filepath.Join(t.TempDir(), "out.kepub.epub")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	parser := epub.NewParser()
	chapters, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine: %v", err)
	}
	all := joinContent(t, parser, path, chapters)
	for _, want := range []string{"First prose paragraph.", "Second chapter standalone prose."} {
		if !strings.Contains(all, want) {
			t.Errorf("kepub round-trip lost %q", want)
		}
	}
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(images) == 0 {
		t.Error("kepub output has no images")
	}
}

// TestPDFBasicStructure is the regression for "pdf sai font, chữ cách nhau xa": the embedded font must parse as a real TTF.
func TestPDFBasicStructure(t *testing.T) {
	reg := regressionRegistry(t)
	out := convert(t, reg, "txt", writeSourceTXT(t), "pdf")
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("output is not a PDF: %.16s", out)
	}
	start := bytes.Index(out, []byte("\x00\x01\x00\x00"))
	if start < 0 {
		t.Fatal("no embedded TrueType font stream")
	}
	end := bytes.Index(out[start:], []byte("endstream"))
	if end < 0 {
		t.Fatal("font stream not terminated")
	}
	if _, err := sfnt.Parse(out[start : start+end]); err != nil {
		t.Fatalf("embedded font is not a valid TTF: %v", err)
	}
}

func writeSourceEPUBRealImages(t *testing.T) string {
	t.Helper()
	pngBytes := encodePNG(t, color.RGBA{255, 0, 0, 128})
	jpgBytes := encodeJPEG(t, color.RGBA{0, 128, 255, 255})
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head><body><h1>One</h1><p><img src="../Images/cover.png" alt="cover"/></p><p>Prose after the images.</p><p><img src="../Images/photo.jpg" alt="photo"/></p></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml":     []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":          []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>PDF Images</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>en</dc:language></metadata><manifest><item id="c1" href="Text/chapter_1.xhtml" media-type="application/xhtml+xml"/><item id="img1" href="Images/cover.png" media-type="image/png"/><item id="img2" href="Images/photo.jpg" media-type="image/jpeg"/></manifest><spine><itemref idref="c1"/></spine></package>`),
		"OEBPS/Text/chapter_1.xhtml": []byte(chapter),
		"OEBPS/Images/cover.png":     pngBytes,
		"OEBPS/Images/photo.jpg":     jpgBytes,
	}
	return writeZip(t, "pdf-images.epub", entries)
}

func encodePNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, c.A})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	return buf.Bytes()
}

// TestConvertToPDFImages is the regression for "pdf chữ cách nhau xa, mất ảnh": the pdf writer must embed both the alpha PNG (as RGB + SMask) and the JPEG (DCTDecode passthrough) as image XObjects.
func TestConvertToPDFImages(t *testing.T) {
	reg := regressionRegistry(t)
	out := convert(t, reg, "epub", writeSourceEPUBRealImages(t), "pdf")
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("output is not a PDF: %.16s", out)
	}
	if got := bytes.Count(out, []byte("/Subtype /Image")); got != 3 {
		t.Errorf("pdf image objects = %d, want 3 (png + smask + jpeg)", got)
	}
	if !bytes.Contains(out, []byte("/DCTDecode")) {
		t.Error("pdf missing /DCTDecode (jpeg passthrough)")
	}
	if !bytes.Contains(out, []byte("/SMask")) {
		t.Error("alpha png should embed a /SMask")
	}
}

// TestPDFImageTextGap is the regression for "pdf chữ nhảy lên ảnh": the content stream must leave a gap between an image's bottom edge and the next text baseline.
func TestPDFImageTextGap(t *testing.T) {
	reg := regressionRegistry(t)
	out := convert(t, reg, "epub", writeSourceEPUBRealImages(t), "pdf")
	imageRe := regexp.MustCompile(`q\n[0-9.]+ 0 0 [0-9.]+ 72 ([0-9.]+) cm\n/Im[0-9]+ Do`)
	textRe := regexp.MustCompile(`\n72 ([0-9.]+) Td`)
	checked, gaps := 0, 0
	for _, stream := range bytes.Split(out, []byte("endstream")) {
		imgs := imageRe.FindAllStringSubmatch(string(stream), -1)
		baselines := textRe.FindAllStringSubmatch(string(stream), -1)
		b := make([]float64, 0, len(baselines))
		for _, m := range baselines {
			f, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				t.Fatalf("bad baseline %q: %v", m[1], err)
			}
			b = append(b, f)
		}
		for _, m := range imgs {
			y, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				t.Fatalf("bad img y %q: %v", m[1], err)
			}
			for _, base := range b {
				if base >= y {
					continue
				}
				checked++
				if gap := y - base; gap < float64(pdfImageGap-2) {
					t.Errorf("gap after image = %.1f, want >= %.1f", gap, float64(pdfImageGap-2))
				} else {
					gaps++
				}
				break
			}
		}
	}
	if checked == 0 || gaps == 0 {
		t.Errorf("no image-then-text pairs found (checked %d, ok %d)", checked, gaps)
	}
}

// TestMobiTOCMatchesSource is the regression for mobi TOC duplication: the source chapter files carry their own <h1>/<h2> headings, and the reader splits its TOC on every heading tag — so a chapter with an internal heading became two "chapters" on round-trip.
func TestMobiTOCMatchesSource(t *testing.T) {
	reg := regressionRegistry(t)
	src := writeSourceEPUB(t, false)
	source, err := epub.NewParser().ParseSpine(src)
	if err != nil {
		t.Fatalf("ParseSpine source: %v", err)
	}
	var sourceTitles []string
	for _, ch := range source {
		sourceTitles = append(sourceTitles, ch.Title)
	}
	for _, target := range []string{"mobi", "azw"} {
		t.Run(target, func(t *testing.T) {
			out := convert(t, reg, "epub", src, target)
			path := filepath.Join(t.TempDir(), "out."+target)
			if err := os.WriteFile(path, out, 0o644); err != nil {
				t.Fatal(err)
			}
			chapters, err := mobi.NewParser().ParseSpine(path)
			if err != nil {
				t.Fatalf("ParseSpine: %v", err)
			}
			if len(chapters) != len(sourceTitles) {
				t.Errorf("mobi chapters = %d, want %d (duplicate in-content headings promoted)", len(chapters), len(sourceTitles))
			}
			seen := make(map[string]int)
			for _, ch := range chapters {
				seen[strings.ToLower(ch.Title)]++
			}
			for _, title := range sourceTitles {
				if seen[strings.ToLower(title)] != 1 {
					t.Errorf("source chapter %q has %d mobi sections, want exactly 1", title, seen[strings.ToLower(title)])
				}
			}
		})
	}
}

// TestMobiTOCChapterSurvives is the regression for the production book whose "Mục lục" spine chapter kept losing its dat a in mobi/azw: a nav-named chapter must stay a first-class section with its item text intact.
func TestMobiTOCChapterSurvives(t *testing.T) {
	reg := regressionRegistry(t)
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mục lục</title></head><body><h1>Mục lục</h1><nav><ol><li><a href="Text/ch1.xhtml">Minh họa</a></li><li><a href="Text/ch2.xhtml">Mở đầu</a></li></ol></nav></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Kepub TOC</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>vi</dc:language></metadata><manifest><item id="c1" href="Text/toc.xhtml" media-type="application/xhtml+xml"/><item id="c2" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/><item id="c3" href="Text/ch2.xhtml" media-type="application/xhtml+xml"/></manifest><spine toc="ncx"><itemref idref="c1"/><itemref idref="c2"/><itemref idref="c3"/></spine></package>`),
		"OEBPS/Text/toc.xhtml":   []byte(chapter),
		"OEBPS/Text/ch1.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Minh họa</title></head><body><h1>Minh họa</h1><p>Một trang minh họa đủ chữ để vượt qua cơ chế lọc mục.</p></body></html>`),
		"OEBPS/Text/ch2.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mở đầu</title></head><body><h1>Mở đầu</h1><p>Văn xuôi mở đầu đủ chữ để vượt qua cơ chế lọc mục.</p></body></html>`),
	}
	src := writeZip(t, "toc.epub", entries)
	for _, target := range []string{"mobi", "azw"} {
		t.Run(target, func(t *testing.T) {
			out := convert(t, reg, "epub", src, target)
			path := filepath.Join(t.TempDir(), "out."+target)
			if err := os.WriteFile(path, out, 0o644); err != nil {
				t.Fatal(err)
			}
			parser := mobi.NewParser()
			chapters, err := parser.ParseSpine(path)
			if err != nil {
				t.Fatalf("ParseSpine: %v", err)
			}
			for _, ch := range chapters {
				if strings.ToLower(ch.Title) != "mục lục" {
					continue
				}
				content, err := parser.GetChapterContent(path, ch.ContentPath)
				if err != nil {
					t.Fatalf("GetChapterContent: %v", err)
				}
				for _, want := range []string{"Minh họa", "Mở đầu"} {
					if !strings.Contains(stripTags(content), want) {
						t.Errorf("TOC section lost %q:\n%s", want, content)
					}
				}
				return
			}
			t.Fatalf("Mục lục section dropped; got %+v", chapters)
		})
	}
}

// TestConvertedTOCTargetsChapterKeys is the regression for "trang mục lục hiện xanh nhưng bấm không nhảy chương": converted books re-host chapters under new content keys (chapter_N.xhtml / mobi-section:N / section:N) while the TOC copy kept the source filenames, so the reader could never map a link back to a chapter.
func TestConvertedTOCTargetsChapterKeys(t *testing.T) {
	reg := regressionRegistry(t)
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mục lục</title></head><body><h1>Mục lục</h1><nav><ol><li><a href="ch1.xhtml">Chapter One</a></li><li><a href="ch2.xhtml">Chapter Two</a></li></ol></nav></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Link TOC</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>en</dc:language></metadata><manifest><item id="c1" href="Text/toc.xhtml" media-type="application/xhtml+xml"/><item id="c2" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/><item id="c3" href="Text/ch2.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/><itemref idref="c2"/><itemref idref="c3"/></spine></package>`),
		"OEBPS/Text/toc.xhtml":   []byte(chapter),
		"OEBPS/Text/ch1.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter One</title></head><body><h1>Chapter One</h1><p>First prose paragraph.</p></body></html>`),
		"OEBPS/Text/ch2.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter Two</title></head><body><h1>Chapter Two</h1><p>Second chapter standalone prose.</p></body></html>`),
	}
	src := writeZip(t, "toclink.epub", entries)
	cases := []struct {
		target string
		want   string
	}{
		{"kepub.epub", `href="chapter_2.xhtml"`},
		{"mobi", `href="mobi-section:1"`},
		{"azw", `href="mobi-section:1"`},
		{"fb2", `href="section:1"`},
	}
	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			out := convert(t, reg, "epub", src, tc.target)
			path := filepath.Join(t.TempDir(), "out."+tc.target)
			if err := os.WriteFile(path, out, 0o644); err != nil {
				t.Fatal(err)
			}
			parser, ok := reg.ParserForFormat(tc.target)
			if !ok {
				t.Fatal("no parser for " + tc.target)
			}
			chapters, err := parser.ParseSpine(path)
			if err != nil {
				t.Fatalf("ParseSpine: %v", err)
			}
			for _, ch := range chapters {
				if strings.ToLower(ch.Title) != "mục lục" {
					continue
				}
				content, err := parser.GetChapterContent(path, ch.ContentPath)
				if err != nil {
					t.Fatalf("GetChapterContent: %v", err)
				}
				if !strings.Contains(content, tc.want) {
					t.Errorf("TOC link not rebased to %s; got\n%s", tc.want, content)
				}
				return
			}
			t.Fatalf("Mục lục section missing in %s", tc.target)
		})
	}
}

// TestConvertedTOCLinksNavigate is the end-to-end round of TestConvertedTOCTargetsChapterKeys: the rebased href must resolve (after the backend's asset-URL rewriting and the reader's path matching) to the target chapter.
func TestConvertedTOCLinksNavigate(t *testing.T) {
	reg := regressionRegistry(t)
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mục lục</title></head><body><h1>Mục lục</h1><nav><ol><li><a href="ch1.xhtml">Chapter One</a></li><li><a href="ch2.xhtml">Chapter Two</a></li></ol></nav></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Link TOC</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>en</dc:language></metadata><manifest><item id="c1" href="Text/toc.xhtml" media-type="application/xhtml+xml"/><item id="c2" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/><item id="c3" href="Text/ch2.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/><itemref idref="c2"/><itemref idref="c3"/></spine></package>`),
		"OEBPS/Text/toc.xhtml":   []byte(chapter),
		"OEBPS/Text/ch1.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter One</title></head><body><h1>Chapter One</h1><p>First prose paragraph.</p></body></html>`),
		"OEBPS/Text/ch2.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter Two</title></head><body><h1>Chapter Two</h1><p>Second chapter standalone prose.</p></body></html>`),
	}
	src := writeZip(t, "toclink.epub", entries)
	for _, target := range []string{"kepub.epub", "mobi", "azw", "fb2"} {
		t.Run(target, func(t *testing.T) {
			out := convert(t, reg, "epub", src, target)
			path := filepath.Join(t.TempDir(), "out."+target)
			if err := os.WriteFile(path, out, 0o644); err != nil {
				t.Fatal(err)
			}
			parser, ok := reg.ParserForFormat(target)
			if !ok {
				t.Fatal("no parser for " + target)
			}
			chapters, err := parser.ParseSpine(path)
			if err != nil {
				t.Fatalf("ParseSpine: %v", err)
			}
			var tocContent string
			for _, ch := range chapters {
				if strings.ToLower(ch.Title) != "mục lục" {
					continue
				}
				tocContent, err = parser.GetChapterContent(path, ch.ContentPath)
				if err != nil {
					t.Fatal(err)
				}
				break
			}
			if tocContent == "" {
				t.Fatal("Mục lục chapter missing")
			}
			found := false
			for _, ch := range chapters {
				if strings.ToLower(ch.Title) == "mục lục" {
					continue
				}
				chPath := strings.ToLower(strings.TrimPrefix(ch.ContentPath, "/"))
				for _, want := range []string{"Chapter One", "Chapter Two"} {
					if ch.Title != want {
						continue
					}
					for _, href := range regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(tocContent, -1) {
						targetPath := strings.TrimPrefix(decodePercentStrings(href[1]), "/")
						if strings.Contains(targetPath, "/asset/") {
							targetPath = targetPath[strings.LastIndex(targetPath, "/asset/")+len("/asset/"):]
						}
						if strings.Contains(targetPath, "?") {
							targetPath = targetPath[:strings.LastIndex(targetPath, "?")]
						}
						if strings.Contains(targetPath, "#") {
							targetPath = targetPath[:strings.LastIndex(targetPath, "#")]
						}
						if chPath == targetPath || strings.HasSuffix(chPath, targetPath) || strings.HasSuffix(targetPath, chPath) {
							found = true
						}
					}
				}
			}
			if !found {
				t.Fatalf("no TOC link in %s resolves to a chapter (content:\n%s)", target, tocContent)
			}
		})
	}
}

func decodePercentStrings(s string) string {
	if v, err := url.PathUnescape(s); err == nil {
		return v
	}
	return s
}

// TestFB2TOCItemsSurvive is the regression for "fb2 trang mục lục mất data": list items under a <nav>/<ol> must render as <p> so the fb2 reader keeps them instead of dropping bare text under its section model.
func TestFB2TOCItemsSurvive(t *testing.T) {
	reg := regressionRegistry(t)
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mục lục</title></head><body><h1>Mục lục</h1><nav><ol><li><a href="Text/ch1.xhtml">Minh họa</a></li><li><a href="Text/ch2.xhtml">Mở đầu – Ước vọng từ quá khứ</a></li></ol></nav></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>FB2 TOC</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>vi</dc:language></metadata><manifest><item id="c1" href="Text/toc.xhtml" media-type="application/xhtml+xml"/></manifest><spine toc="ncx"><itemref idref="c1"/></spine></package>`),
		"OEBPS/Text/toc.xhtml":   []byte(chapter),
	}
	src := writeZip(t, "toc.epub", entries)
	out := convert(t, reg, "epub", src, "fb2")
	path := filepath.Join(t.TempDir(), "out.fb2")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	parser := fb2.NewParser()
	chapters, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine: %v", err)
	}
	all := ""
	for _, ch := range chapters {
		content, err := parser.GetChapterContent(path, ch.ContentPath)
		if err != nil {
			t.Fatalf("GetChapterContent: %v", err)
		}
		all += stripTags(content) + "\n"
	}
	for _, want := range []string{"Minh họa", "Mở đầu – Ước vọng từ quá khứ"} {
		if !strings.Contains(all, want) {
			t.Errorf("fb2 reader lost TOC item %q (got %q)", want, all)
		}
	}
}

// TestPDFVietGlyphCoverage is the regression for "pdf sai font chữ": the embedded font must carry a glyph for every Vietnamese rune the writer might emit, or a reader renders tofu.
func TestPDFVietGlyphCoverage(t *testing.T) {
	f, err := sfnt.Parse(embeddedPDFFont)
	if err != nil {
		t.Fatalf("embedded font: %v", err)
	}
	buf := new(sfnt.Buffer)
	sample := "Xin chào, tôi là sách. Ăn ơi đồng đều. Ở đây ạ ả ã á à â ấ ẫ ẩ ậ đ ê ế ề ể ễ ệ ì ỉ ĩ í ị ò ỏ õ ó ọ ô ố ồ ổ ỗ ộ ờ ở ỡ ớ ợ ù ủ ũ ú ụ ư ứ ừ ử ữ ự ý ỷ ỹ ỵ"
	for _, r := range sample {
		if r == ' ' {
			continue
		}
		index, err := f.GlyphIndex(buf, r)
		if err != nil {
			t.Fatalf("GlyphIndex(%q): %v", r, err)
		}
		if index == 0 {
			t.Errorf("font has no glyph for %q", r)
		}
	}
}

// TestMobiImageOnlySectionSurvives is the regression for "mobi cắt mất trang minh họa": an illustration page carries almost no readable text, so the reader's <24-char section filter used to throw the whole section away.
func TestMobiImageOnlySectionSurvives(t *testing.T) {
	reg := regressionRegistry(t)
	png := encodePNG(t, color.RGBA{0, 128, 255, 255})
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Image Page</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>vi</dc:language></metadata><manifest><item id="c1" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/><item id="c2" href="Text/ch2.xhtml" media-type="application/xhtml+xml"/><item id="c3" href="Text/ch3.xhtml" media-type="application/xhtml+xml"/><item id="img1" href="Images/mh.png" media-type="image/png"/></manifest><spine toc="ncx"><itemref idref="c1"/><itemref idref="c2"/><itemref idref="c3"/></spine></package>`),
		"OEBPS/toc.ncx":          []byte(`<?xml version="1.0"?><ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap><navPoint id="n1" playOrder="1"><navLabel><text>Cover</text></navLabel><content src="Text/ch1.xhtml"/></navPoint><navPoint id="n2" playOrder="2"><navLabel><text>Minh họa</text></navLabel><content src="Text/ch2.xhtml"/></navPoint><navPoint id="n3" playOrder="3"><navLabel><text>Mở đầu</text></navLabel><content src="Text/ch3.xhtml"/></navPoint></navMap></ncx>`),
		"OEBPS/Text/ch1.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Cover</title></head><body><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="150"><image xlink:href="../Images/mh.png" width="100" height="150"/></svg></body></html>`),
		"OEBPS/Text/ch2.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Minh họa</title></head><body><h1>Minh họa</h1><p><img src="../Images/mh.png" alt="minh hoa"/></p></body></html>`),
		"OEBPS/Text/ch3.xhtml":   []byte(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mở đầu</title></head><body><h1>Mở đầu</h1><p>Có một chương dài với văn xuôi thật sự ở đây để vượt qua giới hạn ba mươi hai ký tự của bộ lọc.</p></body></html>`),
		"OEBPS/Images/mh.png":    png,
	}
	src := writeZip(t, "img.epub", entries)
	out := convert(t, reg, "epub", src, "mobi")
	path := filepath.Join(t.TempDir(), "out.mobi")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	parser := mobi.NewParser()
	chapters, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine: %v", err)
	}
	found := make(map[string]bool)
	for _, ch := range chapters {
		key := strings.ToLower(strings.TrimSpace(ch.Title))
		found[key] = true
		content, err := parser.GetChapterContent(path, ch.ContentPath)
		if err != nil {
			t.Fatalf("GetChapterContent(%q): %v", ch.ContentPath, err)
		}
		switch key {
		case "minh họa":
			if !strings.Contains(content, "<img") {
				t.Errorf("image-only section lost its <img>:\n%s", content)
			}
		case "cover":
			if !strings.Contains(content, "images/kindle-") {
				t.Errorf("svg cover section lost its rebased image:\n%s", content)
			}
		}
	}
	for _, want := range []string{"minh họa", "cover", "mở đầu"} {
		if !found[want] {
			t.Fatalf("section %q dropped; sections = %+v", want, chapters)
		}
	}
}

// TestMobiSvgCoverRebased is the regression for the kepub cover page: the writer emits chapter content into mobi rawml, so a source <svg><image xlink:href="..."/></svg> cover must be rebased to the kindle image record.
func TestMobiSvgCoverRebased(t *testing.T) {
	reg := regressionRegistry(t)
	png := encodePNG(t, color.RGBA{200, 30, 30, 255})
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head><body><h1>One</h1><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="150" viewBox="0 0 100 150"><image xlink:href="../Images/cv.png" width="100" height="150"/></svg><p>Trang sau là phần nội dung thật sự của chương truyện này.</p><p>Có khá nhiều câu chữ ở đây để đủ độ dài cho cơ chế đọc mobi.</p></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Svg Cover</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>vi</dc:language></metadata><manifest><item id="c1" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/><item id="img1" href="Images/cv.png" media-type="image/png"/></manifest><spine toc="ncx"><itemref idref="c1"/></spine></package>`),
		"OEBPS/toc.ncx":          []byte(`<?xml version="1.0"?><ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap><navPoint id="n1" playOrder="1"><navLabel><text>One</text></navLabel><content src="Text/ch1.xhtml"/></navPoint></navMap></ncx>`),
		"OEBPS/Text/ch1.xhtml":   []byte(chapter),
		"OEBPS/Images/cv.png":    png,
	}
	src := writeZip(t, "svg.epub", entries)
	out := convert(t, reg, "epub", src, "mobi")
	path := filepath.Join(t.TempDir(), "out.mobi")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	parser := mobi.NewParser()
	chapters, err := parser.ParseSpine(path)
	if err != nil {
		t.Fatalf("ParseSpine: %v", err)
	}
	if len(chapters) == 0 {
		t.Fatal("no chapters")
	}
	content, err := parser.GetChapterContent(path, chapters[0].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}
	if !strings.Contains(content, "images/kindle-") {
		t.Errorf("svg <image xlink:href> not rebased to kindle record:\n%s", content)
	}
	images, err := parser.ListImages(path)
	if err != nil || len(images) == 0 {
		t.Fatalf("ListImages = %v, %v; want 1 image record", images, err)
	}
	data, err := parser.GetAsset(path, images[0])
	if err != nil {
		t.Fatalf("GetAsset(%q): %v", images[0], err)
	}
	if !bytes.Equal(data, png) {
		t.Error("mobi image record mismatch")
	}
}

// TestFB2SingleTitlePerSection is the regression for "Mục lục Mục lục": in-content h1-h6 must not be promoted to <title> inside a section that already carries the writer's own <title>, which the reader then renders as a duplicated chapter heading.
func TestFB2SingleTitlePerSection(t *testing.T) {
	reg := regressionRegistry(t)
	chapter := `<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Mục lục</title></head><body><h1>Mục lục</h1><p>Chương 1......</p><h2>Phần phụ đề</h2><p>Nội dung phần.</p></body></html>`
	entries := map[string][]byte{
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf":      []byte(`<?xml version="1.0"?><package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Kepub FB2</dc:title><dc:creator>Jane Doe</dc:creator><dc:language>vi</dc:language></metadata><manifest><item id="c1" href="Text/ch1.xhtml" media-type="application/xhtml+xml"/></manifest><spine toc="ncx"><itemref idref="c1"/></spine></package>`),
		"OEBPS/Text/ch1.xhtml":   []byte(chapter),
	}
	src := writeZip(t, "dup.epub", entries)
	out := convert(t, reg, "epub", src, "fb2")
	text := string(out)
	if got := strings.Count(text, "<title>"); got != 1 {
		t.Errorf("fb2 output has %d <title> elements, want exactly 1:\n%s", got, text)
	}
	if !strings.Contains(text, "<section><title><p>Mục lục</p></title>") {
		t.Errorf("writer's own section title missing:\n%s", text)
	}
}
