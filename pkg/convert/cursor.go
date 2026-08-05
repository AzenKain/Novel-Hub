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
