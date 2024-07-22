package cache

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrNotFound = errors.New("item not found")
)

type Cache[T any] interface {
	// Get retrieves the cached value for the given key.
	// If the key is not found or expired, the method should return ErrNotFound.
	Get(ctx context.Context, key string) (T, error)
	// Set stores the value in the cache with the given key.
	// If ttl is 0, the item should not be cached and this method should return no error.
	Set(ctx context.Context, key string, value T, ttl time.Duration) error
}

// IsCacheMiss returns true if the error is a cache miss error.
// This is a helper function to determine so users don't have to compare errors manually.
func IsCacheMiss(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// InfiniteTTL returns a time.Duration that represents an infinite TTL.
func InfiniteTTL() time.Duration {
	return math.MaxInt64
}
