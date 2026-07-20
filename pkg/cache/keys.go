package cache

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"

	"novelhub/pkg/jsonx"
)

func BuildKey(parts ...any) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString(":")
		}
		sb.WriteString(fmt.Sprint(p))
	}
	return sb.String()
}

func QueryKey(prefix string, params any) string {
	data, _ := jsonx.Marshal(params)
	return prefix + ":" + strconv.FormatUint(xxhash.Sum64(data), 16)
}

func QueryKeyParts(params any, prefixParts ...any) string {
	prefix := BuildKey(prefixParts...)
	return QueryKey(prefix, params)
}
