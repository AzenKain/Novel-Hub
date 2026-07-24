package bookparser

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"testing"
)

func TestIsSuitablePDFCover(t *testing.T) {
	document := image.NewRGBA(image.Rect(0, 0, 600, 840))
	draw.Draw(document, document.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(document, image.Rect(80, 100, 520, 250), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)

	cover := image.NewRGBA(image.Rect(0, 0, 600, 900))
	draw.Draw(cover, cover.Bounds(), &image.Uniform{C: color.RGBA{R: 20, G: 80, B: 180, A: 255}}, image.Point{}, draw.Src)

	encode := func(img image.Image) []byte {
		var buffer bytes.Buffer
		if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}

	if IsSuitablePDFCover(encode(document)) {
		t.Fatal("mostly-white document page canvas must be rejected so default SVG cover is generated")
	}
	if !IsSuitablePDFCover(encode(cover)) {
		t.Fatal("colored portrait cover artwork should be accepted")
	}
}
