package cache

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryCache_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		setup     func(cache *InMemoryCache[string])
		key       string
		want      any
		wantError error
	}{
		{
			name: "Get existing item without TTL",
			setup: func(c *InMemoryCache[string]) {
				i := &item[string]{
					value: "value",
				}
				c.cache.Store("existing", i)
			},
			key:       "existing",
			want:      "value",
			wantError: nil,
		},
		{
			name: "Get existing item with expired TTL",
			setup: func(cache *InMemoryCache[string]) {
				expiresAt := time.Now().Add(-1 * time.Hour)
				i := &item[string]{
					value:     "value",
					expiresAt: &expiresAt,
				}
				cache.cache.Store("stale", i)
			},
			key:       "stale",
			want:      nil,
			wantError: ErrNotFound,
		},
		{
			name:      "Get non-existing item",
			setup:     func(cache *InMemoryCache[string]) {},
			key:       "non-existing",
			want:      nil,
			wantError: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewInMemoryCache[string]()
			tt.setup(cache)
			got, err := cache.Get(ctx, tt.key)
			if tt.wantError != nil {
				assert.EqualError(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestInMemoryCache_Set(t *testing.T) {
	ctx := context.Background()
	staticTimeGetter := func() time.Time {
		return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name    string
		key     string
		value   string
		ttl     time.Duration
		wantErr error
	}{
		{
			name:    "Set item without TTL",
			key:     "key",
			value:   "value",
			wantErr: nil,
		},
		{
			name:    "Set item with TTL",
			key:     "key",
			value:   "value",
			ttl:     1 * time.Hour,
			wantErr: nil,
		},
		{
			name:    "Set item with infinite TTL",
			key:     "key",
			value:   "value",
			ttl:     InfiniteTTL(),
			wantErr: nil,
		},
		{
			name:    "Set item with invalid TTL",
			key:     "key",
			value:   "value",
			ttl:     -1 * time.Minute,
			wantErr: errors.New("ttl must be greater than 0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := InMemoryCache[string]{
				timeGetter: staticTimeGetter,
			}
			err := c.Set(ctx, tt.key, tt.value, tt.ttl)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				if tt.ttl == 0 {
					// Assert that the item is not expired
					_, err := c.Get(ctx, tt.key)
					assert.ErrorIs(t, err, ErrNotFound)
					return
				}
				rawItem, ok := c.cache.Load(tt.key)
				if !ok {
					t.Fatalf("expected item to be set in cache")
				}
				i, ok := rawItem.(*item[string])
				if !ok {
					t.Fatalf("unexpected item type found in cache")
				}
				assert.Equal(t, tt.value, i.value)
				if tt.ttl > 0 {
					if i.expiresAt == nil {
						t.Fatalf("expected item to have an expiry time")
					}
					assert.Equal(t, staticTimeGetter().Add(tt.ttl), *i.expiresAt)
				} else {
					assert.Nil(t, i.expiresAt)
				}
			}
		})
	}
}

func TestInMemoryCache_SetAndGet(t *testing.T) {
	ctx := context.Background()
	start := time.Now()

	tests := []struct {
		name    string
		ttl     time.Duration
		advance time.Duration
		want    string
		wantErr error
	}{
		{
			name:    "an item without a ttl is not cached at all",
			ttl:     0,
			wantErr: ErrNotFound,
		},
		{
			name:    "an item is served while the clock is within its ttl",
			ttl:     time.Hour,
			advance: time.Minute,
			want:    "value",
		},
		{
			name:    "an item is gone once the clock passes its ttl",
			ttl:     time.Hour,
			advance: 2 * time.Hour,
			wantErr: ErrNotFound,
		},
		{
			name:    "an item with an infinite ttl outlives any advance",
			ttl:     InfiniteTTL(),
			advance: 100 * 365 * 24 * time.Hour,
			want:    "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := start
			cache := NewInMemoryCache[string]()
			cache.timeGetter = func() time.Time { return now }

			require.NoError(t, cache.Set(ctx, "key", "value", tt.ttl))
			now = start.Add(tt.advance)

			got, err := cache.Get(ctx, "key")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInMemoryCache_DeleteAndClear(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		drop    func(cache *InMemoryCache[string]) error
		wantKey string
		want    string
		wantErr error
	}{
		{
			name:    "delete drops the named key",
			drop:    func(c *InMemoryCache[string]) error { return c.Delete(ctx, "first") },
			wantKey: "first",
			wantErr: ErrNotFound,
		},
		{
			name:    "delete leaves the other keys alone",
			drop:    func(c *InMemoryCache[string]) error { return c.Delete(ctx, "first") },
			wantKey: "second",
			want:    "second-value",
		},
		{
			name:    "delete of an absent key is not an error",
			drop:    func(c *InMemoryCache[string]) error { return c.Delete(ctx, "absent") },
			wantKey: "first",
			want:    "first-value",
		},
		{
			name:    "clear drops every key",
			drop:    func(c *InMemoryCache[string]) error { return c.Clear(ctx) },
			wantKey: "second",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewInMemoryCache[string]()
			require.NoError(t, cache.Set(ctx, "first", "first-value", time.Hour))
			require.NoError(t, cache.Set(ctx, "second", "second-value", time.Hour))

			require.NoError(t, tt.drop(cache))

			got, err := cache.Get(ctx, tt.wantKey)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
