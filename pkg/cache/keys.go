package cache

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"

	"novelhub/pkg/jsonx"
)

func BuildKey(parts ...any) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString(":")
		}
		appendPart(&sb, p)
	}
	return sb.String()
}

func appendPart(sb *strings.Builder, p any) {
	if p == nil {
		sb.WriteString("nil")
		return
	}

	switch v := p.(type) {
	case string:
		sb.WriteString(v)
	case int64:
		sb.WriteString(strconv.FormatInt(v, 10))
	case int:
		sb.WriteString(strconv.Itoa(v))
	case uint64:
		sb.WriteString(strconv.FormatUint(v, 10))
	case bool:
		if v {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case *time.Time:
		if v == nil {
			sb.WriteString("nil")
			return
		}
		sb.WriteString(v.Format(time.RFC3339Nano))
	case *string:
		if v == nil {
			sb.WriteString("nil")
			return
		}
		sb.WriteString(*v)
	case fmt.Stringer:
		sb.WriteString(v.String())
	default:
		fmt.Fprint(sb, p)
	}
}

func QueryKey(prefix string, params any) string {
	data, _ := jsonx.Marshal(params)
	return prefix + ":" + strconv.FormatUint(xxhash.Sum64(data), 16)
}

func QueryKeyParts(params any, prefixParts ...any) string {
	prefix := BuildKey(prefixParts...)
	return QueryKey(prefix, params)
}
