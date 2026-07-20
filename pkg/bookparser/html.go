package bookparser

import (
	"bufio"
	"bytes"
	"html"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
)

func TitleFromPath(filePath string) string {
	name := filepath.Base(filePath)
	ext := filepath.Ext(name)
	if strings.HasSuffix(strings.ToLower(name), ".kepub.epub") {
		ext = ".kepub.epub"
	}
	title := strings.TrimSuffix(name, ext)
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.TrimSpace(title)
	if title == "" {
		return "Untitled"
	}
	return title
}

func PlainTextToHTML(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return `<article><p>No readable text was found in this file.</p></article>`
	}
	var out strings.Builder
	out.WriteString(`<article>`)
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 1024), 4*1024*1024)
	var paragraph bytes.Buffer
	flush := func() {
		value := strings.TrimSpace(paragraph.String())
		if value == "" {
			paragraph.Reset()
			return
		}
		out.WriteString("<p>")
		out.WriteString(html.EscapeString(value))
		out.WriteString("</p>")
		paragraph.Reset()
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			flush()
			continue
		}
		if paragraph.Len() > 0 {
			paragraph.WriteByte(' ')
		}
		paragraph.WriteString(line)
	}
	flush()
	out.WriteString(`</article>`)
	return out.String()
}

func PreformattedTextToHTML(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return `<article><p>No readable text was found in this file.</p></article>`
	}
	return `<article><pre style="white-space: pre-wrap; font-family: inherit; line-height: inherit;">` + html.EscapeString(text) + `</pre></article>`
}

func MarkdownToHTML(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return `<article><p>No readable text was found in this file.</p></article>`
	}
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(text), &buf); err != nil {
		return PlainTextToHTML(text)
	}
	return `<article>` + buf.String() + `</article>`
}

func FirstMarkdownHeading(text string) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}
		value := strings.TrimLeft(line, "#")
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
