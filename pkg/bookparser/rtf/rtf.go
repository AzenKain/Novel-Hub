package rtf

import (
	"encoding/hex"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
)

type Parser struct{}

type rtfState struct {
	skip      bool
	ucSkip    int
	bold      bool
	italic    bool
	underline bool
	strike    bool
	align     string
}

type rtfAsset struct {
	Name string
	Data []byte
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read rtf metadata: %w", err)
	}
	text := extractRTFText(string(data))
	meta := &bookparser.BookMetadata{
		Title:       bookparser.TitleFromPath(filePath),
		Description: preview(text),
	}
	images, err := p.ListImages(filePath)
	if err == nil && len(images) > 0 {
		coverData, err := p.GetAsset(filePath, images[0])
		if err == nil && len(coverData) > 0 {
			meta.CoverData = coverData
			meta.CoverType = "image/jpeg"
			if strings.HasSuffix(strings.ToLower(images[0]), ".png") {
				meta.CoverType = "image/png"
			}
		}
	}
	merged := bookparser.MergeMetadataSidecar(filePath, meta)
	if len(merged.CoverData) == 0 {
		merged.CoverData = defaultcover.GenerateSVG(merged.Title, merged.Author)
		merged.IsDefaultCover = true
		merged.CoverType = "image/svg+xml"
	}
	return merged, nil
}

func (p *Parser) ParseSpine(filePath string) ([]bookparser.ChapterData, error) {
	return []bookparser.ChapterData{{
		Title:       bookparser.TitleFromPath(filePath),
		ContentPath: "document",
		Index:       0,
	}}, nil
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
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read rtf content: %w", err)
	}
	return extractRTFHTML(string(data)), nil
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assets, err := readRTFAssets(filePath)
	if err != nil {
		return nil, err
	}
	assetPath = strings.TrimLeft(assetPath, "/")
	for _, asset := range assets {
		if asset.Name == assetPath {
			return asset.Data, nil
		}
	}
	return nil, fmt.Errorf("rtf asset %q not found", assetPath)
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	assets, err := readRTFAssets(filePath)
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

func readRTFAssets(filePath string) ([]rtfAsset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read rtf assets: %w", err)
	}
	return extractRTFAssets(string(data)), nil
}

func extractRTFAssets(input string) []rtfAsset {
	var assets []rtfAsset
	for offset := 0; offset < len(input); {
		start := strings.Index(input[offset:], `{\pict`)
		if start < 0 {
			break
		}
		start += offset
		end := balancedGroupEnd(input, start)
		if end <= start {
			offset = start + len(`{\pict`)
			continue
		}
		group := input[start:end]
		data := decodeRTFPictureHex(group)
		if len(data) > 0 {
			ext := rtfPictureExt(group)
			assets = append(assets, rtfAsset{
				Name: fmt.Sprintf("images/pict-%03d.%s", len(assets)+1, ext),
				Data: data,
			})
		}
		offset = end
	}
	return assets
}

func balancedGroupEnd(input string, start int) int {
	depth := 0
	escaped := false
	for i := start; i < len(input); i++ {
		c := input[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(input)
}

func decodeRTFPictureHex(group string) []byte {
	longest := ""
	var current strings.Builder
	flush := func() {
		value := current.String()
		current.Reset()
		if len(value) > len(longest) {
			longest = value
		}
	}
	for _, r := range group {
		switch {
		case (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F'):
			current.WriteRune(r)
		case unicode.IsSpace(r):
			continue
		default:
			flush()
		}
	}
	flush()
	if len(longest) < 16 {
		return nil
	}
	if len(longest)%2 == 1 {
		longest = longest[:len(longest)-1]
	}
	data, err := hex.DecodeString(longest)
	if err != nil {
		return nil
	}
	return data
}

func rtfPictureExt(group string) string {
	lower := strings.ToLower(group)
	switch {
	case strings.Contains(lower, `\pngblip`):
		return "png"
	case strings.Contains(lower, `\jpegblip`):
		return "jpg"
	case strings.Contains(lower, `\emfblip`):
		return "emf"
	case strings.Contains(lower, `\wmetafile`):
		return "wmf"
	case strings.Contains(lower, `\dibitmap`):
		return "bmp"
	default:
		return "bin"
	}
}

func extractRTFHTML(input string) string {
	state := rtfState{ucSkip: 1}
	stack := []rtfState{state}
	var out strings.Builder
	var pBuf strings.Builder
	var activeBold, activeItalic, activeUnderline, activeStrike bool
	var curAlign string
	skipChars := 0

	out.WriteString("<article>")
	flushPara := func() {
		if activeStrike {
			pBuf.WriteString("</s>")
			activeStrike = false
		}
		if activeUnderline {
			pBuf.WriteString("</u>")
			activeUnderline = false
		}
		if activeItalic {
			pBuf.WriteString("</i>")
			activeItalic = false
		}
		if activeBold {
			pBuf.WriteString("</b>")
			activeBold = false
		}

		val := strings.TrimSpace(pBuf.String())
		if val != "" {
			out.WriteString("<p")
			if curAlign != "" {
				out.WriteString(fmt.Sprintf(` align="%s"`, curAlign))
			}
			out.WriteString(">")
			out.WriteString(val)
			out.WriteString("</p>\n")
		}
		pBuf.Reset()
		curAlign = ""
	}

	syncStyles := func() {
		if state.bold != activeBold {
			if state.bold {
				pBuf.WriteString("<b>")
			} else {
				pBuf.WriteString("</b>")
			}
			activeBold = state.bold
		}
		if state.italic != activeItalic {
			if state.italic {
				pBuf.WriteString("<i>")
			} else {
				pBuf.WriteString("</i>")
			}
			activeItalic = state.italic
		}
		if state.underline != activeUnderline {
			if state.underline {
				pBuf.WriteString("<u>")
			} else {
				pBuf.WriteString("</u>")
			}
			activeUnderline = state.underline
		}
		if state.strike != activeStrike {
			if state.strike {
				pBuf.WriteString("<s>")
			} else {
				pBuf.WriteString("</s>")
			}
			activeStrike = state.strike
		}
		if state.align != "" && curAlign == "" {
			curAlign = state.align
		}
	}

	for i := 0; i < len(input); {
		c := input[i]
		switch c {
		case '{':
			stack = append(stack, state)
			i++
		case '}':
			if len(stack) > 1 {
				state = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				syncStyles()
			}
			i++
		case '\\':
			var isPar bool
			ni, skip, isP := parseControlHTML(input, i+1, &state, &pBuf, syncStyles)
			i = ni
			skipChars = skip
			isPar = isP
			if isPar {
				flushPara()
			}
		case '\r', '\n':
			i++
		default:
			r, size := utf8.DecodeRuneInString(input[i:])
			if skipChars > 0 {
				skipChars--
			} else if !state.skip {
				syncStyles()
				pBuf.WriteString(html.EscapeString(string(r)))
			}
			i += size
		}
	}
	flushPara()
	out.WriteString("</article>")
	return out.String()
}

func parseControlHTML(input string, i int, state *rtfState, out *strings.Builder, syncStyles func()) (int, int, bool) {
	if i >= len(input) {
		return i, 0, false
	}
	c := input[i]
	if c == '\\' || c == '{' || c == '}' {
		if !state.skip {
			syncStyles()
			out.WriteString(html.EscapeString(string(c)))
		}
		return i + 1, 0, false
	}
	if c == '~' {
		if !state.skip {
			syncStyles()
			out.WriteString("&nbsp;")
		}
		return i + 1, 0, false
	}
	if c == '_' {
		if !state.skip {
			syncStyles()
			out.WriteByte('-')
		}
		return i + 1, 0, false
	}
	if c == '\'' && i+2 < len(input) {
		if !state.skip {
			if value, err := strconv.ParseUint(input[i+1:i+3], 16, 8); err == nil {
				syncStyles()
				out.WriteString(html.EscapeString(string(charmap.Windows1252.DecodeByte(byte(value)))))
			}
		}
		return i + 3, 0, false
	}
	if c == '*' {
		state.skip = true
		return i + 1, 0, false
	}
	if !isASCIILetter(c) {
		return i + 1, 0, false
	}

	start := i
	for i < len(input) && isASCIILetter(input[i]) {
		i++
	}
	word := input[start:i]
	sign := 1
	if i < len(input) && input[i] == '-' {
		sign = -1
		i++
	}
	numStart := i
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	hasParam := i > numStart
	param := 0
	if hasParam {
		param, _ = strconv.Atoi(input[numStart:i])
		param *= sign
	}
	if i < len(input) && input[i] == ' ' {
		i++
	}

	if isDestination(word) {
		state.skip = true
		return i, 0, false
	}
	if state.skip {
		return i, 0, false
	}

	switch word {
	case "par", "line":
		return i, 0, true
	case "tab":
		syncStyles()
		out.WriteString("&emsp;")
	case "emdash":
		syncStyles()
		out.WriteString("&mdash;")
	case "endash":
		syncStyles()
		out.WriteString("&ndash;")
	case "bullet":
		syncStyles()
		out.WriteString("&bull; ")
	case "b":
		state.bold = (!hasParam || param != 0)
	case "i":
		state.italic = (!hasParam || param != 0)
	case "ul":
		state.underline = (!hasParam || param != 0)
	case "ulnone":
		state.underline = false
	case "strike":
		state.strike = (!hasParam || param != 0)
	case "qc":
		state.align = "center"
	case "qr":
		state.align = "right"
	case "qj":
		state.align = "justify"
	case "ql":
		state.align = "left"
	case "pard":
		state.align = ""
	case "uc":
		if hasParam && param >= 0 {
			state.ucSkip = param
		}
	case "u":
		if hasParam {
			if param < 0 {
				param += 65536
			}
			syncStyles()
			out.WriteString(html.EscapeString(string(rune(param))))
			return i, state.ucSkip, false
		}
	}
	return i, 0, false
}

func extractRTFText(input string) string {
	state := rtfState{ucSkip: 1}
	stack := []rtfState{state}
	var out strings.Builder
	skipChars := 0

	for i := 0; i < len(input); {
		c := input[i]
		switch c {
		case '{':
			stack = append(stack, state)
			i++
		case '}':
			if len(stack) > 1 {
				state = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
			i++
		case '\\':
			ni, skip := parseControl(input, i+1, &state, &out)
			i = ni
			skipChars = skip
		case '\r', '\n':
			i++
		default:
			r, size := utf8.DecodeRuneInString(input[i:])
			if skipChars > 0 {
				skipChars--
			} else if !state.skip {
				out.WriteRune(r)
			}
			i += size
		}
	}
	return cleanText(out.String())
}

func parseControl(input string, i int, state *rtfState, out *strings.Builder) (int, int) {
	if i >= len(input) {
		return i, 0
	}
	c := input[i]
	if c == '\\' || c == '{' || c == '}' {
		if !state.skip {
			out.WriteByte(c)
		}
		return i + 1, 0
	}
	if c == '~' {
		if !state.skip {
			out.WriteRune('\u00a0')
		}
		return i + 1, 0
	}
	if c == '_' {
		if !state.skip {
			out.WriteByte('-')
		}
		return i + 1, 0
	}
	if c == '\'' && i+2 < len(input) {
		if !state.skip {
			if value, err := strconv.ParseUint(input[i+1:i+3], 16, 8); err == nil {
				out.WriteRune(charmap.Windows1252.DecodeByte(byte(value)))
			}
		}
		return i + 3, 0
	}
	if c == '*' {
		state.skip = true
		return i + 1, 0
	}
	if !isASCIILetter(c) {
		return i + 1, 0
	}

	start := i
	for i < len(input) && isASCIILetter(input[i]) {
		i++
	}
	word := input[start:i]
	sign := 1
	if i < len(input) && input[i] == '-' {
		sign = -1
		i++
	}
	numStart := i
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	hasParam := i > numStart
	param := 0
	if hasParam {
		param, _ = strconv.Atoi(input[numStart:i])
		param *= sign
	}
	if i < len(input) && input[i] == ' ' {
		i++
	}

	if isDestination(word) {
		state.skip = true
		return i, 0
	}
	if state.skip {
		return i, 0
	}
	switch word {
	case "par", "line":
		out.WriteString("\n\n")
	case "tab":
		out.WriteByte('\t')
	case "emdash":
		out.WriteString("--")
	case "endash":
		out.WriteByte('-')
	case "bullet":
		out.WriteString("* ")
	case "b":
		state.bold = (!hasParam || param != 0)
	case "i":
		state.italic = (!hasParam || param != 0)
	case "ul":
		state.underline = (!hasParam || param != 0)
	case "ulnone":
		state.underline = false
	case "strike":
		state.strike = (!hasParam || param != 0)
	case "qc":
		state.align = "center"
	case "qr":
		state.align = "right"
	case "qj":
		state.align = "justify"
	case "ql":
		state.align = "left"
	case "uc":
		if hasParam && param >= 0 {
			state.ucSkip = param
		}
	case "u":
		if hasParam {
			if param < 0 {
				param += 65536
			}
			out.WriteRune(rune(param))
			return i, state.ucSkip
		}
	}
	return i, 0
}

func isDestination(word string) bool {
	switch word {
	case "fonttbl", "colortbl", "stylesheet", "pict", "object", "datastore", "themedata", "generator", "fontemb", "filetbl", "revtbl", "xmlopen", "xmlattrname", "xmlattrvalue", "listtable", "listoverridetable", "info":
		return true
	default:
		return false
	}
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func cleanText(value string) string {
	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimFunc(line, func(r rune) bool {
			return unicode.IsSpace(r) && r != '\t'
		})
		if line == "" {
			if !blank {
				cleaned = append(cleaned, "")
			}
			blank = true
			continue
		}
		cleaned = append(cleaned, strings.Join(strings.Fields(line), " "))
		blank = false
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func preview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return strings.TrimSpace(text[:500]) + "..."
	}
	return text
}
