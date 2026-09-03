package calibre

import (
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

func EncodeName(s string) string {
	return hex.EncodeToString([]byte(s))
}

func DecodeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	bytes, err := hex.DecodeString(s)
	if err != nil || !utf8.Valid(bytes) {
		return s
	}
	return string(bytes)
}
