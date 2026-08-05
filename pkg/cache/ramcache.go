package cache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Yiling-J/theine-go"
	"golang.org/x/sync/singleflight"

	"novelhub/pkg/config"
	"novelhub/pkg/jsonx"
)

var ErrCacheMiss = errors.New("cache miss")

const (
	defaultMaxCost     int64 = 64 << 20
	minAutoMaxCost     int64 = 128 << 20
	autoMaxCostDivisor int64 = 16
)

type CacheStats struct {
	Hits       uint64  `json:"hits"`
	Misses     uint64  `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	MaxCost    int64   `json:"max_cost"`
	EntryCount int64   `json:"entry_count"`
}

type Cache interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, dest any) error
	Del(ctx context.Context, keys ...string) error
	DelByPattern(ctx context.Context, pattern string) error
	MGet(ctx context.Context, keys ...string) [][]byte
	MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
	GetOrFetch(ctx context.Context, key string, dest any, ttl time.Duration, fetcher func() (any, error)) error
	Stats() CacheStats
}

type RamCache struct {
	items    *theine.Cache[string, []byte]
	sf       singleflight.Group
	versions map[string]int64
	verMu    sync.RWMutex
	maxCost  int64
}

func NewRamCache() Cache {
	return NewTheineCache(autoMaxCost())
}

func NewTheineCache(maxCost int64) Cache {
	if maxCost <= 0 {
		maxCost = autoMaxCost()
	}
	items, err := theine.NewBuilder[string, []byte](maxCost).
		Cost(func(value []byte) int64 {
			if len(value) == 0 {
				return 1
			}
			return int64(len(value))
		}).
		Build()
	if err != nil {
		panic(err)
	}
	return &RamCache{
		items:    items,
		versions: make(map[string]int64),
		maxCost:  maxCost,
	}
}

func autoMaxCost() int64 {
	if configured := config.GetIntConfigWithDefault("CACHE_MAX_COST_BYTES", 0); configured > 0 {
		return int64(configured)
	}
	total := systemMemoryBytes()
	if total <= 0 {
		return defaultMaxCost
	}
	maxCost := total / autoMaxCostDivisor
	if maxCost < minAutoMaxCost {
		return minAutoMaxCost
	}
	return maxCost
}

func systemMemoryBytes() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "MemTotal:" {
			continue
		}
		kib, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return kib * 1024
	}
	return 0
}

func (r *RamCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := jsonx.Marshal(value)
	if err != nil {
		return err
	}
	if ok := r.items.SetWithTTL(key, data, int64(len(data)), ttl); !ok {
		return errors.New("cache set rejected")
	}
	return nil
}

func (r *RamCache) Get(ctx context.Context, key string, dest any) error {
	data, ok := r.items.Get(key)
	if !ok {
		return ErrCacheMiss
	}
	return jsonx.Unmarshal(data, dest)
}

func (r *RamCache) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		r.items.Delete(key)
	}
	return nil
}

func (r *RamCache) DelByPattern(ctx context.Context, pattern string) error {
	keys := make([]string, 0)
	r.items.Range(func(key string, value []byte) bool {
		matched, err := filepath.Match(pattern, key)
		if err == nil && matched {
			keys = append(keys, key)
		}
		return true
	})
	for _, key := range keys {
		r.items.Delete(key)
	}
	return nil
}

func (r *RamCache) MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error {
	for key, value := range pairs {
		if err := r.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

func (r *RamCache) MGet(ctx context.Context, keys ...string) [][]byte {
	results := make([][]byte, len(keys))
	for i, key := range keys {
		data, ok := r.items.Get(key)
		if !ok {
			continue
		}
		results[i] = append([]byte(nil), data...)
	}
	return results
}

func (r *RamCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := r.items.Get(key)
	return ok, nil
}

func (r *RamCache) GetOrFetch(ctx context.Context, key string, dest any, ttl time.Duration, fetcher func() (any, error)) error {
	if err := r.Get(ctx, key, dest); err == nil {
		return nil
	}
	res, err, _ := r.sf.Do(key, func() (any, error) {
		val, fetchErr := fetcher()
		if fetchErr != nil {
			return nil, fetchErr
		}
		data, marshalErr := jsonx.Marshal(val)
		if marshalErr != nil {
			return nil, marshalErr
		}
		_ = r.items.SetWithTTL(key, data, int64(len(data)), ttl)
		return data, nil
	})
	if err != nil {
		return err
	}
	data, ok := res.([]byte)
	if !ok {
		return errors.New("invalid cache payload")
	}
	return jsonx.Unmarshal(data, dest)
}

func (r *RamCache) Stats() CacheStats {
	st := r.items.Stats()
	return CacheStats{
		Hits:       st.Hits(),
		Misses:     st.Misses(),
		HitRate:    st.HitRatio(),
		MaxCost:    r.maxCost,
		EntryCount: int64(r.items.Len()),
	}
}
