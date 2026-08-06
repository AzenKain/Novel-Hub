package comic

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/nwaples/rardecode/v2"
)

// getRARAsset scans entries from the start every call, so a late page costs a full walk.
// rardecode.List + File.Open gives random access instead. Solid archives are the catch: their
// entries depend on preceding ones and Open refuses.
func TestRarRandomAccessBeatsSequentialScan(t *testing.T) {
	realCBR := realCBRPath(t)
	parser := NewParser("cbr")
	names, err := parser.ListImages(realCBR)
	if err != nil {
		t.Fatal(err)
	}

	files, err := rardecode.List(realCBR)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]*rardecode.File, len(files))
	solid := 0
	for _, f := range files {
		byName[f.Name] = f
		if f.Solid {
			solid++
		}
	}
	fmt.Printf("entries=%d solid=%d\n", len(files), solid)

	for _, idx := range []int{0, len(names) / 2, len(names) - 1} {
		name := names[idx]

		start := time.Now()
		seqData, err := parser.GetAsset(realCBR, name)
		if err != nil {
			t.Fatal(err)
		}
		seqCost := time.Since(start)

		f, ok := byName[name]
		if !ok {
			t.Fatalf("entry %q missing from List", name)
		}
		start = time.Now()
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("random access failed on %q: %v", name, err)
		}
		randData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		randCost := time.Since(start)

		if len(seqData) != len(randData) {
			t.Errorf("page %d: sequential %d bytes vs random %d bytes", idx+1, len(seqData), len(randData))
		}
		fmt.Printf("page %3d/%d  sequential=%-8s random=%-8s speedup=%.1fx\n",
			idx+1, len(names), seqCost.Round(time.Millisecond), randCost.Round(time.Millisecond),
			float64(seqCost)/float64(randCost))
	}
}

// 17ms even with random access says the cost is decompression, not seeking. If the entries are
// stored (already-JPEG pages usually are), that 17ms is pure I/O of a 1.4MB payload and the
// only real lever is not re-reading it.
func TestRarEntryCompressionAndIOFloor(t *testing.T) {
	realCBR := realCBRPath(t)
	files, err := rardecode.List(realCBR)
	if err != nil {
		t.Fatal(err)
	}

	var stored, packed int
	var packedBytes, unpackedBytes int64
	for _, f := range files {
		if f.IsDir {
			continue
		}
		if f.PackedSize >= f.UnPackedSize {
			stored++
		} else {
			packed++
		}
		packedBytes += f.PackedSize
		unpackedBytes += f.UnPackedSize
	}
	fmt.Printf("entries stored=%d compressed=%d ratio=%.3f (packed=%dMB unpacked=%dMB)\n",
		stored, packed, float64(packedBytes)/float64(unpackedBytes),
		packedBytes/1024/1024, unpackedBytes/1024/1024)

	// Second read of the same entry, with the OS page cache already warm.
	f := files[len(files)/2]
	for i := 0; i < 3; i++ {
		start := time.Now()
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		n, err := io.Copy(io.Discard, rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("  read #%d: %-8s %dKB\n", i+1, time.Since(start).Round(time.Millisecond), n/1024)
	}
}
