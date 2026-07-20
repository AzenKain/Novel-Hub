package jsonx

import "github.com/bytedance/sonic"

func Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}
