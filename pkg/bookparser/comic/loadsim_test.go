package comic

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// What N concurrent readers actually cost: wall clock, peak heap, and GC pressure.
func TestRealCBRConcurrentReaderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("load simulation")
	}
	realCBR := realCBRPath(t)
	parser := NewParser("cbr")
	names, err := parser.ListImages(realCBR)
	if err != nil {
		t.Fatal(err)
	}

	for _, readers := range []int{1, 10, 50} {
		const pagesEach = 10
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)

		start := time.Now()
		var wg sync.WaitGroup
		for r := 0; r < readers; r++ {
			wg.Add(1)
			go func(r int) {
				defer wg.Done()
				for p := 0; p < pagesEach; p++ {
					if _, err := parser.GetAsset(realCBR, names[(r*pagesEach+p)%len(names)]); err != nil {
						t.Error(err)
						return
					}
				}
			}(r)
		}
		wg.Wait()
		elapsed := time.Since(start)
		runtime.ReadMemStats(&after)

		total := readers * pagesEach
		t.Logf("readers=%-3d pages=%-4d wall=%-8s per_page=%-8s peak_heap=%4dMB alloc_total=%5dMB gc=%d",
			readers, total,
			elapsed.Round(time.Millisecond),
			(elapsed / time.Duration(total)).Round(time.Microsecond),
			after.HeapInuse/1024/1024,
			(after.TotalAlloc-before.TotalAlloc)/1024/1024,
			after.NumGC-before.NumGC)
	}
}
