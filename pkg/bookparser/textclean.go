package bookparser

import (
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
