package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestByteCacheCollapsesConcurrentLoads(t *testing.T) {
	bc, err := NewByteCache(8<<20, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Close()

	var loads atomic.Int64
	page := make([]byte, 64<<10)
	load := func() ([]byte, error) {
		loads.Add(1)
		time.Sleep(20 * time.Millisecond)
		return page, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := bc.GetOrLoad("book:page1", load)
			if err != nil {
				t.Error(err)
				return
			}
			if len(data) != len(page) {
				t.Errorf("got %d bytes, want %d", len(data), len(page))
			}
		}()
	}
	wg.Wait()

	if got := loads.Load(); got != 1 {
		t.Fatalf("50 concurrent readers triggered %d decompressions, want 1", got)
	}

	for i := 0; i < 10; i++ {
		if _, err := bc.GetOrLoad("book:page1", load); err != nil {
			t.Fatal(err)
		}
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("cached reads triggered %d more decompressions, want 0", got-1)
	}
}

func TestByteCacheKeyIsolatesVersions(t *testing.T) {
	bc, err := NewByteCache(8<<20, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Close()

	stale := []byte("old page bytes")
	fresh := []byte("rescanned page bytes")

	got, err := bc.GetOrLoad(BuildKey("asset", "file-1", "2026-01-01", "page1.jpg"), func() ([]byte, error) {
		return stale, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(stale) {
		t.Fatalf("got %q, want %q", got, stale)
	}

	got, err = bc.GetOrLoad(BuildKey("asset", "file-1", "2026-06-01", "page1.jpg"), func() ([]byte, error) {
		return fresh, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(fresh) {
		t.Fatalf("mod_time changed but got %q, want %q", got, fresh)
	}
}

func TestByteCacheDoesNotCacheFailures(t *testing.T) {
	bc, err := NewByteCache(8<<20, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Close()

	boom := errors.New("corrupt archive")
	if _, err := bc.GetOrLoad("k", func() ([]byte, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
	data, err := bc.GetOrLoad("k", func() ([]byte, error) { return []byte("recovered"), nil })
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "recovered" {
		t.Fatalf("got %q, want recovered", data)
	}
}
