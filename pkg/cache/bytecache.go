package cache

import (
	"time"

	"github.com/Yiling-J/theine-go"
	"golang.org/x/sync/singleflight"

	"novelhub/pkg/config"
)

const (
	minByteCacheMaxCost     int64 = 32 << 20
	maxByteCacheMaxCost     int64 = 512 << 20
	byteCacheMaxCostDivisor int64 = 32
)

// Stores []byte verbatim under its own budget.
type ByteCache interface {
	GetOrLoad(key string, load func() ([]byte, error)) ([]byte, error)
	Close()
}

type byteCache struct {
	items *theine.Cache[string, []byte]
	sf    singleflight.Group
	ttl   time.Duration
}

func NewByteCache(maxCost int64, ttl time.Duration) (ByteCache, error) {
	if maxCost <= 0 {
		maxCost = autoByteCacheMaxCost()
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
		return nil, err
	}
	return &byteCache{items: items, ttl: ttl}, nil
}

func autoByteCacheMaxCost() int64 {
	if configured := config.GetIntConfigWithDefault("ASSET_CACHE_MAX_COST_BYTES", 0); configured > 0 {
		return int64(configured)
	}
	total := systemMemoryBytes()
	if total <= 0 {
		return minByteCacheMaxCost
	}
	maxCost := total / byteCacheMaxCostDivisor
	if maxCost < minByteCacheMaxCost {
		return minByteCacheMaxCost
	}
	if maxCost > maxByteCacheMaxCost {
		return maxByteCacheMaxCost
	}
	return maxCost
}

// The returned slice is the cached buffer itself -- shared with every concurrent reader and handed to fasthttp by fiber's Send without a copy -- so callers must not write to it.
func (b *byteCache) GetOrLoad(key string, load func() ([]byte, error)) ([]byte, error) {
	if data, ok := b.items.Get(key); ok {
		return data, nil
	}
	res, err, _ := b.sf.Do(key, func() (any, error) {
		if data, ok := b.items.Get(key); ok {
			return data, nil
		}
		data, loadErr := load()
		if loadErr != nil {
			return nil, loadErr
		}
		cost := int64(len(data))
		if cost == 0 {
			cost = 1
		}
		b.items.SetWithTTL(key, data, cost, b.ttl)
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]byte), nil
}

func (b *byteCache) Close() {
	b.items.Close()
}
