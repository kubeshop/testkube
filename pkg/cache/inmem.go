package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

type item[T any] struct {
	value     T
	expiresAt *time.Time
}

// timeGetter is a function that returns the current time.
type timeGetter func() time.Time

// flushInterval controls how often the cache is scanned for expired entries:
// one sweep is triggered every flushInterval calls to Set.
const flushInterval = 10

type InMemoryCache[T any] struct {
	cache      sync.Map
	timeGetter timeGetter
	setCount   atomic.Uint64
}

// NewInMemoryCache creates a new in-memory cache.
// The underlying cache implementation uses a sync.Map so it is thread-safe.
func NewInMemoryCache[T any]() *InMemoryCache[T] {
	return &InMemoryCache[T]{
		timeGetter: time.Now,
	}
}

func (c *InMemoryCache[T]) Get(ctx context.Context, key string) (T, error) {
	var defaultVal T
	rawItem, ok := c.cache.Load(key)
	if !ok {
		return defaultVal, ErrNotFound
	}
	i, ok := rawItem.(*item[T])
	if !ok {
		return defaultVal, errors.New("unexpected item type found in cache")
	}

	if i.expiresAt != nil && i.expiresAt.Before(time.Now()) {
		c.cache.Delete(key)
		return defaultVal, ErrNotFound
	}

	return i.value, nil
}

func (c *InMemoryCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if ttl < 0 {
		return errors.New("ttl must be greater than 0")
	}
	if ttl == 0 {
		return nil
	}

	i := &item[T]{
		value: value,
	}
	if ttl > 0 {
		expiresAt := c.timeGetter().Add(ttl)
		i.expiresAt = &expiresAt
	}
	c.cache.Store(key, i)

	// Periodically sweep the map for expired entries to prevent unbounded growth.
	if c.setCount.Add(1)%flushInterval == 0 {
		c.flush()
	}

	return nil
}

// flush removes all expired entries from the cache.
// It collects keys to delete first, then deletes them, to avoid modifying the
// map during Range iteration.
func (c *InMemoryCache[T]) flush() {
	now := c.timeGetter()
	var expired []any
	c.cache.Range(func(key, rawItem any) bool {
		i, ok := rawItem.(*item[T])
		if ok && i.expiresAt != nil && i.expiresAt.Before(now) {
			expired = append(expired, key)
		}
		return true
	})
	for _, key := range expired {
		c.cache.Delete(key)
	}
}

var _ Cache[any] = &InMemoryCache[any]{}
