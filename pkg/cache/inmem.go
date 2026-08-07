package cache

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type item[T any] struct {
	value     T
	expiresAt *time.Time
}

// timeGetter is a function that returns the current time.
type timeGetter func() time.Time

type InMemoryCache[T any] struct {
	cache      sync.Map
	timeGetter timeGetter
}

type InMemoryCacheOption[T any] func(*InMemoryCache[T])

// WithTimeGetter replaces the time source used to set and check expiry, so that callers can
// exercise TTL behavior without waiting on the wall clock.
func WithTimeGetter[T any](now timeGetter) InMemoryCacheOption[T] {
	return func(c *InMemoryCache[T]) {
		if now != nil {
			c.timeGetter = now
		}
	}
}

// NewInMemoryCache creates a new in-memory cache.
// The underlying cache implementation uses a sync.Map so it is thread-safe.
func NewInMemoryCache[T any](opts ...InMemoryCacheOption[T]) *InMemoryCache[T] {
	c := &InMemoryCache[T]{
		timeGetter: time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}

	return c
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

	if i.expiresAt != nil && i.expiresAt.Before(c.timeGetter()) {
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

	return nil
}

func (c *InMemoryCache[T]) Delete(ctx context.Context, key string) error {
	c.cache.Delete(key)

	return nil
}

func (c *InMemoryCache[T]) Clear(ctx context.Context) error {
	c.cache.Range(func(key, _ any) bool {
		c.cache.Delete(key)
		return true
	})

	return nil
}

var _ Cache[any] = &InMemoryCache[any]{}
