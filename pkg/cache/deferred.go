package cache

import (
	"context"
	"sync"
)

// DeferredCache buffers invalidations and replays them deduplicated on Flush. Issuing them inline
// costs the SQLite write lock for the length of a full-cache scan, once per row; deferring also
// closes the window where a concurrent reader re-caches the pre-commit value after the Del.
type DeferredCache struct {
	Cache
	mu       sync.Mutex
	keys     map[string]struct{}
	patterns map[string]struct{}
}

func NewDeferred(c Cache) *DeferredCache {
	return &DeferredCache{
		Cache:    c,
		keys:     make(map[string]struct{}),
		patterns: make(map[string]struct{}),
	}
}

func (d *DeferredCache) Del(ctx context.Context, keys ...string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, k := range keys {
		d.keys[k] = struct{}{}
	}
	return nil
}

func (d *DeferredCache) DelByPattern(ctx context.Context, pattern string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.patterns[pattern] = struct{}{}
	return nil
}

func (d *DeferredCache) Flush(ctx context.Context) {
	d.mu.Lock()
	keys := make([]string, 0, len(d.keys))
	for k := range d.keys {
		keys = append(keys, k)
	}
	patterns := make([]string, 0, len(d.patterns))
	for p := range d.patterns {
		patterns = append(patterns, p)
	}
	d.keys = make(map[string]struct{})
	d.patterns = make(map[string]struct{})
	d.mu.Unlock()

	if len(keys) > 0 {
		_ = d.Cache.Del(ctx, keys...)
	}
	for _, p := range patterns {
		_ = d.Cache.DelByPattern(ctx, p)
	}
}
