package bookparser

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestReadAllLimitRejectsExtraByte(t *testing.T) {
	if _, err := ReadAllLimit(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("expected oversized input to be rejected")
	}
	data, err := ReadAllLimit(strings.NewReader("1234"), 4)
	if err != nil || string(data) != "1234" {
		t.Fatalf("expected exact-limit input, got %q, %v", data, err)
	}
}

func TestValidateImageUsesContent(t *testing.T) {
	var data bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	ext, err := ValidateImage(data.Bytes(), int64(data.Len()))
	if err != nil || ext != ".png" {
		t.Fatalf("expected PNG, got %q, %v", ext, err)
	}
	if _, err := ValidateImage([]byte("<svg><script/></svg>"), 1024); err == nil {
		t.Fatal("expected active non-raster image to be rejected")
	}
}
