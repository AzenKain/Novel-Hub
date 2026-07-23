package bookparser

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const maxImagePixels = 40_000_000

func ValidateImage(data []byte, maxBytes int64) (string, error) {
	if len(data) == 0 || int64(len(data)) > maxBytes {
		return "", fmt.Errorf("image exceeds size limit")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return "", fmt.Errorf("invalid or oversized image")
	}
	switch format {
	case "jpeg":
		return ".jpg", nil
	case "png":
		return ".png", nil
	case "gif":
		return ".gif", nil
	default:
		return "", fmt.Errorf("unsupported image format")
	}
}
