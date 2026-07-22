package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"html"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"novelhub/pkg/bookparser"
)

type Parser struct{}

type mobiSection struct {
	Title   string
	Content string
}

type mobiAsset struct {
	Name        string
	Data        []byte
	Ext         string
	RecordIndex int
	KindleIndex int
}

type palmDocHeader struct {
	Compression uint16
	TextLength  uint32
	RecordCount uint16
	RecordSize  uint16
	Encoding    uint32
	Title       string
}

var (
	bodyContentRegex        = regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	headBlockRegex          = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	scriptBlockRegex        = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	xmlDeclRegex            = regexp.MustCompile(`(?is)<\?xml[^>]*>`)
	htmlBodyTagRegex        = regexp.MustCompile(`(?is)</?(html|body)[^>]*>`)
	htmlTagRegex            = regexp.MustCompile(`(?is)<[^>]+>`)
	emptyParagraphRegex     = regexp.MustCompile(`(?is)<p(?:\s+[a-z][a-z0-9_-]*\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]*))*\s*>(?:\s|&nbsp;|<br\s*/?>)*</p>`)
	headingRegex            = regexp.MustCompile(`(?is)<h[1-3]\b[^>]*>(.*?)</h[1-3]>`)
	fileposLinkRegex        = regexp.MustCompile(`(?is)<a\b[^>]*\bfilepos\s*=\s*["']?([0-9]+)["']?[^>]*>(.*?)</a>`)
	kindleImageRegex        = regexp.MustCompile(`(?is)<img\b[^>]*(?:src|href)=["']kindle:[^"']*["'][^>]*>`)
	kindleAttrRegex         = regexp.MustCompile(`(?is)\s+(?:src|href)=["']kindle:[^"']*["']`)
	kindleRefRegex          = regexp.MustCompile(`(?i)kindle:(?:embed|flow):([0-9a-f]+)`)
	brokenAttrTextRegex     = regexp.MustCompile(`(?im)(^|[\s>])(?:[a-z][a-z0-9:_-]{0,31})\s*=\s*(?:"[^"\n<>]{0,160}"|'[^'\n<>]{0,160}')>`)
	escapedEmptyAnchorRegex = regexp.MustCompile(`(?is)&lt;a\b[^&]*&gt;\s*&lt;/a&gt;`)
)

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read mobi metadata: %w", err)
	}
	header, err := parseHeader(data, filePath)
	if err != nil {
		return &bookparser.BookMetadata{Title: bookparser.TitleFromPath(filePath)}, nil
	}
	title := strings.TrimSpace(header.Title)
	if title == "" {
		title = bookparser.TitleFromPath(filePath)
	}
	return bookparser.MergeMetadataSidecar(filePath, &bookparser.BookMetadata{Title: title}), nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	meta, _ := p.ParseMetadata(filePath)
	title := bookparser.TitleFromPath(filePath)
	if meta != nil && strings.TrimSpace(meta.Title) != "" {
		title = meta.Title
	}
	readerHTML, err := p.readerHTML(filePath)
	if err != nil {
		return nil, err
	}
	sections := splitMobiSections(readerHTML, title)
	if len(sections) == 0 {
		return []bookparser.ChapterData{{
			Title:       title,
			ContentPath: "mobi-document",
			Index:       0,
		}}, nil
	}
	chapters := make([]bookparser.ChapterData, 0, len(sections))
	for index, section := range sections {
		chapters = append(chapters, bookparser.ChapterData{
			Title:       section.Title,
			ContentPath: "mobi-section:" + strconv.Itoa(index),
			Index:       index,
		})
	}
	return chapters, nil
}

func (p *Parser) ParseBook(filePath string) (*bookparser.BookData, error) {
	meta, err := p.ParseMetadata(filePath)
	if err != nil {
		return nil, err
	}
	chapters, err := p.ParseSpine(filePath)
	if err != nil {
		return nil, err
	}
	for i := range chapters {
		content, err := p.GetChapterContent(filePath, chapters[i].ContentPath)
		if err == nil {
			chapters[i].Content = content
		}
	}
	return &bookparser.BookData{Metadata: *meta, Chapters: chapters}, nil
}

func (p *Parser) GetChapterContent(filePath, contentPath string) (string, error) {
	readerHTML, err := p.readerHTML(filePath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(contentPath, "mobi-section:") {
		index, err := strconv.Atoi(strings.TrimPrefix(contentPath, "mobi-section:"))
		if err != nil {
			return "", fmt.Errorf("invalid mobi section path %q", contentPath)
		}
		meta, _ := p.ParseMetadata(filePath)
		title := bookparser.TitleFromPath(filePath)
		if meta != nil && strings.TrimSpace(meta.Title) != "" {
			title = meta.Title
		}
		sections := splitMobiSections(readerHTML, title)
		if index < 0 || index >= len(sections) {
			return "", fmt.Errorf("mobi section %d not found", index)
		}
		return normalizeMobiSectionHTML(sections[index].Content), nil
	}
	return normalizeMobiSectionHTML(readerHTML), nil
}

func (p *Parser) readerHTML(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read mobi content: %w", err)
	}
	header, text, err := extractText(data, filePath)
	if err != nil {
		return unsupportedHTML(filePath, err), nil
	}
	value := decodeText(text, header.Encoding)
	if looksLikeHTML(value) {
		readerHTML := htmlDocumentToReaderHTML(value)
		if readableHTMLTextLength(readerHTML) < 32 {
			return unsupportedHTML(filePath, fmt.Errorf("no readable text found in decoded MOBI/AZW payload")), nil
		}
		return readerHTML, nil
	}
	return bookparser.PlainTextToHTML(value), nil
}

func fallbackMobiSpine(title string) []bookparser.ChapterData {
	return []bookparser.ChapterData{{
		Title:       title,
		ContentPath: "mobi-document",
		Index:       0,
	}}
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assets, err := readMobiAssets(filePath)
	if err != nil {
		return nil, err
	}
	assetPath = normalizeMobiAssetPath(assetPath)
	if assetPath == "" {
		return nil, fmt.Errorf("invalid mobi asset path")
	}

	if kindleIndex, ok := mobiKindleRefIndex(assetPath); ok {
		for _, asset := range assets {
			if asset.KindleIndex == kindleIndex {
				return asset.Data, nil
			}
		}
		if kindleIndex > 0 && kindleIndex <= len(assets) {
			return assets[kindleIndex-1].Data, nil
		}
	}

	for _, asset := range assets {
		if mobiAssetNameMatches(asset, assetPath) {
			return asset.Data, nil
		}
	}
	return nil, fmt.Errorf("mobi asset %q not found", assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	assets, err := readMobiAssets(filePath)
	if err != nil {
		return nil, err
	}
	images := make([]string, 0, len(assets))
	for _, asset := range assets {
		images = append(images, asset.Name)
	}
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func readMobiAssets(filePath string) ([]mobiAsset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read mobi assets: %w", err)
	}
	records, err := recordSlices(data)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []mobiAsset{}, nil
	}

	firstImageIndex := mobiFirstImageIndex(records[0])
	assets := make([]mobiAsset, 0)
	for recordIndex, record := range records {
		ext, ok := detectMobiImageExt(record)
		if !ok {
			continue
		}
		kindleIndex := 0
		if firstImageIndex >= 0 && recordIndex >= firstImageIndex {
			kindleIndex = recordIndex - firstImageIndex + 1
		}
		name := mobiAssetName(recordIndex, kindleIndex, ext)
		assets = append(assets, mobiAsset{
			Name:        name,
			Data:        record,
			Ext:         ext,
			RecordIndex: recordIndex,
			KindleIndex: kindleIndex,
		})
	}
	return assets, nil
}

func mobiFirstImageIndex(first []byte) int {
	mobiStart := bytes.Index(first, []byte("MOBI"))
	if mobiStart < 0 || len(first) < mobiStart+0x60 {
		return -1
	}
	value := binary.BigEndian.Uint32(first[mobiStart+0x5c : mobiStart+0x60])
	if value == 0 || value == 0xffffffff {
		return -1
	}
	return int(value)
}

func detectMobiImageExt(record []byte) (string, bool) {
	if bytes.HasPrefix(record, []byte{0xff, 0xd8, 0xff}) {
		return "jpg", true
	}
	if bytes.HasPrefix(record, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "png", true
	}
	if bytes.HasPrefix(record, []byte("GIF8")) {
		return "gif", true
	}
	if len(record) >= 12 && bytes.Equal(record[:4], []byte("RIFF")) && bytes.Equal(record[8:12], []byte("WEBP")) {
		return "webp", true
	}
	if bytes.HasPrefix(record, []byte("BM")) {
		return "bmp", true
	}
	trimmed := bytes.TrimLeft(record, "\x00\t\r\n ")
	probeLength := len(trimmed)
	if probeLength > 512 {
		probeLength = 512
	}
	probe := bytes.ToLower(trimmed[:probeLength])
	if bytes.HasPrefix(probe, []byte("<svg")) || (bytes.HasPrefix(probe, []byte("<?xml")) && bytes.Contains(probe, []byte("<svg"))) {
		return "svg", true
	}
	return "", false
}

func mobiAssetName(recordIndex int, kindleIndex int, ext string) string {
	if kindleIndex > 0 {
		return fmt.Sprintf("images/kindle-%04X.%s", kindleIndex, ext)
	}
	return fmt.Sprintf("images/record-%04d.%s", recordIndex, ext)
}

func normalizeMobiAssetPath(assetPath string) string {
	assetPath = strings.TrimSpace(assetPath)
	if decoded, err := url.PathUnescape(assetPath); err == nil {
		assetPath = decoded
	}
	assetPath = strings.ReplaceAll(assetPath, "\\", "/")
	assetPath = strings.TrimLeft(assetPath, "/")
	return assetPath
}

func mobiKindleRefIndex(assetPath string) (int, bool) {
	match := kindleRefRegex.FindStringSubmatch(assetPath)
	if len(match) < 2 {
		return 0, false
	}
	value, err := strconv.ParseInt(match[1], 16, 32)
	if err != nil || value <= 0 {
		return 0, false
	}
	return int(value), true
}

func mobiAssetNameMatches(asset mobiAsset, assetPath string) bool {
	pathWithoutQuery, _, _ := strings.Cut(assetPath, "?")
	pathWithoutFragment, _, _ := strings.Cut(pathWithoutQuery, "#")
	normalized := strings.ToLower(strings.TrimLeft(pathWithoutFragment, "/"))
	name := strings.ToLower(asset.Name)
	if normalized == name {
		return true
	}
	if strings.TrimPrefix(normalized, "images/") == strings.TrimPrefix(name, "images/") {
		return true
	}
	recordName := fmt.Sprintf("images/record-%04d.%s", asset.RecordIndex, asset.Ext)
	return normalized == strings.ToLower(recordName) ||
		strings.TrimPrefix(normalized, "images/") == strings.TrimPrefix(strings.ToLower(recordName), "images/")
}

func parseHeader(data []byte, filePath string) (*palmDocHeader, error) {
	records, err := recordSlices(data)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 || len(records[0]) < 16 {
		return nil, fmt.Errorf("missing PalmDOC header")
	}
	first := records[0]
	header := &palmDocHeader{
		Compression: binary.BigEndian.Uint16(first[0:2]),
		TextLength:  binary.BigEndian.Uint32(first[4:8]),
		RecordCount: binary.BigEndian.Uint16(first[8:10]),
		RecordSize:  binary.BigEndian.Uint16(first[10:12]),
		Title:       palmDatabaseName(data),
	}
	if mobiStart := bytes.Index(first, []byte("MOBI")); mobiStart >= 0 {
		if len(first) >= mobiStart+32 {
			header.Encoding = binary.BigEndian.Uint32(first[mobiStart+28 : mobiStart+32])
		}
		if title := mobiFullName(first, mobiStart); title != "" {
			header.Title = title
		}
	}
	if header.Title == "" {
		header.Title = bookparser.TitleFromPath(filePath)
	}
	return header, nil
}

func extractText(data []byte, filePath string) (*palmDocHeader, []byte, error) {
	records, err := recordSlices(data)
	if err != nil {
		return nil, nil, err
	}
	header, err := parseHeader(data, filePath)
	if err != nil {
		return nil, nil, err
	}
	recordCount := int(header.RecordCount)
	if recordCount <= 0 || recordCount > len(records)-1 {
		recordCount = len(records) - 1
	}
	var out []byte
	for i := 1; i <= recordCount; i++ {
		record := records[i]
		var part []byte
		switch header.Compression {
		case 1:
			part = record
		case 2:
			part, err = decompressPalmDOC(record)
			if err != nil {
				return header, nil, err
			}
		default:
			return header, nil, fmt.Errorf("unsupported MOBI compression %d", header.Compression)
		}
		out = append(out, part...)
		if header.TextLength > 0 && uint32(len(out)) >= header.TextLength {
			out = out[:header.TextLength]
			break
		}
	}
	return header, out, nil
}

func recordSlices(data []byte) ([][]byte, error) {
	if len(data) < 78 {
		return nil, fmt.Errorf("invalid PalmDB header")
	}
	recordCount := int(binary.BigEndian.Uint16(data[76:78]))
	if recordCount == 0 {
		return nil, fmt.Errorf("PalmDB has no records")
	}
	tableEnd := 78 + recordCount*8
	if len(data) < tableEnd {
		return nil, fmt.Errorf("truncated PalmDB record table")
	}
	offsets := make([]int, recordCount+1)
	for i := 0; i < recordCount; i++ {
		offset := int(binary.BigEndian.Uint32(data[78+i*8 : 82+i*8]))
		if offset < tableEnd || offset > len(data) {
			return nil, fmt.Errorf("invalid PalmDB record offset")
		}
		if i > 0 && offset < offsets[i-1] {
			return nil, fmt.Errorf("PalmDB record offsets are not ordered")
		}
		offsets[i] = offset
	}
	offsets[recordCount] = len(data)
	records := make([][]byte, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		if offsets[i+1] < offsets[i] {
			return nil, fmt.Errorf("invalid PalmDB record range")
		}
		records = append(records, data[offsets[i]:offsets[i+1]])
	}
	return records, nil
}

func decompressPalmDOC(data []byte) ([]byte, error) {
	out := make([]byte, 0, len(data)*2)
	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case c >= 1 && c <= 8:
			count := int(c)
			if i+1+count > len(data) {
				out = append(out, data[i+1:]...)
				return out, nil
			}
			out = append(out, data[i+1:i+1+count]...)
			i += count
		case c <= 0x7f:
			out = append(out, c)
		case c <= 0xbf:
			if i+1 >= len(data) {
				return out, nil
			}
			next := data[i+1]
			i++
			value := (uint16(c&0x3f) << 8) | uint16(next)
			distance := int(value >> 3)
			length := int(value&0x07) + 3
			if distance == 0 || distance > len(out) {
				continue
			}
			for j := 0; j < length; j++ {
				out = append(out, out[len(out)-distance])
			}
		default:
			out = append(out, ' ', c^0x80)
		}
	}
	return out, nil
}

func palmDatabaseName(data []byte) string {
	if len(data) < 32 {
		return ""
	}
	return cleanText(string(bytes.Trim(data[:32], "\x00 ")))
}

func mobiFullName(first []byte, mobiStart int) string {
	if len(first) < mobiStart+0x5c {
		return ""
	}
	offset := int(binary.BigEndian.Uint32(first[mobiStart+0x54 : mobiStart+0x58]))
	length := int(binary.BigEndian.Uint32(first[mobiStart+0x58 : mobiStart+0x5c]))
	if offset <= 0 || length <= 0 || offset+length > len(first) {
		return ""
	}
	return cleanText(string(first[offset : offset+length]))
}

func decodeText(data []byte, encoding uint32) string {
	text := string(data)
	if encoding == 65001 || encoding == 0 {
		return strings.ToValidUTF8(text, "")
	}
	// Most MOBI files in small personal libraries are UTF-8 or Windows-1252.
	// We keep bytes readable without pulling in a full charset dependency.
	return strings.ToValidUTF8(text, "")
}

func looksLikeHTML(value string) bool {
	probe := strings.ToLower(value)
	if len(probe) > 4096 {
		probe = probe[:4096]
	}
	return strings.Contains(probe, "<html") ||
		strings.Contains(probe, "<body") ||
		strings.Contains(probe, "<p") ||
		strings.Contains(probe, "<div") ||
		strings.Contains(probe, "<mbp:")
}

func htmlDocumentToReaderHTML(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = scriptBlockRegex.ReplaceAllString(value, "")
	value = headBlockRegex.ReplaceAllString(value, "")
	full := value
	matches := bodyContentRegex.FindAllStringSubmatch(value, -1)
	if len(matches) > 0 {
		body := ""
		for _, match := range matches {
			if len(match) > 1 && len(strings.TrimSpace(match[1])) > len(strings.TrimSpace(body)) {
				body = match[1]
			}
		}
		value = body
		if readableHTMLTextLength(value) < 32 {
			value = htmlFragmentsToReaderHTML(full)
		}
	} else {
		value = htmlFragmentsToReaderHTML(full)
	}
	value = strings.ReplaceAll(value, "<mbp:pagebreak/>", "")
	value = strings.ReplaceAll(value, "<mbp:pagebreak />", "")
	value = kindleImageRegex.ReplaceAllString(value, "")
	value = kindleAttrRegex.ReplaceAllString(value, "")
	value = stripBrokenMobiTextNodes(value)
	return `<article class="novelhub-mobi">` + strings.TrimSpace(value) + `</article>`
}

func splitMobiSections(readerHTML string, fallbackTitle string) []mobiSection {
	inner := articleInnerHTML(readerHTML)
	if sections := splitMobiSectionsByHeadings(inner, fallbackTitle); len(sections) > 0 {
		return sections
	}
	return splitMobiSectionsByFilepos(inner, fallbackTitle)
}

func splitMobiSectionsByHeadings(inner string, fallbackTitle string) []mobiSection {
	matches := headingRegex.FindAllStringSubmatchIndex(inner, -1)
	if len(matches) < 2 {
		return nil
	}

	all := make([]mobiSection, 0, len(matches))
	contentEnd := len(inner)
	for index, match := range matches {
		if len(match) < 4 || match[2] < 0 || match[3] < 0 {
			continue
		}
		start := match[0]
		end := contentEnd
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		title := cleanHTMLText(inner[match[2]:match[3]])
		if title == "" {
			title = fallbackTitle
		}
		content := strings.TrimSpace(inner[start:end])
		if readableHTMLTextLength(content) < 24 {
			continue
		}
		all = append(all, mobiSection{
			Title:   title,
			Content: `<article class="novelhub-mobi">` + content + `</article>`,
		})
	}
	if len(all) < 2 {
		return nil
	}

	chapters := make([]mobiSection, 0, len(all))
	for _, section := range all {
		if isMobiNavigationTitle(section.Title) {
			continue
		}
		chapters = append(chapters, section)
	}
	if len(chapters) >= 2 {
		return chapters
	}
	return all
}

type mobiFileposPoint struct {
	Title  string
	Offset int
	Start  int
}

func splitMobiSectionsByFilepos(inner string, fallbackTitle string) []mobiSection {
	matches := fileposLinkRegex.FindAllStringSubmatch(inner, -1)
	if len(matches) < 2 {
		return nil
	}

	points := make([]mobiFileposPoint, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		title := cleanHTMLText(match[2])
		if title == "" || isMobiNavigationTitle(title) {
			continue
		}
		offset, err := strconv.Atoi(strings.TrimLeft(match[1], "0"))
		if err != nil {
			offset = 0
		}
		if offset <= 0 || offset >= len(inner) {
			continue
		}
		start, ok := mobiFileposSectionStart(inner, offset, title)
		if !ok {
			continue
		}
		key := strconv.Itoa(start) + "\x00" + strings.ToLower(title)
		if seen[key] {
			continue
		}
		seen[key] = true
		points = append(points, mobiFileposPoint{Title: title, Offset: offset, Start: start})
	}
	if len(points) < 2 {
		return nil
	}

	sort.SliceStable(points, func(i, j int) bool {
		if points[i].Start == points[j].Start {
			return points[i].Offset < points[j].Offset
		}
		return points[i].Start < points[j].Start
	})

	ordered := points[:0]
	lastStart := -1
	for _, point := range points {
		if point.Start <= lastStart {
			continue
		}
		ordered = append(ordered, point)
		lastStart = point.Start
	}
	if len(ordered) < 2 {
		return nil
	}

	sections := make([]mobiSection, 0, len(ordered))
	for index, point := range ordered {
		end := len(inner)
		if index+1 < len(ordered) {
			end = ordered[index+1].Start
		}
		if point.Start < 0 || point.Start >= end || end > len(inner) {
			continue
		}
		title := point.Title
		if title == "" {
			title = fallbackTitle
		}
		content := strings.TrimSpace(inner[point.Start:end])
		if readableHTMLTextLength(content) < 24 {
			continue
		}
		sections = append(sections, mobiSection{
			Title:   title,
			Content: `<article class="novelhub-mobi">` + content + `</article>`,
		})
	}
	if len(sections) < 2 {
		return nil
	}
	return sections
}

func mobiFileposSectionStart(inner string, offset int, title string) (int, bool) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(inner) {
		offset = len(inner)
	}
	for _, window := range []int{1200, 4000, 12000} {
		start := offset - window
		if start < 0 {
			start = 0
		}
		end := offset + window
		if end > len(inner) {
			end = len(inner)
		}
		if start >= end {
			continue
		}
		if rel := strings.Index(inner[start:end], title); rel >= 0 {
			return mobiNormalizeSectionStart(inner, mobiNearestBlockStart(inner, start+rel)), true
		}
	}
	return mobiNormalizeSectionStart(inner, mobiNearestBlockStart(inner, offset)), true
}

func mobiNearestBlockStart(inner string, index int) int {
	if index < 0 {
		return 0
	}
	if index > len(inner) {
		index = len(inner)
	}
	searchStart := index - 800
	if searchStart < 0 {
		searchStart = 0
	}
	probe := strings.ToLower(inner[searchStart:index])
	best := -1
	for _, tag := range []string{"<h1", "<h2", "<h3", "<p", "<div", "<section", "<article", "<blockquote"} {
		if rel := strings.LastIndex(probe, tag); rel >= 0 && searchStart+rel > best {
			best = searchStart + rel
		}
	}
	if best >= 0 {
		return best
	}
	if rel := strings.LastIndex(inner[:index], "<"); rel >= 0 && index-rel <= 300 {
		return rel
	}
	return index
}

func mobiNormalizeSectionStart(inner string, start int) int {
	if start <= 0 {
		return 0
	}
	if start >= len(inner) {
		return len(inner)
	}
	if inner[start] == '<' {
		return start
	}

	searchStart := start - 160
	if searchStart < 0 {
		searchStart = 0
	}
	probe := inner[searchStart:start]
	if rel := strings.LastIndex(probe, "<"); rel >= 0 {
		candidate := searchStart + rel
		segment := inner[candidate:start]
		if !strings.Contains(segment, ">") {
			return candidate
		}
	}
	return start
}

func articleInnerHTML(value string) string {
	lower := strings.ToLower(value)
	start := strings.Index(lower, "<article")
	if start < 0 {
		return strings.TrimSpace(value)
	}
	openEnd := strings.Index(value[start:], ">")
	if openEnd < 0 {
		return strings.TrimSpace(value)
	}
	innerStart := start + openEnd + 1
	end := strings.LastIndex(lower, "</article>")
	if end < innerStart {
		end = len(value)
	}
	return strings.TrimSpace(value[innerStart:end])
}

func cleanHTMLText(value string) string {
	value = htmlTagRegex.ReplaceAllString(value, " ")
	return bookparser.CleanChapterTitle(value)
}

func isMobiNavigationTitle(value string) bool {
	normalized := strings.ToLower(cleanText(value))
	switch normalized {
	case "table of contents", "contents", "toc", "landmarks", "navigation", "muc luc", "mục lục":
		return true
	default:
		return false
	}
}

func htmlFragmentsToReaderHTML(value string) string {
	value = xmlDeclRegex.ReplaceAllString(value, "")
	value = htmlBodyTagRegex.ReplaceAllString(value, "")
	return value
}

func stripBrokenMobiTextNodes(value string) string {
	matches := htmlTagRegex.FindAllStringIndex(value, -1)
	if len(matches) == 0 {
		return stripBrokenMobiTextFragment(value)
	}

	var out strings.Builder
	out.Grow(len(value))
	last := 0
	for _, match := range matches {
		if match[0] > last {
			out.WriteString(stripBrokenMobiTextFragment(value[last:match[0]]))
		}
		out.WriteString(value[match[0]:match[1]])
		last = match[1]
	}
	if last < len(value) {
		out.WriteString(stripBrokenMobiTextFragment(value[last:]))
	}
	return out.String()
}

func stripBrokenMobiTextFragment(value string) string {
	value = escapedEmptyAnchorRegex.ReplaceAllString(value, "")
	return brokenAttrTextRegex.ReplaceAllStringFunc(value, func(fragment string) string {
		prefixLength := 0
		for prefixLength < len(fragment) {
			switch fragment[prefixLength] {
			case ' ', '\n', '\r', '\t', '>':
				prefixLength++
			default:
				return fragment[:prefixLength]
			}
		}
		return ""
	})
}

func normalizeMobiReaderMarkup(value string) string {
	value = emptyParagraphRegex.ReplaceAllString(value, "")
	return strings.TrimSpace(value)
}

func normalizeMobiSectionHTML(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	return normalizeMobiReaderMarkup(value)
}

func unsupportedHTML(filePath string, err error) string {
	title := html.EscapeString(bookparser.TitleFromPath(filePath))
	message := html.EscapeString(err.Error())
	return `<article><h2>` + title + `</h2><p>NovelHub opened this MOBI/AZW file, but this variant uses a MOBI feature that the native reader cannot decode yet.</p><p><strong>Reason:</strong> ` + message + `.</p><p>Attach an EPUB, PDF, TXT, or DOCX copy for full in-browser reading while this parser is expanded.</p></article>`
}

func readableHTMLTextLength(value string) int {
	text := htmlTagRegex.ReplaceAllString(value, " ")
	text = strings.Join(strings.Fields(text), " ")
	return len([]rune(text))
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
