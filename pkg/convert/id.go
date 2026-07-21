package convert

import (
	"fmt"
	"strconv"
)

func ParseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid ID: %s", value)
	}
	return id, nil
}
