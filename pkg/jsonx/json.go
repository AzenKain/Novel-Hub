package jsonx

import "github.com/bytedance/sonic"

func Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}

func MarshalString(v any) (string, error) {
	return sonic.MarshalString(v)
}

func UnmarshalString(buf string, val any) error {
	return sonic.UnmarshalString(buf, val)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return sonic.ConfigDefault.MarshalIndent(v, prefix, indent)
}
