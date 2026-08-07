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

func CursorTimeString(v string) string {
	if v == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	return v
}
