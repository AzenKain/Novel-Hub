package convert

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

func EncodeCursor(v any, id string) string {
	var valStr string
	switch t := v.(type) {
	case time.Time:
		valStr = t.Format(time.RFC3339Nano)
	default:
		valStr = fmt.Sprintf("%v", v)
	}
	raw := valStr + "," + id
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor splits on the LAST comma, not the first: the sort value comes first and can be
// free text, while the id is always a UUID. Splitting on the first comma stranded pagination on
// any author or publisher whose name contains one — and "Surname, Given" is the standard
// dc:creator form, so that was the norm rather than an edge case. A nil return makes the caller
// fall back to "no cursor", which serves page 1 again and loops infinite scroll forever.
func DecodeCursor(cursorStr string) []string {
	if cursorStr == "" {
		return nil
	}
	bytes, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil
	}
	raw := string(bytes)
	separator := strings.LastIndex(raw, ",")
	if separator < 0 {
		return nil
	}
	return []string{raw[:separator], raw[separator+1:]}
}

// CursorTimeString converts a decoded cursor's time half into the "YYYY-MM-DD HH:MM:SS" text
// SQLite stores, which is what the keyset predicates compare against so the created_at index
// can be seeked. Cursors travel over the wire as RFC3339, where 'T' sorts above ' ' and the
// comparison would quietly match nothing but the first page. Unparseable input is returned
// unchanged so a hand-made cursor still behaves as before rather than becoming page 1.
func CursorTimeString(v string) string {
	if v == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	return v
}
