package ebookconv

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

const (
	palmDOCRecordSize = 4096
	mobiHeaderLen     = 232
)

func writeMOBI(book *bookparser.BookData, images []Image) ([]byte, error) {
	html := mobiHTML(book, images)
	compressed := palmDOCCompress(html)
	title := strings.TrimSpace(fallbackTitle(book))
	if title == "" {
		title = "Untitled"
	}

	chunks := splitRecords(compressed)
	// Record 0 holds the PalmDOC header, MOBI header and title only; the
	// parser's extractText reads records[1..recordCount] for the text.
	// textLength is the UNCOMPRESSED size: the reader compares its
	// decompressed output against it, so a compressed size would truncate
	// the book mid-walk (the "first chapter becomes chapter 3" bug).
	record0 := make([]byte, 0, 16+mobiHeaderLen+len(title))
	record0 = append(record0, palmDOCHeader(uint16(len(chunks)), uint32(len(html)))...)
	record0 = append(record0, mobiHeader(title, uint32(len(chunks)+1))...)
	record0 = append(record0, title...)

	records := make([][]byte, 0, len(chunks)+1+len(images))
	records = append(records, record0)
	for _, chunk := range chunks {
		records = append(records, chunk)
	}
	for _, img := range images {
		records = append(records, img.Data)
	}

	return buildPDB(title, records), nil
}

func palmDOCHeader(recordCount uint16, textLength uint32) []byte {
	h := make([]byte, 16)
	binary.BigEndian.PutUint16(h[0:2], 2) // compression: PalmDOC LZ77
	binary.BigEndian.PutUint32(h[4:8], textLength)
	binary.BigEndian.PutUint16(h[8:10], recordCount)
	binary.BigEndian.PutUint16(h[10:12], palmDOCRecordSize)
	return h
}

func mobiHeader(title string, firstImageIndex uint32) []byte {
	h := make([]byte, mobiHeaderLen)
	copy(h[0:4], "MOBI")
	binary.BigEndian.PutUint32(h[28:32], 65001)
	binary.BigEndian.PutUint32(h[0x54:0x58], 16+mobiHeaderLen)
	binary.BigEndian.PutUint32(h[0x58:0x5c], uint32(len(title)))
	binary.BigEndian.PutUint32(h[0x5c:0x60], firstImageIndex)
	return h
}

func buildPDB(name string, records [][]byte) []byte {
	total := len(records)
	out := make([]byte, 0, 78+total*8)
	hdr := make([]byte, 78)
	copy(hdr[0:32], name)
	copy(hdr[60:64], "BOOK")
	copy(hdr[64:68], "MOBI")
	binary.BigEndian.PutUint16(hdr[76:78], uint16(total))
	out = append(out, hdr...)

	table := make([]byte, total*8)
	offset := 78 + total*8
	for i, rec := range records {
		binary.BigEndian.PutUint32(table[i*8:i*8+4], uint32(offset))
		offset += len(rec)
	}
	out = append(out, table...)
	for _, rec := range records {
		out = append(out, rec...)
	}
	return out
}

func splitRecords(data []byte) [][]byte {
	var chunks [][]byte
	for len(data) > palmDOCRecordSize {
		chunks = append(chunks, data[:palmDOCRecordSize])
		data = data[palmDOCRecordSize:]
	}
	if len(data) > 0 {
		chunks = append(chunks, data)
	}
	return chunks
}

func mobiHTML(book *bookparser.BookData, images []Image) []byte {
	var b strings.Builder
	b.WriteString("<html><head><title>")
	b.WriteString(escapeXML(fallbackTitle(book)))
	b.WriteString("</title></head><body>\n")
	for _, ch := range book.Chapters {
		title := strings.TrimSpace(ch.Title)
		if title == "" {
			title = fallbackTitle(book)
		}
		b.WriteString("<h1>")
		b.WriteString(escapeXML(title))
		b.WriteString("</h1>\n")
		body := rebaseChapterLinks(ch.Content, ch.ContentPath, book.Chapters, func(c bookparser.ChapterData) string {
			return fmt.Sprintf("mobi-section:%d", c.Index)
		})
		b.WriteString(demoteMobiHeadings(rebaseImagesToKindle(body, images)))
		b.WriteString("\n")
	}
	b.WriteString("</body></html>")
	return []byte(b.String())
}

func rebaseImagesToKindle(content string, images []Image) string {
	imgIndex := imageLookup(images)
	nodes := fragmentNodes(content)
	if len(nodes) == 0 {
		return ""
	}
	var rewrite func(n *nethtml.Node)
	rewrite = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			switch n.Data {
			case "img":
				for i, a := range n.Attr {
					if a.Key == "src" {
						if idx, ok := imgIndex[base(a.Val)]; ok {
							n.Attr[i].Val = fmt.Sprintf("images/kindle-%04X.%s", idx+1, mobiImageExt(images[idx]))
						}
					}
				}
			case "image":
				for i, a := range n.Attr {
					if (a.Namespace == "xlink" && a.Key == "href") || a.Key == "xlink:href" {
						if idx, ok := imgIndex[base(a.Val)]; ok {
							n.Attr[i].Val = fmt.Sprintf("images/kindle-%04X.%s", idx+1, mobiImageExt(images[idx]))
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rewrite(c)
		}
	}
	for _, n := range nodes {
		rewrite(n)
	}
	var b strings.Builder
	for _, n := range nodes {
		switch strings.ToLower(n.Data) {
		case "html", "head", "body", "title", "meta", "link", "script", "style":
			continue
		}
		_ = nethtml.Render(&b, n)
	}
	return b.String()
}

// demoteMobiHeadings rewrites h1-h6 inside a chapter fragment to <p> so the
// only headings in the mobi body are the <h1> chapter titles the writer emits
// itself. The reader splits its TOC on heading tags; leaving a chapter's own
// <h2> in place promoted it to a duplicate "chapter" on round-trip.
func demoteMobiHeadings(content string) string {
	nodes := fragmentNodes(content)
	if len(nodes) == 0 {
		return ""
	}
	var rewrite func(n *nethtml.Node)
	rewrite = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			switch strings.ToLower(n.Data) {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				n.Data = "p"
				n.DataAtom = 0
				n.Attr = nil
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rewrite(c)
		}
	}
	for _, n := range nodes {
		rewrite(n)
	}
	var b strings.Builder
	for _, n := range nodes {
		switch strings.ToLower(n.Data) {
		case "html", "head", "body", "title", "meta", "link", "script", "style":
			continue
		}
		_ = nethtml.Render(&b, n)
	}
	return b.String()
}

func mobiImageExt(img Image) string {
	switch {
	case bytes.HasPrefix(img.Data, []byte{0xff, 0xd8, 0xff}):
		return "jpg"
	case bytes.HasPrefix(img.Data, []byte{0x89, 'P', 'N', 'G'}):
		return "png"
	case bytes.HasPrefix(img.Data, []byte("GIF8")):
		return "gif"
	case bytes.HasPrefix(img.Data, []byte{'R', 'I', 'F', 'F'}) && len(img.Data) >= 12 && bytes.Equal(img.Data[8:12], []byte("WEBP")):
		return "webp"
	case bytes.HasPrefix(img.Data, []byte("BM")):
		return "bmp"
	default:
		return imageExt(img.Src)
	}
}

// palmDOCCompress runs the PalmDOC LZ77 back-reference scheme (max distance
// 2047, length 3-10, literal runs ≤8). The format is the inverse of the
// decompressor in pkg/bookparser/mobi.
func palmDOCCompress(data []byte) []byte {
	out := make([]byte, 0, len(data))
	positions := make(map[uint32][]int)
	i := 0
	for i < len(data) {
		bestLen, bestDist := 0, 0
		if i+3 <= len(data) {
			h := hash3(data[i : i+3])
			candidates := positions[h]
			scanned := 0
			for k := len(candidates) - 1; k >= 0 && scanned < 32; k-- {
				scanned++
				pos := candidates[k]
				dist := i - pos
				if dist > 2047 {
					continue
				}
				maxLen := len(data) - i
				if maxLen > 10 {
					maxLen = 10
				}
				l := 1
				for l < maxLen && data[pos+l] == data[i+l] {
					l++
				}
				if l >= 3 && l > bestLen {
					bestLen, bestDist = l, dist
					if l == 10 {
						break
					}
				}
			}
		}
		if bestLen >= 3 {
			value := uint16(bestDist)<<3 | uint16(bestLen-3)
			out = append(out, 0x80|byte(value>>8), byte(value))
			for end := i + bestLen; i < end; i++ {
				if i+3 <= len(data) {
					addPosition(positions, data, i)
				}
			}
			continue
		}
		j := i
		for j < len(data) {
			if j+3 <= len(data) {
				if hasRecentMatch(positions, data, j) && j-i >= 8 {
					break
				}
			}
			j++
		}
		for i < j {
			n := j - i
			if n > 8 {
				n = 8
			}
			out = append(out, byte(n))
			out = append(out, data[i:i+n]...)
			for end := i + n; i < end; i++ {
				if i+3 <= len(data) {
					addPosition(positions, data, i)
				}
			}
		}
	}
	return out
}

func hash3(b []byte) uint32 {
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

func addPosition(positions map[uint32][]int, data []byte, i int) {
	h := hash3(data[i : i+3])
	list := append(positions[h], i)
	if len(list) > 64 {
		list = list[len(list)-32:]
	}
	positions[h] = list
}

func hasRecentMatch(positions map[uint32][]int, data []byte, i int) bool {
	list := positions[hash3(data[i:i+3])]
	for k := len(list) - 1; k >= 0; k-- {
		pos := list[k]
		if i-pos > 2047 {
			continue
		}
		if data[pos] == data[i] && data[pos+1] == data[i+1] && data[pos+2] == data[i+2] {
			return true
		}
	}
	return false
}