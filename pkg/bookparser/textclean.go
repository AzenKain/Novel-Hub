package bookparser

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var malformedOfficeBulletPrefix = regexp.MustCompile(`^(?:[□�]\s*\?\s*|[ðï]\s*\?\s*)+`)

func CleanOfficeTextLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(
		"\uf0b7", "•",
		"\uf0a7", "•",
		"\uf0d8", "•",
		"\uf0fc", "•",
		"\uf06c", "•",
		"\u2022", "•",
		"", "•",
		"", "•",
		"□?", "• ",
		"□ ?", "• ",
		"�?", "• ",
		"ð?", "• ",
		"ï?", "• ",
	).Replace(value)
	value = malformedOfficeBulletPrefix.ReplaceAllString(value, "• ")
	value = strings.Join(strings.Fields(value), " ")
	if strings.HasPrefix(value, "•") {
		value = "• " + strings.TrimSpace(strings.TrimPrefix(value, "•"))
	}
	return strings.TrimSpace(value)
}

func CleanChapterTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}

	if strings.Contains(title, "%") {
		if unescaped, err := url.PathUnescape(title); err == nil && unescaped != "" {
			title = unescaped
		} else if unescaped, err := url.QueryUnescape(title); err == nil && unescaped != "" {
			title = unescaped
		}
	}

	title = html.UnescapeString(title)
	return strings.Join(strings.Fields(strings.TrimSpace(title)), " ")
}
