package cache

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

func TestInMemoryCache(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		key               string
		value             string
		ttl               time.Duration
		waitForExpiration bool
		want              any
	}{
		{
			name:  "Set and Get existing item without TTL",
			key:   "existing",
			value: "value",
			ttl:   0,
		},
		{
			name:  "Set and Get existing item with TTL",
			key:   "existingWithTTL",
			value: "value",
			ttl:   1 * time.Hour,
			want:  "value",
		},
		{
			name:              "Set and Get item which expired",
			key:               "existingWithTTL",
			value:             "value",
			ttl:               100 * time.Millisecond,
			waitForExpiration: true,
			want:              "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewInMemoryCache[string]()

			err := cache.Set(ctx, tt.key, tt.value, tt.ttl)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.ttl == 0 {
				// Set should not have cached the item if TTL is 0
				_, err := cache.Get(ctx, tt.key)
				assert.ErrorIs(t, err, ErrNotFound)
				return
			}
			if tt.waitForExpiration {
				// Assert that a not found error eventually gets returned
				assert.Eventually(t, func() bool {
					_, err := cache.Get(ctx, tt.key)
					return errors.Is(err, ErrNotFound)
				}, 300*time.Millisecond, 30*time.Millisecond)
				// Assert that any subsequent Get calls return a not found error
				_, err := cache.Get(ctx, tt.key)
				assert.Equal(t, ErrNotFound, err)

				return
			}
			got, err := cache.Get(ctx, tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
