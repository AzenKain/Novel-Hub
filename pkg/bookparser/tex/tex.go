package tex

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/bookparser/defaultcover"
	"novelhub/pkg/constants"
	"novelhub/pkg/localfs"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

var (
	titleRegex       = regexp.MustCompile(`\\title\{([^}]+)\}`)
	authorRegex      = regexp.MustCompile(`\\author\{([^}]+)\}`)
	dateRegex        = regexp.MustCompile(`\\date\{([^}]+)\}`)
	abstractRegex    = regexp.MustCompile(`(?s)\\begin\{abstract\}(.*?)\\end\{abstract\}`)
	commentRegex     = regexp.MustCompile(`(?m)(^|[^\\])%.*$`)
	chapterSecRegex  = regexp.MustCompile(`(?m)^\s*\\(chapter|section)\*?\{([^}]+)\}`)
)

type texSection struct {
	Title   string
	Content string
}

func (p *Parser) ParseMetadata(filePath string) (*bookparser.BookMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read tex metadata: %w", err)
	}
	text := string(data)

	title := ""
	if m := titleRegex.FindStringSubmatch(text); len(m) > 1 {
		title = cleanLatexInline(m[1])
	}
	if title == "" {
		title = bookparser.TitleFromPath(filePath)
	}

	author := ""
	if m := authorRegex.FindStringSubmatch(text); len(m) > 1 {
		author = cleanLatexInline(m[1])
	}

	date := ""
	if m := dateRegex.FindStringSubmatch(text); len(m) > 1 {
		date = cleanLatexInline(m[1])
	}

	desc := ""
	if m := abstractRegex.FindStringSubmatch(text); len(m) > 1 {
		desc = cleanLatexInline(m[1])
	}

	meta := &bookparser.BookMetadata{
		Title:       title,
		Author:      author,
		Date:        date,
		Description: desc,
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
	sections, title, err := p.extractSections(filePath)
	if err != nil {
		return nil, err
	}
	if len(sections) <= 1 {
		return []bookparser.ChapterData{{
			Title:       title,
			ContentPath: "document",
			Index:       0,
		}}, nil
	}

	chapters := make([]bookparser.ChapterData, 0, len(sections))
	for i, sec := range sections {
		chapters = append(chapters, bookparser.ChapterData{
			Title:       sec.Title,
			ContentPath: "section:" + strconv.Itoa(i),
			Index:       i,
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
	sections, _, err := p.extractSections(filePath)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(contentPath, "section:") {
		idxStr := strings.TrimPrefix(contentPath, "section:")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(sections) {
			return "", fmt.Errorf("invalid tex section index %q", contentPath)
		}
		return "<article>" + latexBodyToHTML(sections[idx].Content) + "</article>", nil
	}

	if len(sections) > 0 {
		var fullContent strings.Builder
		for _, sec := range sections {
			fullContent.WriteString(sec.Content)
			fullContent.WriteString("\n\n")
		}
		return "<article>" + latexBodyToHTML(fullContent.String()) + "</article>", nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return "<article>" + latexBodyToHTML(string(data)) + "</article>", nil
}

func (p *Parser) extractSections(filePath string) ([]texSection, string, error) {
	meta, _ := p.ParseMetadata(filePath)
	docTitle := bookparser.TitleFromPath(filePath)
	if meta != nil && strings.TrimSpace(meta.Title) != "" {
		docTitle = meta.Title
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("read tex: %w", err)
	}
	raw := string(data)

	// Strip comments
	raw = commentRegex.ReplaceAllString(raw, "$1")

	// Extract body inside \begin{document}...\end{document} if present
	body := raw
	if start := strings.Index(body, `\begin{document}`); start >= 0 {
		body = body[start+len(`\begin{document}`):]
	}
	if end := strings.LastIndex(body, `\end{document}`); end >= 0 {
		body = body[:end]
	}
	body = strings.TrimSpace(body)

	matches := chapterSecRegex.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return []texSection{{Title: docTitle, Content: body}}, docTitle, nil
	}

	var sections []texSection
	// Pre-section intro (if any)
	if matches[0][0] > 0 {
		pre := strings.TrimSpace(body[:matches[0][0]])
		if len(pre) > 30 {
			sections = append(sections, texSection{
				Title:   docTitle,
				Content: pre,
			})
		}
	}

	for i, m := range matches {
		secTitle := cleanLatexInline(body[m[4]:m[5]])
		if secTitle == "" {
			secTitle = fmt.Sprintf("Section %d", len(sections)+1)
		}
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		content := strings.TrimSpace(body[m[0]:end])
		sections = append(sections, texSection{
			Title:   secTitle,
			Content: content,
		})
	}

	return sections, docTitle, nil
}

func cleanLatexInline(s string) string {
	cmdRe := regexp.MustCompile(`\\([a-zA-Z]+)\*?(?:\[[^\]]*\])?\{([^{}]*)\}`)
	for {
		prev := s
		s = cmdRe.ReplaceAllString(s, "$2")
		if s == prev {
			break
		}
	}
	s = strings.ReplaceAll(s, `\\`, " ")
	s = strings.ReplaceAll(s, `\ `, " ")
	s = strings.ReplaceAll(s, `$`, "")
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func latexBodyToHTML(s string) string {
	// Headings — resolve nested inline commands in heading text
	cmdRe := regexp.MustCompile(`\\([a-zA-Z]+)\*?(?:\[[^\]]*\])?\{([^{}]*)\}`)
	resolveNested := func(text string) string {
		for {
			prev := text
			text = cmdRe.ReplaceAllStringFunc(text, func(m string) string {
				sub := cmdRe.FindStringSubmatch(m)
				if len(sub) < 3 {
					return m
				}
				return sub[2]
			})
			if text == prev {
				break
			}
		}
		return text
	}
	headings := []struct {
		pattern string
		tag    string
	}{
		{`(?m)^\s*\\chapter\*?\{([^}]+)\}`, "h1"},
		{`(?m)^\s*\\section\*?\{([^}]+)\}`, "h2"},
		{`(?m)^\s*\\subsection\*?\{([^}]+)\}`, "h3"},
		{`(?m)^\s*\\subsubsection\*?\{([^}]+)\}`, "h4"},
		{`(?m)^\s*\\paragraph\*?\{([^}]+)\}`, "h5"},
	}
	for _, h := range headings {
		s = regexp.MustCompile(h.pattern).ReplaceAllStringFunc(s, func(m string) string {
			sub := regexp.MustCompile(`\{([^}]+)\}`).FindStringSubmatch(m)
			if len(sub) < 2 {
				return m
			}
			return "\n<" + h.tag + ">" + resolveNested(sub[1]) + "</" + h.tag + ">\n"
		})
	}

	// Alignment environments
	s = regexp.MustCompile(`(?s)\\begin\{center\}(.*?)\\end\{center\}`).ReplaceAllString(s, `<div align="center">$1</div>`)
	s = regexp.MustCompile(`(?s)\\begin\{flushright\}(.*?)\\end\{flushright\}`).ReplaceAllString(s, `<div align="right">$1</div>`)
	s = regexp.MustCompile(`(?s)\\begin\{flushleft\}(.*?)\\end\{flushleft\}`).ReplaceAllString(s, `<div align="left">$1</div>`)
	s = regexp.MustCompile(`(?s)\\begin\{quote\}(.*?)\\end\{quote\}`).ReplaceAllString(s, `<blockquote>$1</blockquote>`)
	s = regexp.MustCompile(`(?s)\\begin\{quotation\}(.*?)\\end\{quotation\}`).ReplaceAllString(s, `<blockquote>$1</blockquote>`)

	// Lists
	s = regexp.MustCompile(`(?s)\\begin\{itemize\}(.*?)\\end\{itemize\}`).ReplaceAllString(s, `<ul>$1</ul>`)
	s = regexp.MustCompile(`(?s)\\begin\{enumerate\}(.*?)\\end\{enumerate\}`).ReplaceAllString(s, `<ol>$1</ol>`)
	s = regexp.MustCompile(`(?m)^\s*\\item\s*`).ReplaceAllString(s, "<li>")

	// Figures and Images (before inline formatting so \includegraphics/\caption survive)
	s = regexp.MustCompile(`(?s)\\begin\{figure\*?\}(.*?)\\end\{figure\*?\}`).ReplaceAllStringFunc(s, func(figBlock string) string {
		imgMatch := regexp.MustCompile(`\\includegraphics\*?(?:\[[^\]]*\])?\{([^}]+)\}`).FindStringSubmatch(figBlock)
		capMatch := regexp.MustCompile(`\\caption\*?\{([^}]+)\}`).FindStringSubmatch(figBlock)
		if len(imgMatch) > 1 {
			imgPath := cleanLatexInline(imgMatch[1])
			caption := ""
			if len(capMatch) > 1 {
				caption = cleanLatexInline(capMatch[1])
			}
			if caption != "" {
				return fmt.Sprintf("\n<figure style=\"text-align: center; margin: 1.5em 0;\"><img src=\"%s\" style=\"max-width: 100%%; height: auto; border-radius: 6px;\" /><figcaption style=\"margin-top: 0.5em; font-size: 0.9em; opacity: 0.8;\">%s</figcaption></figure>\n", html.EscapeString(imgPath), html.EscapeString(caption))
			}
			return fmt.Sprintf("\n<div class=\"tex-image\" style=\"text-align: center; margin: 1.5em 0;\"><img src=\"%s\" style=\"max-width: 100%%; height: auto; border-radius: 6px;\" /></div>\n", html.EscapeString(imgPath))
		}
		return figBlock
	})

	// Standalone \includegraphics
	s = regexp.MustCompile(`\\includegraphics\*?(?:\[[^\]]*\])?\{([^}]+)\}`).ReplaceAllStringFunc(s, func(match string) string {
		sub := regexp.MustCompile(`\{([^}]+)\}`).FindStringSubmatch(match)
		if len(sub) > 1 {
			imgPath := cleanLatexInline(sub[1])
			return fmt.Sprintf("<div class=\"tex-image\" style=\"text-align: center; margin: 1.5em 0;\"><img src=\"%s\" style=\"max-width: 100%%; height: auto; border-radius: 6px;\" /></div>", html.EscapeString(imgPath))
		}
		return ""
	})

	// Inline formatting (innermost-first loop to handle nested braces)
	s = latexCmdToHTML(s, map[string]string{
		`textbf`: "b", `textit`: "i", `emph`: "i",
		`underline`: "u", `sout`: "s",
		`textsc`: `span class="small-caps"`, `uppercase`: `span class="uppercase"`,
		`texttt`: "code",
	})

	// Math inline: $...$ -> <code>...</code>
	s = regexp.MustCompile(`\$([^$]+)\$`).ReplaceAllString(s, "<code>$1</code>")

	// Line breaks
	s = strings.ReplaceAll(s, `\\`, "<br/>")

	// Remove unhandled commands e.g. \maketitle, \tableofcontents, \vspace{...}
	s = regexp.MustCompile(`\\[a-zA-Z]+\*?(?:\[[^\]]*\])?(?:\{[^{}]*\})?`).ReplaceAllString(s, "")

	// Wrap standalone paragraphs into <p>...</p>
	lines := strings.Split(s, "\n\n")
	var out strings.Builder
	for _, block := range lines {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if strings.HasPrefix(block, "<h") || strings.HasPrefix(block, "<div") ||
			strings.HasPrefix(block, "<ul") || strings.HasPrefix(block, "<ol") ||
			strings.HasPrefix(block, "<blockquote") || strings.HasPrefix(block, "<figure") {
			out.WriteString(block)
		} else {
			out.WriteString("<p>")
			out.WriteString(block)
			out.WriteString("</p>")
		}
		out.WriteString("\n")
	}

	return out.String()
}

func (p *Parser) GetAsset(filePath, assetPath string) ([]byte, error) {
	assetPath = strings.TrimLeft(filepath.ToSlash(assetPath), "/")
	if assetPath == "" || !filepath.IsLocal(assetPath) {
		return nil, fmt.Errorf("invalid asset path")
	}

	dir := filepath.Dir(filePath)
	fullPath, err := localfs.SafeJoin(dir, assetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve tex asset: %w", err)
	}

	// If file doesn't have an extension, try common image extensions
	if filepath.Ext(fullPath) == "" {
		for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".svg", ".pdf"} {
			if _, err := os.Stat(fullPath + ext); err == nil {
				fullPath = fullPath + ext
				break
			}
		}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read tex asset: %w", err)
	}
	if int64(len(data)) > constants.MaxArchiveAssetSize {
		return nil, fmt.Errorf("asset file exceeds maximum allowed size")
	}
	return data, nil
}

func (p *Parser) ListImages(filePath string) ([]string, error) {
	dir := filepath.Dir(filePath)
	var images []string

	scanDir := func(subDir string, relPrefix string) {
		entries, err := os.ReadDir(subDir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			lower := strings.ToLower(entry.Name())
			ext := filepath.Ext(lower)
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".svg" {
				if relPrefix != "" {
					images = append(images, filepath.ToSlash(filepath.Join(relPrefix, entry.Name())))
				} else {
					images = append(images, entry.Name())
				}
			}
		}
	}

	scanDir(dir, "")
	for _, sub := range []string{"figures", "images", "img", "assets", "pictures"} {
		scanDir(filepath.Join(dir, sub), sub)
	}

	sort.Strings(images)
	return images, nil
}

func (p *Parser) SaveOriginalMetadataAndFix(filePath string, meta *bookparser.BookMetadata) error {
	return bookparser.SaveMetadataSidecar(filePath, meta)
}

func latexCmdToHTML(s string, cmdTags map[string]string) string {
	re := regexp.MustCompile(`\\([a-zA-Z]+)\*?(?:\[[^\]]*\])?\{([^{}]*)\}`)
	for {
		prev := s
		s = re.ReplaceAllStringFunc(s, func(match string) string {
			sub := re.FindStringSubmatch(match)
			if len(sub) < 3 {
				return match
			}
			cmd, content := sub[1], sub[2]
			tag, ok := cmdTags[cmd]
			if !ok {
				return content
			}
			if strings.HasPrefix(tag, "span ") {
				return fmt.Sprintf("<%s>%s</span>", tag, content)
			}
			return fmt.Sprintf("<%s>%s</%s>", tag, content, tag)
		})
		if s == prev {
			break
		}
	}
	return s
}
