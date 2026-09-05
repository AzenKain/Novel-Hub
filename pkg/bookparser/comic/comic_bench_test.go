package comic

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func buildBenchCBZ(tb testing.TB, pages int) string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), fmt.Sprintf("bench-%d.cbz", pages))

	img := image.NewRGBA(image.Rect(0, 0, 800, 1200))
	for y := 0; y < 1200; y += 3 {
		for x := 0; x < 800; x += 3 {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: uint8(x ^ y), A: 255})
		}
	}
	var page bytes.Buffer
	if err := jpeg.Encode(&page, img, &jpeg.Options{Quality: 80}); err != nil {
		tb.Fatal(err)
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for i := 1; i <= pages; i++ {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:   fmt.Sprintf("page%03d.jpg", i),
			Method: zip.Store,
		})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write(page.Bytes()); err != nil {
			tb.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		tb.Fatal(err)
	}
	return path
}

// BenchmarkKomgaPageServe measures one /books/{id}/pages/{n} request as the Komga service issues it: ListImages to map the 1-based number onto a name, then GetAsset to read it.
func BenchmarkKomgaPageServe(b *testing.B) {
	for _, pages := range []int{50, 200, 800} {
		path := buildBenchCBZ(b, pages)
		parser := NewParser("cbz")

		b.Run(fmt.Sprintf("pages=%d", pages), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; b.Loop(); i++ {
				names, err := parser.ListImages(path)
				if err != nil {
					b.Fatal(err)
				}
				if _, err := parser.GetAsset(path, names[i%len(names)]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The two halves, separately: ListImages is pure overhead per page request since the mapping never changes for a given file.
func BenchmarkComicListImages(b *testing.B) {
	for _, pages := range []int{50, 200, 800} {
		path := buildBenchCBZ(b, pages)
		parser := NewParser("cbz")
		b.Run(fmt.Sprintf("pages=%d", pages), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := parser.ListImages(path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkComicGetAsset(b *testing.B) {
	for _, pages := range []int{50, 200, 800} {
		path := buildBenchCBZ(b, pages)
		parser := NewParser("cbz")
		names, err := parser.ListImages(path)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("pages=%d", pages), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; b.Loop(); i++ {
				if _, err := parser.GetAsset(path, names[i%len(names)]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// A reader swipes through a whole volume; this is what that costs end to end.
func BenchmarkKomgaFullVolumeRead(b *testing.B) {
	const pages = 200
	path := buildBenchCBZ(b, pages)
	parser := NewParser("cbz")

	b.ReportAllocs()
	for b.Loop() {
		for n := 1; n <= pages; n++ {
			names, err := parser.ListImages(path)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := parser.GetAsset(path, names[n-1]); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// What the cached page-name mapping buys: the list step happens once per file instead of once per page.
func BenchmarkKomgaPageServeWithCachedNames(b *testing.B) {
	for _, pages := range []int{50, 200, 800} {
		path := buildBenchCBZ(b, pages)
		parser := NewParser("cbz")
		names, err := parser.ListImages(path)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("pages=%d", pages), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; b.Loop(); i++ {
				if _, err := parser.GetAsset(path, names[i%len(names)]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
