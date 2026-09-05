package ebookconv

import (
	"fmt"
	"regexp"
	"strings"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

var whitespaceSplit = regexp.MustCompile(`(\s+)`)

func writeKEPUB(book *bookparser.BookData, images []Image) ([]byte, error) {
	return writeEPUBWithKobo(book, images, true)
}

func kepubify(content string) string {
	nodes := fragmentNodes(content)
	if len(nodes) == 0 {
		return content
	}
	uuid := strings.ReplaceAll(newID(), "-", "")
	span := 1
	var rewrite func(n *nethtml.Node)
	rewrite = func(n *nethtml.Node) {
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if c.Type == nethtml.TextNode {
				replacement := koboSpans(c.Data, uuid, &span)
				if len(replacement) > 0 {
					for _, sn := range replacement {
						n.InsertBefore(sn, c)
					}
					n.RemoveChild(c)
				}
			} else {
				rewrite(c)
			}
			c = next
		}
	}
	for _, n := range nodes {
		rewrite(n)
	}
	var b strings.Builder
	for _, n := range nodes {
		_ = nethtml.Render(&b, n)
	}
	return b.String()
}

func koboSpans(text, uuid string, counter *int) []*nethtml.Node {
	runs := whitespaceSplit.FindAllStringIndex(text, -1)
	if len(runs) == 0 {
		return nil
	}
	out := make([]*nethtml.Node, 0, len(runs)*2+1)
	last := 0
	for _, loc := range runs {
		if loc[0] > last {
			*counter++
			out = append(out, koboWord(text[last:loc[0]], uuid, *counter))
		}
		out = append(out, &nethtml.Node{Type: nethtml.TextNode, Data: text[loc[0]:loc[1]]})
		last = loc[1]
	}
	if last < len(text) {
		*counter++
		out = append(out, koboWord(text[last:], uuid, *counter))
	}
	return out
}

func koboWord(word, uuid string, n int) *nethtml.Node {
	return &nethtml.Node{
		Type: nethtml.ElementNode,
		Data: "span",
		Attr: []nethtml.Attribute{
			{Key: "class", Val: "koboSpan"},
			{Key: "id", Val: fmt.Sprintf("kobo.%s.%d", uuid, n)},
		},
		FirstChild: &nethtml.Node{Type: nethtml.TextNode, Data: word},
	}
}
