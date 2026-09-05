package ebookconv

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	nethtml "golang.org/x/net/html"

	"novelhub/pkg/bookparser"
)

func imageLookup(images []Image) map[string]int {
	m := make(map[string]int, len(images)*2)
	for i, img := range images {
		m[base(img.Src)] = i
		m[base(img.Name)] = i
	}
	return m
}

func base(src string) string {
	src = strings.SplitN(src, "?", 2)[0]
	src = strings.SplitN(src, "#", 2)[0]
	if decoded, err := url.PathUnescape(src); err == nil {
		src = decoded
	}
	return strings.ToLower(filepath.Base(src))
}

func fileAttr(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func splitAuthor(author string) (string, string) {
	author = strings.TrimSpace(author)
	if author == "" {
		return "", ""
	}
	parts := strings.Fields(author)
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func fallbackTitle(book *bookparser.BookData) string {
	if strings.TrimSpace(book.Metadata.Title) != "" {
		return strings.TrimSpace(book.Metadata.Title)
	}
	for _, ch := range book.Chapters {
		if strings.TrimSpace(ch.Title) != "" {
			return strings.TrimSpace(ch.Title)
		}
	}
	return "Untitled"
}

func newID() string {
	return uuid.NewString()
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func mediaTypeForBytes(data []byte, hint string) string {
	if hint != "" {
		return hint
	}
	return http.DetectContentType(data)
}
