package ebookconv

import (
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html/atom"

	nethtml "golang.org/x/net/html"
)

// fragmentNodes parses an XHTML chapter fragment (the form every bookparser
// GetChapterContent returns) into a node tree.
func fragmentNodes(content string) []*nethtml.Node {
	nodes, err := nethtml.ParseFragment(strings.NewReader(content), &nethtml.Node{Type: nethtml.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return nil
	}
	return nodes
}

// fragmentText flattens a fragment to plain text: block elements and <br>
// become line breaks.
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

// splitParagraphs returns one HTML-snippet string per block-level element in
// the fragment (p, headings, divs, figures). Used by the DOCX writer.
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

// renderChildren serializes a node's children back to HTML, preserving inline
// markup (strong/em/b/i) that the DOCX writer needs.
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
					sb.WriteString(" ")
					sb.WriteString(a.Val)
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
