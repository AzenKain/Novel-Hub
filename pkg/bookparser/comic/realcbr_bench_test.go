package comic

import (
	"os"
	"testing"
)

func realCBRPath(tb testing.TB) string {
	tb.Helper()
	path := os.Getenv("NOVELHUB_BENCH_CBR")
	if path == "" {
		tb.Skip("set NOVELHUB_BENCH_CBR to a .cbr file to run this benchmark")
	}
	if _, err := os.Stat(path); err != nil {
		tb.Skipf("NOVELHUB_BENCH_CBR=%s: %v", path, err)
	}
	return path
}

func BenchmarkRealCBRListImages(b *testing.B) {
	realCBR := realCBRPath(b)
	parser := NewParser("cbr")
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parser.ListImages(realCBR); err != nil {
			b.Fatal(err)
		}
	}
}

// RAR has no central directory usable for random access here, so a late page costs a scan from the start.
func BenchmarkRealCBRGetAssetFirstPage(b *testing.B) {
	realCBR := realCBRPath(b)
	parser := NewParser("cbr")
	names, err := parser.ListImages(realCBR)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parser.GetAsset(realCBR, names[0]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRealCBRGetAssetLastPage(b *testing.B) {
	realCBR := realCBRPath(b)
	parser := NewParser("cbr")
	names, err := parser.ListImages(realCBR)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("pages=%d", len(names))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parser.GetAsset(realCBR, names[len(names)-1]); err != nil {
			b.Fatal(err)
		}
	}
}

// The Komga page endpoint on a real 248MB CBR, with and without the cached name mapping.
func BenchmarkRealCBRPageServe(b *testing.B) {
	realCBR := realCBRPath(b)
	parser := NewParser("cbr")
	warm, err := parser.ListImages(realCBR)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("relist_each_page", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			names, err := parser.ListImages(realCBR)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := parser.GetAsset(realCBR, names[i%len(names)]); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("cached_names", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			if _, err := parser.GetAsset(realCBR, warm[i%len(warm)]); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Concurrency is where the per-page allocation actually bites: every reader in flight holds its own decompressed page plus whatever the archive reader allocated to reach it.
func BenchmarkRealCBRPageServeParallel(b *testing.B) {
	realCBR := realCBRPath(b)
	parser := NewParser("cbr")
	names, err := parser.ListImages(realCBR)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if _, err := parser.GetAsset(realCBR, names[i%len(names)]); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}
