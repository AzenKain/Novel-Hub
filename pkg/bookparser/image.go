package bookparser

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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

func IsBlankImage(data []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 1 || h < 1 {
		return true
	}

	const maxSamples = 2500
	stepX := w / 50
	stepY := h / 50
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}

	total := 0
	nearWhite := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8

			gray, _, _, _ := color.GrayModel.Convert(img.At(x, y)).RGBA()
			gray8 := gray >> 8

			if (r8 > 245 && g8 > 245 && b8 > 245) || gray8 > 245 {
				nearWhite++
			}
			total++
			if total >= maxSamples {
				goto done
			}
		}
	}
done:
	if total == 0 {
		return true
	}
	return float64(nearWhite)/float64(total) > 0.97
}

func IsSuitablePDFCover(data []byte) bool {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 50 || cfg.Height < 50 || int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return false
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 1 || h < 1 {
		return false
	}

	const maxSamples = 2500
	stepX, stepY := max(w/50, 1), max(h/50, 1)
	nearWhite, total := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y && total < maxSamples; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X && total < maxSamples; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			if r>>8 > 240 && g>>8 > 240 && b>>8 > 240 {
				nearWhite++
			}
			total++
		}
	}

	if total == 0 {
		return false
	}
	// If image is mostly white background (>85% white)
	return float64(nearWhite)/float64(total) < 0.85
}
