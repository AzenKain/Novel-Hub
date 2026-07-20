package cache

import (
	"strconv"

	"github.com/cespare/xxhash/v2"

	"novelhub/pkg/jsonx"
)

func QueryKey(prefix string, params any) string {
	data, _ := jsonx.Marshal(params)
	return prefix + ":" + strconv.FormatUint(xxhash.Sum64(data), 16)
}
