package bookparser

import (
	"fmt"
	"io"
)

func ReadAllLimit(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("archive entry exceeds %d bytes", max)
	}
	return data, nil
}
