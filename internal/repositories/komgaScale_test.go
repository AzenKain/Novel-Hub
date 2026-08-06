package repositories

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"novelhub/pkg/cache"
)

// The 500x10 bench answers "is the query shape right". This answers "does it hold at library
// scale under real concurrency": 10k series x 10 books, N readers each paging through the
// catalogue the way Mihon does on first sync.
func TestKomgaCatalogueUnderConcurrentLoad(t *testing.T) {
	if os.Getenv("NOVELHUB_SCALE_TEST") == "" {
		t.Skip("set NOVELHUB_SCALE_TEST=1 to seed 100k books and run this")
	}
	const seriesCount, booksPerSeries = 10000, 10

	seedStart := time.Now()
	db := seedKomgaBench(t, seriesCount, booksPerSeries)
	t.Logf("seeded %d series x %d books = %d books in %s",
		seriesCount, booksPerSeries, seriesCount*booksPerSeries, time.Since(seedStart).Round(time.Millisecond))

	repo := NewKomgaRepository(db, cache.NewRamCache())
	ctx := context.Background()
	libs := []string{"lib-1"}

	const pageSize = 20
	page := func(offset int64) error {
		rows, err := repo.ListSeries(ctx, libs, "", pageSize, offset)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]string, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		_, err = repo.SeriesProgressByIDs(ctx, "u-1", ids, libs)
		return err
	}

	if err := page(0); err != nil {
		t.Fatal(err)
	}

	// Past the connection-pool ceiling (16) throughput falls off -- 50 readers on a 12-core box
	// measured 320 pg/s against 950 at 8. That is SQLite write-lock and pool contention, not a
	// query problem, so the curve is printed rather than a single number quoted as "the" figure.
	for _, readers := range []int{1, 8, 16, 50} {
		const pagesEach = 20
		var failures atomic.Int64
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
					offset := int64((r*7+p)%(seriesCount/pageSize)) * pageSize
					if err := page(offset); err != nil {
						failures.Add(1)
						return
					}
				}
			}(r)
		}
		wg.Wait()

		elapsed := time.Since(start)
		runtime.ReadMemStats(&after)
		if n := failures.Load(); n > 0 {
			t.Fatalf("%d page loads failed", n)
		}
		total := readers * pagesEach
		fmt.Printf("  readers=%-3d pages=%-5d wall=%-9s per_page=%-9s throughput=%5.0f pg/s alloc=%4dMB\n",
			readers, total, elapsed.Round(time.Millisecond),
			(elapsed / time.Duration(total)).Round(time.Microsecond),
			float64(total)/elapsed.Seconds(),
			(after.TotalAlloc-before.TotalAlloc)/1024/1024)
	}
}
