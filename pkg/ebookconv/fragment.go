package ebookconv

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html/atom"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

func fragmentNodes(content string) []*nethtml.Node {
	nodes, err := nethtml.ParseFragment(strings.NewReader(content), &nethtml.Node{Type: nethtml.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return nil
	}
	return nodes
}

func rebaseChapterLinks(content, basePath string, chapters []bookparser.ChapterData, keyFor func(bookparser.ChapterData) string) string {
	nodes := fragmentNodes(content)
	if len(nodes) == 0 {
		return content
	}
	keys := make(map[string]string, len(chapters))
	for _, ch := range chapters {
		if p := strings.ToLower(filepath.ToSlash(ch.ContentPath)); p != "" {
			keys[p] = keyFor(ch)
		}
	}
	base := filepath.ToSlash(filepath.Dir(basePath))
	if base == "." {
		base = ""
	}
	var rewrite func(n *nethtml.Node)
	rewrite = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "a" {
			for i, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				value := strings.TrimSpace(attr.Val)
				lower := strings.ToLower(value)
				if strings.HasPrefix(value, "#") ||
					strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") ||
					strings.HasPrefix(value, "//") ||
					strings.HasPrefix(lower, "kindle:") || strings.HasPrefix(lower, "mobi-section:") || strings.HasPrefix(lower, "section:") {
					continue
				}
				resolved := strings.ToLower(filepath.ToSlash(filepath.Join(base, value)))
				if key, ok := keys[resolved]; ok {
					n.Attr[i].Val = key
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
	var sb bytes.Buffer
	for _, n := range nodes {
		if err := nethtml.Render(&sb, n); err != nil {
			return content
		}
	}
	return sb.String()
}

func fragmentText(content string) string {
	var sb strings.Builder
	for _, n := range fragmentNodes(content) {
		writeText(n, &sb)
	}
	return strings.TrimSpace(sb.String())
}

func writeText(n *nethtml.Node, sb *strings.Builder) {
	switch n.Type {
	case nethtml.TextNode:
		sb.WriteString(n.Data)
	case nethtml.ElementNode:
		switch strings.ToLower(n.Data) {
		case "br":
			sb.WriteString("\n")
		case "p", "div", "article", "section", "figure", "figcaption", "h1", "h2", "h3", "h4", "h5", "h6",
			"li", "blockquote", "table", "tr", "td", "th", "pre", "ul", "ol":
			sb.WriteString("\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				writeText(c, sb)
			}
			sb.WriteString("\n")
			return
		case "script", "style", "head", "title":
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		writeText(c, sb)
	}
}

func splitParagraphs(content string) []string {
	var out []string
	var add func(n *nethtml.Node, inline bool)
	add = func(n *nethtml.Node, inline bool) {
		if n.Type == nethtml.TextNode {
			if strings.TrimSpace(n.Data) != "" && inline {
				out = append(out, n.Data)
			}
			return
		}
		if n.Type != nethtml.ElementNode {
			return
		}
		switch strings.ToLower(n.Data) {
		case "p", "h1", "h2", "h3", "h4", "h5", "h6", "figure", "blockquote", "li", "pre":
			inner := renderChildren(n)
			out = append(out, n.Data+"|"+strings.TrimSpace(inner))
			return
		case "br":
			return
		case "script", "style", "head", "title":
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			add(c, inline)
		}
	}
	for _, n := range fragmentNodes(content) {
		add(n, false)
	}
	return out
}

func renderChildren(n *nethtml.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNode(c, &sb)
	}
	return sb.String()
}

func renderNode(n *nethtml.Node, sb *strings.Builder) {
	switch n.Type {
	case nethtml.TextNode:
		sb.WriteString(n.Data)
	case nethtml.ElementNode:
		tag := strings.ToLower(n.Data)
		switch tag {
		case "strong", "b":
			sb.WriteString("<b>")
			sb.WriteString(renderChildren(n))
			sb.WriteString("</b>")
		case "em", "i":
			sb.WriteString("<i>")
			sb.WriteString(renderChildren(n))
			sb.WriteString("</i>")
		case "br":
			sb.WriteString(" ")
		case "img":
			for _, a := range n.Attr {
				if a.Key == "src" {
					sb.WriteString(` <img src="`)
					sb.WriteString(escapeXML(a.Val))
					sb.WriteString(`"/>`)
				}
			}
		case "script", "style", "head", "title":
			return
		default:
			sb.WriteString(renderChildren(n))
		}
	}
}

func escapeXML(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func imageExt(src string) string {
	src = strings.SplitN(src, "?", 2)[0]
	src = strings.SplitN(src, "#", 2)[0]
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".jpeg" || ext == ".jpg" {
		return ".jpg"
	}
	if ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".svg" || ext == ".bmp" {
		return ext
	}
	return ".jpg"
}

func imageFilename(src string, idx int) string {
	return "img_" + strconv.Itoa(idx) + imageExt(src)
}

func mediaType(src string) string {
	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}
