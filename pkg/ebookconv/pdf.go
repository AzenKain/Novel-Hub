package ebookconv

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"sort"
	"strings"

	_ "embed"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	nethtml "golang.org/x/net/html"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"novelhub/pkg/bookparser"
)

//go:embed fonts/LiberationSans-Regular.ttf
var embeddedPDFFont []byte

const (
	pdfPageWidth    = 612
	pdfPageHeight   = 792
	pdfMargin       = 72
	pdfBodySize     = 11
	pdfHeaderSize   = 16
	pdfBodyLead     = 14.5
	pdfHeadLead     = 20.0
	pdfMaxImageW    = 216
	pdfMaxImageH    = 648
	pdfImageGap     = 18
	pdfParagraphGap = 8.0
	pdfHeaderGap    = 12.0
)

type pdfLine struct {
	text   string
	size   int
	lead   float64
	img    *pdfImage
	imgIdx int
	iw     float64
	ih     float64
	gap    float64
}

type pdfImage struct {
	src    string
	data   []byte
	filter string
	width  int
	height int
	alpha  []byte
}

func writePDF(book *bookparser.BookData, images []Image) ([]byte, error) {
	f, err := sfnt.Parse(embeddedPDFFont)
	if err != nil {
		return nil, fmt.Errorf("parse embedded pdf font: %w", err)
	}
	w := &pdfWriter{
		font:     f,
		buf:      new(sfnt.Buffer),
		upem:     int(f.UnitsPerEm()),
		glyphIDs: make(map[rune]uint16),
		widths:   make(map[uint16]int),
		unicode:  make(map[uint16]rune),
	}
	return w.render(book, images)
}

type pdfWriter struct {
	font     *sfnt.Font
	buf      *sfnt.Buffer
	upem     int
	glyphIDs map[rune]uint16
	widths   map[uint16]int
	unicode  map[uint16]rune
}

func (w *pdfWriter) gid(r rune) uint16 {
	if id, ok := w.glyphIDs[r]; ok {
		return id
	}
	index, _ := w.font.GlyphIndex(w.buf, r)
	id := uint16(index)
	if id == 0 {
		space, _ := w.font.GlyphIndex(w.buf, ' ')
		id = uint16(space)
		w.glyphIDs[r] = id
		return id
	}
	w.glyphIDs[r] = id
	if _, ok := w.widths[id]; !ok {
		adv, _ := w.font.GlyphAdvance(w.buf, index, fixed.I(w.upem), font.HintingNone)
		w.widths[id] = int(adv)>>6*1000 + w.upem/2
		w.widths[id] /= w.upem
		w.unicode[id] = r
	}
	return id
}

func (w *pdfWriter) lineWidth(text string, size int) float64 {
	pt := float64(size) / 1000.0
	sum := 0.0
	for _, r := range text {
		sum += float64(w.widths[w.gid(r)]) * pt
	}
	return sum
}

func (w *pdfWriter) wrap(text string, size int) []string {
	maxW := float64(pdfPageWidth - 2*pdfMargin)
	spaceW := w.lineWidth(" ", size)
	var out []string
	var cur strings.Builder
	curW := 0.0
	for _, word := range strings.Fields(text) {
		ww := w.lineWidth(word, size)
		if cur.Len() > 0 {
			if curW+spaceW+ww > maxW {
				out = append(out, cur.String())
				cur.Reset()
				curW = 0
			} else {
				cur.WriteString(" ")
				curW += spaceW
			}
		}
		cur.WriteString(word)
		curW += ww
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func (w *pdfWriter) render(book *bookparser.BookData, images []Image) ([]byte, error) {
	imgs := make([]*pdfImage, 0, len(images))
	imgByPos := make(map[string]int)
	for _, img := range images {
		p, err := resolvePDFImage(img)
		if err != nil {
			continue
		}
		imgByPos[base(p.src)] = len(imgs)
		imgs = append(imgs, p)
	}

	var pages [][]pdfLine
	var page []pdfLine
	pageBottom := float64(pdfPageHeight - pdfMargin)
	pageTop := float64(pdfMargin)
	used := 0.0
	push := func(l pdfLine) {
		if used > 0 && used+l.lead+l.gap > pageBottom-pageTop {
			pages = append(pages, page)
			page = nil
			used = 0
		}
		page = append(page, l)
		used += l.lead + l.gap
	}
	imageLine := func(p *pdfImage, idx int) pdfLine {
		scale := pdfMaxImageW / float64(p.width)
		if h := float64(p.height) * scale; h > pdfMaxImageH {
			scale = pdfMaxImageH / float64(p.height)
		}
		return pdfLine{img: p, imgIdx: idx, lead: float64(p.height) * scale, iw: float64(p.width) * scale, ih: float64(p.height) * scale}
	}

	for _, ch := range book.Chapters {
		title := strings.TrimSpace(ch.Title)
		if title == "" {
			title = fallbackTitle(book)
		}
		headerLines := w.wrap(title, pdfHeaderSize)
		for i, line := range headerLines {
			gap := 0.0
			if i == len(headerLines)-1 {
				gap = pdfHeaderGap
			}
			push(pdfLine{text: line, size: pdfHeaderSize, lead: pdfHeadLead, gap: gap})
		}
		var cur strings.Builder
		flush := func() {
			t := strings.TrimSpace(cur.String())
			cur.Reset()
			if t == "" {
				return
			}
			wrapped := w.wrap(t, pdfBodySize)
			for i, line := range wrapped {
				gap := 0.0
				if i == len(wrapped)-1 {
					gap = pdfParagraphGap
				}
				push(pdfLine{text: line, size: pdfBodySize, lead: pdfBodyLead, gap: gap})
			}
		}
		var walk func(n *nethtml.Node)
		walk = func(n *nethtml.Node) {
			switch n.Type {
			case nethtml.TextNode:
				cur.WriteString(n.Data)
			case nethtml.ElementNode:
				switch strings.ToLower(n.Data) {
				case "img":
					flush()
					if i, ok := imgByPos[base(fileAttr(n, "src"))]; ok {
						push(imageLine(imgs[i], i))
					}
				case "br":
					flush()
				case "p", "div", "article", "section", "figure", "figcaption", "blockquote", "li", "pre",
					"h1", "h2", "h3", "h4", "h5", "h6":
					flush()
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						walk(c)
					}
					flush()
				case "script", "style", "head", "title":
					return
				default:
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						walk(c)
					}
				}
			}
		}
		for _, n := range fragmentNodes(ch.Content) {
			walk(n)
		}
		flush()
	}
	if len(page) > 0 {
		pages = append(pages, page)
	}
	if len(pages) == 0 {
		pages = [][]pdfLine{{}}
	}
	return w.buildPDF(pages, imgs)
}

func resolvePDFImage(img Image) (*pdfImage, error) {
	if bytes.HasPrefix(img.Data, []byte{0xff, 0xd8}) {
		cfg, err := decodeImageConfig(img.Data)
		if err != nil {
			return nil, err
		}
		return &pdfImage{src: img.Src, data: img.Data, filter: "/DCTDecode", width: cfg.Width, height: cfg.Height}, nil
	}
	src, _, err := image.Decode(bytes.NewReader(img.Data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	opaque := true
	if o, ok := src.(interface{ Opaque() bool }); ok {
		opaque = o.Opaque()
	}
	rgb := make([]byte, 0, w*h*3)
	var alpha []byte
	if !opaque {
		alpha = make([]byte, 0, w*h)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			rgb = append(rgb, byte(r>>8), byte(g>>8), byte(bl>>8))
			if !opaque {
				alpha = append(alpha, byte(a>>8))
			}
		}
	}
	return &pdfImage{src: img.Src, data: flate(rgb), filter: "/FlateDecode", width: w, height: h, alpha: flateOrNil(alpha)}, nil
}

func decodeImageConfig(data []byte) (image.Config, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	return cfg, err
}

func flate(data []byte) []byte {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	zw.Write(data)
	zw.Close()
	return buf.Bytes()
}

func flateOrNil(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	return flate(data)
}

func (w *pdfWriter) buildPDF(pages [][]pdfLine, imgs []*pdfImage) ([]byte, error) {
	P := len(pages)
	numFontFile := 3 + 2*P
	numDescriptor := numFontFile + 1
	numCIDFont := numDescriptor + 1
	numType0 := numCIDFont + 1
	numToUnicode := numType0 + 1
	imgObj := make([]int, len(imgs))
	total := numToUnicode + 1
	for i := range imgs {
		imgObj[i] = total
		total++
		if len(imgs[i].alpha) > 0 {
			total++
		}
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	var offsets []int
	offsets = append(offsets, 0)
	writeObj := func(body string) int {
		offsets = append(offsets, out.Len())
		num := len(offsets) - 1
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", num, body)
		return num
	}

	catalog := "<< /Type /Catalog /Pages 2 0 R >>"
	writeObj(catalog)
	kids := make([]string, P)
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", 4+2*i)
	}
	writeObj(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), P))

	startY := float64(pdfPageHeight - pdfMargin)
	for _, items := range pages {
		xo := make(map[int]string)
		for _, it := range items {
			if it.img != nil {
				xo[it.imgIdx] = fmt.Sprintf("/Im%d %d 0 R", it.imgIdx, imgObj[it.imgIdx])
			}
		}
		var xoStr strings.Builder
		for _, s := range xo {
			xoStr.WriteString(s)
			xoStr.WriteString(" ")
		}
		var content strings.Builder
		y := startY
		inText := false
		needPos := true
		lastSize := 0
		prevGap := 0.0
		closeText := func() {
			if inText {
				content.WriteString("ET\n")
				inText = false
			}
		}
		for _, it := range items {
			if it.img != nil {
				closeText()
				y -= pdfImageGap
				y -= it.ih
				fmt.Fprintf(&content, "q\n%.1f 0 0 %.1f 72 %.2f cm\n/Im%d Do\nQ\n", it.iw, it.ih, y, it.imgIdx)
				y -= pdfImageGap
				needPos = true
				prevGap = 0.0
				continue
			}
			if !inText {
				content.WriteString("BT\n")
				inText = true
				lastSize = 0
			}
			if it.size != lastSize {
				fmt.Fprintf(&content, "/F1 %d Tf\n", it.size)
				lastSize = it.size
			}
			if needPos {
				y -= it.lead
				fmt.Fprintf(&content, "72 %.1f Td\n", y)
				needPos = false
			} else {
				dy := it.lead + prevGap
				fmt.Fprintf(&content, "0 -%.1f Td\n", dy)
				y -= dy
			}
			prevGap = it.gap
			content.WriteString("<")
			content.WriteString(w.hexGIDs(it.text))
			content.WriteString("> Tj\n")
		}
		closeText()
		contentObj := writeObj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String()))
		pageObj := fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 %d 0 R >> /XObject << %s>> >> /Contents %d 0 R >>",
			pdfPageWidth, pdfPageHeight, numType0, xoStr.String(), contentObj)
		writeObj(pageObj)
	}

	writeObj(fmt.Sprintf("<< /Length %d /Length1 %d >>\nstream\n", len(embeddedPDFFont), len(embeddedPDFFont)) + string(embeddedPDFFont) + "\nendstream")

	metrics, err := w.font.Metrics(w.buf, fixed.I(w.upem), font.HintingNone)
	if err != nil {
		return nil, err
	}
	bounds, err := w.font.Bounds(w.buf, fixed.I(w.upem), font.HintingNone)
	if err != nil {
		return nil, err
	}
	ascent := int(metrics.Ascent) >> 6
	descent := -(int(metrics.Descent) >> 6)
	capHeight := os2CapHeight(embeddedPDFFont, ascent)
	descriptor := fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /LiberationSans /Flags 4 /FontBBox [%d %d %d %d] /ItalicAngle 0 /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %d 0 R >>",
		int(bounds.Min.X)>>6, int(bounds.Min.Y)>>6, int(bounds.Max.X)>>6, int(bounds.Max.Y)>>6,
		ascent, descent, capHeight, numFontFile)
	writeObj(descriptor)

	descendant := fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /LiberationSans /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor %d 0 R /W [%s] /CIDToGIDMap /Identity >>",
		numDescriptor, w.widthRuns())
	writeObj(descendant)

	type0 := fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /LiberationSans /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>",
		numCIDFont, numToUnicode)
	writeObj(type0)

	cmap := w.toUnicodeCMap()
	writeObj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(cmap), cmap))

	for i, im := range imgs {
		imgBody := func(smask int) string {
			if smask > 0 {
				return fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter %s /SMask %d 0 R /Length %d >>\nstream\n%s\nendstream",
					im.width, im.height, im.filter, smask, len(im.data), im.data)
			}
			return fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter %s /Length %d >>\nstream\n%s\nendstream",
				im.width, im.height, im.filter, len(im.data), im.data)
		}
		if len(im.alpha) > 0 {
			writeObj(imgBody(imgObj[i] + 1))
			writeObj(fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream",
				im.width, im.height, len(im.alpha), im.alpha))
		} else {
			writeObj(imgBody(0))
		}
	}

	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", total)
	fmt.Fprintf(&out, "0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	for i := len(offsets); i < total; i++ {
		fmt.Fprintf(&out, "0000000000 00000 n \n")
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", total, xref)

	return out.Bytes(), nil
}

func (w *pdfWriter) hexGIDs(text string) string {
	var b strings.Builder
	for _, r := range text {
		fmt.Fprintf(&b, "%04X", w.gid(r))
	}
	return b.String()
}

func (w *pdfWriter) widthRuns() string {
	gids := make([]int, 0, len(w.widths))
	for g := range w.widths {
		gids = append(gids, int(g))
	}
	sort.Ints(gids)
	var b strings.Builder
	for i := 0; i < len(gids); {
		j := i
		for j+1 < len(gids) && gids[j+1] == gids[j]+1 {
			j++
		}
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%d [", gids[i])
		for k := i; k <= j; k++ {
			if k > i {
				b.WriteString(" ")
			}
			fmt.Fprintf(&b, "%d", w.widths[uint16(gids[k])])
		}
		b.WriteString("]")
		i = j + 1
	}
	return b.String()
}

func (w *pdfWriter) toUnicodeCMap() string {
	gids := make([]int, 0, len(w.unicode))
	for g := range w.unicode {
		gids = append(gids, int(g))
	}
	sort.Ints(gids)
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	fmt.Fprintf(&b, "%d beginbfchar\n", len(gids))
	for _, g := range gids {
		fmt.Fprintf(&b, "<%04X> <%s>\n", g, utf16Hex(w.unicode[uint16(g)]))
	}
	b.WriteString("endbfchar\nendcmap\nCMapName currentdict /CMap defineresource pop\nend\nend")
	return b.String()
}

func utf16Hex(r rune) string {
	if r <= 0xFFFF {
		return fmt.Sprintf("%04X", r)
	}
	r1 := rune(0xD800 + (r-0x10000)>>10)
	r2 := rune(0xDC00 + (r-0x10000)&0x3FF)
	return fmt.Sprintf("%04X%04X", r1, r2)
}

func os2CapHeight(ttf []byte, fallback int) int {
	table := ttfTable(ttf, []byte("OS/2"))
	if len(table) < 100 {
		return fallback
	}
	version := binary.BigEndian.Uint16(table[0:2])
	if version < 2 {
		return fallback
	}
	v := int(int16(binary.BigEndian.Uint16(table[98:100])))
	if v > 0 {
		return v
	}
	return fallback
}

func ttfTable(data []byte, tag []byte) []byte {
	if len(data) < 12 {
		return nil
	}
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	for i := 0; i < numTables; i++ {
		entry := data[12+i*16 : 12+i*16+16]
		if bytes.Equal(entry[0:4], tag) {
			offset := int(binary.BigEndian.Uint32(entry[8:12]))
			length := int(binary.BigEndian.Uint32(entry[12:16]))
			if offset+length <= len(data) {
				return data[offset : offset+length]
			}
		}
	}
	return nil
}
