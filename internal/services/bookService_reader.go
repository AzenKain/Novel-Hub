package services

import (
	"bytes"
	"html"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"novelhub/pkg/bookparser"

	nethtml "golang.org/x/net/html"
)

func fileChapterIndex(fileID string, chapterID string) (int, bool) {
	prefix := fileID + ":"
	if !strings.HasPrefix(chapterID, prefix) {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimPrefix(chapterID, prefix))
	return index, err == nil
}

func rawFileReaderHTML(bookID string, fileID string, filePath string) string {
	sourceURL := `/api/v1/reader/` + url.PathEscape(bookID) + `/file?file_id=` + url.QueryEscape(fileID)
	title := html.EscapeString(bookparser.TitleFromPath(filePath))
	return `<div class="novelhub-raw-reader" style="width: 100%; height: 100%; margin: 0; padding: 0; overflow: hidden;"><iframe title="` + title + `" src="` + sourceURL + `" style="width: 100%; height: 100%; border: 0; background: #fff;" loading="eager"></iframe></div>`
}

func rewriteReaderHTML(content string, bookID string, contentPath string, fileID string) string {
	baseDir := filepath.ToSlash(filepath.Dir(contentPath))
	if baseDir == "." {
		baseDir = ""
	}

	doc, err := nethtml.Parse(strings.NewReader(content))
	if err != nil {
		return content
	}

	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			for i, attr := range n.Attr {
				if attr.Key == "src" || attr.Key == "href" || attr.Key == "xlink:href" || (attr.Namespace == "xlink" && attr.Key == "href") {
					value := strings.TrimSpace(attr.Val)
					lower := strings.ToLower(value)
					if strings.HasPrefix(value, "#") {
						continue
					}
					if strings.HasPrefix(lower, "http:") || strings.HasPrefix(lower, "https:") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "vbscript:") || strings.HasPrefix(value, "//") {
						n.Attr[i].Val = "#"
						continue
					}

					resolved := filepath.ToSlash(filepath.Join(baseDir, value))
					assetURL := `/api/v1/reader/` + url.PathEscape(bookID) + `/asset/` + escapeAssetPath(resolved)
					if fileID != "" {
						assetURL += `?file_id=` + url.QueryEscape(fileID)
					}
					n.Attr[i].Val = assetURL
				}
			}

			if n.Data == "style" {
				if n.FirstChild != nil && n.FirstChild.Type == nethtml.TextNode {
					n.FirstChild.Data = scopeReaderCSS(n.FirstChild.Data)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	var buf bytes.Buffer
	if err := nethtml.Render(&buf, doc); err != nil {
		return content
	}

	return buf.String()
}

func escapeAssetPath(assetPath string) string {
	parts := strings.Split(filepath.ToSlash(assetPath), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func scopeReaderCSS(css string) string {
	var out strings.Builder
	for i := 0; i < len(css); {
		openRel := strings.IndexByte(css[i:], '{')
		if openRel < 0 {
			out.WriteString(css[i:])
			break
		}
		open := i + openRel
		close := matchingCSSBrace(css, open)
		if close < 0 {
			out.WriteString(css[i:])
			break
		}

		selector := css[i:open]
		selectorTrimmed := strings.TrimSpace(selector)
		out.WriteString(scopeReaderSelectorList(selector))
		out.WriteByte('{')
		block := css[open+1 : close]
		if readerCSSAtRuleScopesChildren(selectorTrimmed) {
			out.WriteString(scopeReaderCSS(block))
		} else {
			out.WriteString(block)
		}
		out.WriteByte('}')
		i = close + 1
	}
	return out.String()
}

func matchingCSSBrace(css string, open int) int {
	depth := 0
	for i := open; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func scopeReaderSelectorList(selector string) string {
	trimmed := strings.TrimSpace(selector)
	if strings.HasPrefix(trimmed, "@") {
		return selector
	}
	parts := strings.Split(selector, ",")
	for i, part := range parts {
		parts[i] = scopeReaderSelector(part)
	}
	return strings.Join(parts, ",")
}

func scopeReaderSelector(selector string) string {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return selector
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, ".reader-content") {
		return trimmed
	}
	for _, prefix := range []string{"body", "html"} {
		if lower == prefix {
			return ".reader-content"
		}
		if strings.HasPrefix(lower, prefix+" ") {
			return ".reader-content " + strings.TrimSpace(trimmed[len(prefix):])
		}
		if strings.HasPrefix(lower, prefix+">") {
			return ".reader-content " + strings.TrimSpace(trimmed[len(prefix):])
		}
		if strings.HasPrefix(lower, prefix+".") || strings.HasPrefix(lower, prefix+"#") || strings.HasPrefix(lower, prefix+":") {
			restStart := len(prefix)
			for restStart < len(trimmed) && trimmed[restStart] != ' ' && trimmed[restStart] != '>' && trimmed[restStart] != '+' && trimmed[restStart] != '~' {
				restStart++
			}
			if restStart < len(trimmed) {
				return ".reader-content " + strings.TrimSpace(trimmed[restStart:])
			}
			return ".reader-content"
		}
	}
	return ".reader-content " + trimmed
}

func readerCSSAtRuleScopesChildren(selector string) bool {
	lower := strings.ToLower(strings.TrimSpace(selector))
	return strings.HasPrefix(lower, "@media") ||
		strings.HasPrefix(lower, "@supports") ||
		strings.HasPrefix(lower, "@container") ||
		strings.HasPrefix(lower, "@layer") ||
		strings.HasPrefix(lower, "@scope")
}

func normalizeCoverExt(ext string) string {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if ext == ".jpeg" {
		return ".jpg"
	}
	switch ext {
	case ".jpg", ".png", ".webp", ".gif", ".bmp", ".svg":
		return ext
	default:
		return ".jpg"
	}
}

func coverExtFromContent(contentType string, data []byte) string {
	contentType = strings.ToLower(contentType)
	if !isSupportedCoverContentType(contentType) {
		contentType = strings.ToLower(http.DetectContentType(data))
	}
	switch {
	case strings.Contains(contentType, "svg"):
		return ".svg"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "bmp"):
		return ".bmp"
	default:
		return ".jpg"
	}
}

func isSupportedCoverContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "jpeg") ||
		strings.Contains(contentType, "jpg") ||
		strings.Contains(contentType, "png") ||
		strings.Contains(contentType, "webp") ||
		strings.Contains(contentType, "gif") ||
		strings.Contains(contentType, "bmp") ||
		strings.Contains(contentType, "svg")
}

func readerAssetContentType(assetPath string) string {
	switch strings.ToLower(filepath.Ext(assetPath)) {
	case ".css":
		return "text/css"
	case ".html", ".xhtml":
		return "text/html"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".avif":
		return "image/avif"
	case ".svg":
		return "image/svg+xml"
	case ".js":
		return "application/javascript"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".m4b":
		return "audio/mp4"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	default:
		return "application/octet-stream"
	}
}
