package secret

import (
	"context"
	"maps"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/cache"
)

// negativeCacheTTLDivisor shortens the lifetime of cached not found results, so that a secret
// created after a failed lookup is picked up sooner than a regular refresh.
const negativeCacheTTLDivisor = 3

// readCache adds what the store it writes into does not provide: a shorter lifetime for not
// found results, coalescing of concurrent misses, and a revision that lets a mutation discard
// reads it superseded.
type readCache[T any] struct {
	store       cache.Cache[cacheEntry[T]]
	ttl         time.Duration
	negativeTTL time.Duration
	reads       singleflight.Group

	mu       sync.Mutex
	revision uint64
}

// cacheEntry keeps the error alongside the value, since the store holds values rather than
// call results and a missing secret is worth remembering.
type cacheEntry[T any] struct {
	value T
	err   error
}

func newReadCache[T any](ttl time.Duration, now func() time.Time) *readCache[T] {
	return &readCache[T]{
		store:       cache.NewInMemoryCache[cacheEntry[T]](cache.WithTimeGetter[cacheEntry[T]](now)),
		ttl:         ttl,
		negativeTTL: ttl / negativeCacheTTLDivisor,
	}
}

func (c *readCache[T]) load(ctx context.Context, key string, read func() (T, error)) (T, error) {
	entry, revision, ok := c.get(ctx, key)
	if ok {
		return entry.value, entry.err
	}

	// The revision is part of the flight identity, not just of the decision to cache the
	// result. A caller that missed after an invalidation must not join a read that started
	// before it, because that read answers a question the mutation has already changed.
	shared, err, _ := c.reads.Do(strconv.FormatUint(revision, 10)+"/"+key, func() (any, error) {
		// A read that finished between the miss above and here has already populated the entry.
		if entry, _, ok := c.get(ctx, key); ok {
			return entry, entry.err
		}

		value, readErr := read()
		entry := cacheEntry[T]{value: value, err: readErr}
		c.set(ctx, key, revision, entry)

		return entry, readErr
	})

	result, _ := shared.(cacheEntry[T])

	return result.value, err
}

func (c *readCache[T]) get(ctx context.Context, key string) (cacheEntry[T], uint64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, err := c.store.Get(ctx, key)
	if err != nil {
		return cacheEntry[T]{}, c.revision, false
	}

	return entry, c.revision, true
}

// set caches a missing secret too, but never any other failure, which may be transient.
func (c *readCache[T]) set(ctx context.Context, key string, revision uint64, entry cacheEntry[T]) {
	if entry.err != nil && !k8serrors.IsNotFound(entry.err) {
		return
	}

	ttl := c.ttl
	if entry.err != nil {
		ttl = c.negativeTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// A result read before an invalidation describes a value the mutation already replaced.
	if c.revision != revision {
		return
	}

	_ = c.store.Set(ctx, key, entry, ttl)
}

func (c *readCache[T]) delete(ctx context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.revision++
	_ = c.store.Delete(ctx, key)
}

func (c *readCache[T]) clear(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.revision++
	_ = c.store.Clear(ctx)
}

// cachedClient decorates a secret client with a short lived read cache. Mutating methods are
// passed through and invalidate the entries they affect.
type cachedClient struct {
	inner     Interface
	namespace string
	values    *readCache[map[string]string]
	objects   *readCache[*v1.Secret]
}

// NewCachedClient wraps inner with a TTL cache for reads. A zero or negative ttl disables
// caching and returns inner unchanged. namespace must be the one inner falls back to, so that a
// read with an explicit namespace and one without share a cache entry.
func NewCachedClient(inner Interface, namespace string, ttl time.Duration) Interface {
	if ttl <= 0 {
		return inner
	}

	return newCachedClient(inner, namespace, ttl, time.Now)
}

func newCachedClient(inner Interface, namespace string, ttl time.Duration, now func() time.Time) *cachedClient {
	return &cachedClient{
		inner:     inner,
		namespace: namespace,
		values:    newReadCache[map[string]string](ttl, now),
		objects:   newReadCache[*v1.Secret](ttl, now),
	}
}

func (c *cachedClient) Get(id string, namespace ...string) (map[string]string, error) {
	ctx := context.Background()
	data, err := c.values.load(ctx, c.key(id, namespace...), func() (map[string]string, error) {
		return c.inner.Get(id, namespace...)
	})

	// Copy on the way out, so that neither the cache nor another caller sharing this read can
	// be mutated through the returned map.
	return maps.Clone(data), err
}

func (c *cachedClient) GetObject(id string) (*v1.Secret, error) {
	ctx := context.Background()
	object, err := c.objects.load(ctx, c.key(id), func() (*v1.Secret, error) {
		return c.inner.GetObject(id)
	})

	return object.DeepCopy(), err
}

// List is not cached: it does not resolve a single key, so there is nothing to key a cache
// entry by.
func (c *cachedClient) List(all bool, namespace string) (map[string]map[string]string, error) {
	return c.inner.List(all, namespace)
}

func (c *cachedClient) Create(id string, labels, stringData map[string]string, namespace ...string) error {
	defer c.invalidate(c.key(id, namespace...))

	return c.inner.Create(id, labels, stringData, namespace...)
}

func (c *cachedClient) Apply(id string, labels, stringData map[string]string) error {
	defer c.invalidate(c.key(id))

	return c.inner.Apply(id, labels, stringData)
}

func (c *cachedClient) Update(id string, labels, stringData map[string]string) error {
	defer c.invalidate(c.key(id))

	return c.inner.Update(id, labels, stringData)
}

func (c *cachedClient) Delete(id string) error {
	defer c.invalidate(c.key(id))

	return c.inner.Delete(id)
}

// DeleteAll drops the whole cache rather than invalidating individual keys, since the selector
// it applies to the inner client can affect names it never resolved a key for.
func (c *cachedClient) DeleteAll(selector string) error {
	defer func() {
		ctx := context.Background()
		c.values.clear(ctx)
		c.objects.clear(ctx)
	}()

	return c.inner.DeleteAll(selector)
}

// key resolves the namespace the way the inner client does for the same arguments.
func (c *cachedClient) key(id string, namespace ...string) string {
	ns := c.namespace
	if len(namespace) != 0 {
		ns = namespace[0]
	}

	return ns + "/" + id
}

func (c *cachedClient) invalidate(key string) {
	ctx := context.Background()
	c.values.delete(ctx, key)
	c.objects.delete(ctx, key)
}
