package comic

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/singleflight"

	"novelhub/pkg/cache"
)

// RAR decode is ~20ms of pure CPU per page (disk floor is 0.2ms, ratio 0.982 so the compression
// buys nothing). With C readers on a box with N cores, throughput is capped at N/0.02 pages/sec
// no matter how the archive is opened — unless a page is decoded once and reused.
//
// This measures the three levers against that ceiling, on the access pattern Mihon actually
// produces: readers clustered on the same volume, each prefetching a few pages ahead.
func TestPageDeliveryUnderRealisticLoad(t *testing.T) {
	realCBR := realCBRPath(t)
	parser := NewParser("cbr")
	names, err := parser.ListImages(realCBR)
	if err != nil {
		t.Fatal(err)
	}

	type strategy struct {
		name string
		get  func(string, *int64) ([]byte, error)
	}

	sfg := &singleflight.Group{}
	byteCache, err := cache.NewByteCache(128<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer byteCache.Close()
	jsonCache := cache.NewTheineCache(128 << 20)
	jsonSfg := &singleflight.Group{}

	strategies := []strategy{
		{"baseline", func(n string, decodes *int64) ([]byte, error) {
			atomic.AddInt64(decodes, 1)
			return parser.GetAsset(realCBR, n)
		}},
		{"singleflight", func(n string, decodes *int64) ([]byte, error) {
			v, err, _ := sfg.Do(n, func() (any, error) {
				atomic.AddInt64(decodes, 1)
				return parser.GetAsset(realCBR, n)
			})
			if err != nil {
				return nil, err
			}
			return v.([]byte), nil
		}},
		{"json cache+sfg", func(n string, decodes *int64) ([]byte, error) {
			key := "pg:" + n
			var hit []byte
			if err := jsonCache.Get(context.Background(), key, &hit); err == nil && len(hit) > 0 {
				return hit, nil
			}
			v, err, _ := jsonSfg.Do(n, func() (any, error) {
				atomic.AddInt64(decodes, 1)
				data, err := parser.GetAsset(realCBR, n)
				if err != nil {
					return nil, err
				}
				_ = jsonCache.Set(context.Background(), key, data, time.Hour)
				return data, nil
			})
			if err != nil {
				return nil, err
			}
			return v.([]byte), nil
		}},
		{"ByteCache (shipped)", func(n string, decodes *int64) ([]byte, error) {
			return byteCache.GetOrLoad(n, func() ([]byte, error) {
				atomic.AddInt64(decodes, 1)
				return parser.GetAsset(realCBR, n)
			})
		}},
	}

	// 50 readers, 20 pages each, all on one popular volume with overlapping windows —
	// 1000 requests over ~60 distinct pages.
	const readers, pagesEach = 50, 20
	fmt.Printf("\n%d readers x %d pages on one volume (%d requests, ~60 distinct pages)\n",
		readers, pagesEach, readers*pagesEach)

	for _, s := range strategies {
		var decodes int64
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()

		var wg sync.WaitGroup
		for r := 0; r < readers; r++ {
			wg.Add(1)
			go func(r int) {
				defer wg.Done()
				offset := (r % 10) * 4
				for p := 0; p < pagesEach; p++ {
					if _, err := s.get(names[(offset+p)%len(names)], &decodes); err != nil {
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
		fmt.Printf("  %-20s wall=%-8s p50_est=%-9s decodes=%-5d throughput=%4.0f pg/s alloc=%5dMB gc=%d\n",
			s.name, elapsed.Round(time.Millisecond),
			(elapsed / time.Duration(total)).Round(time.Microsecond),
			decodes,
			float64(total)/elapsed.Seconds(),
			(after.TotalAlloc-before.TotalAlloc)/1024/1024,
			after.NumGC-before.NumGC)
	}
}
