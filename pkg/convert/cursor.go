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
	parts := strings.Split(string(bytes), ",")
	if len(parts) != 2 {
		return nil
	}
	return parts
}
