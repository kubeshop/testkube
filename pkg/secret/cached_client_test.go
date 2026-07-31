package secret

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	testCacheTTL  = 30 * time.Second
	testNamespace = "testkube"
)

type fakeClock struct {
	current time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.current
}

func (c *fakeClock) advance(d time.Duration) {
	c.current = c.current.Add(d)
}

func TestCachedClient_Get(t *testing.T) {
	notFound := k8serrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "secret")
	throttled := k8serrors.NewTooManyRequestsError("try again later")

	type step struct {
		advance   time.Duration
		namespace []string
	}

	tests := []struct {
		name      string
		innerErr  error
		steps     []step
		wantCalls int
	}{
		{
			name:      "second read within the ttl is served from the cache",
			steps:     []step{{}, {advance: 10 * time.Second}},
			wantCalls: 1,
		},
		{
			name:      "read after the ttl reaches the inner client again",
			steps:     []step{{}, {advance: testCacheTTL + time.Second}},
			wantCalls: 2,
		},
		{
			name:      "explicit default namespace shares the entry with the implicit one",
			steps:     []step{{}, {namespace: []string{testNamespace}}},
			wantCalls: 1,
		},
		{
			name:      "a different namespace does not collide with the default one",
			steps:     []step{{}, {namespace: []string{"other"}}, {namespace: []string{"other"}}},
			wantCalls: 2,
		},
		{
			name:      "missing secret is remembered for the negative ttl",
			innerErr:  notFound,
			steps:     []step{{}, {advance: 5 * time.Second}},
			wantCalls: 1,
		},
		{
			name:      "missing secret is looked up again after the negative ttl",
			innerErr:  notFound,
			steps:     []step{{}, {advance: 11 * time.Second}},
			wantCalls: 2,
		},
		{
			name:      "transport failure is not cached",
			innerErr:  throttled,
			steps:     []step{{}, {advance: time.Second}},
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := NewMockInterface(gomock.NewController(t))
			clock := &fakeClock{current: time.Now()}
			calls := 0
			inner.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(id string, namespace ...string) (map[string]string, error) {
				calls++
				if tt.innerErr != nil {
					return nil, tt.innerErr
				}

				return map[string]string{"key": "value"}, nil
			}).AnyTimes()

			client := newCachedClient(inner, testNamespace, testCacheTTL, clock.Now)
			for _, s := range tt.steps {
				clock.advance(s.advance)

				data, err := client.Get("secret", s.namespace...)
				if tt.innerErr != nil {
					assert.Equal(t, tt.innerErr, err)
					continue
				}

				require.NoError(t, err)
				assert.Equal(t, map[string]string{"key": "value"}, data)
				// Poisoning the returned map must not reach the next read.
				data["key"] = "poisoned"
			}

			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestCachedClient_GetObject(t *testing.T) {
	tests := []struct {
		name      string
		advance   time.Duration
		wantCalls int
	}{
		{
			name:      "second read within the ttl is served from the cache",
			advance:   10 * time.Second,
			wantCalls: 1,
		},
		{
			name:      "read after the ttl reaches the inner client again",
			advance:   testCacheTTL + time.Second,
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := NewMockInterface(gomock.NewController(t))
			clock := &fakeClock{current: time.Now()}
			calls := 0
			inner.EXPECT().GetObject(gomock.Any()).DoAndReturn(func(id string) (*v1.Secret, error) {
				calls++
				return &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: id}}, nil
			}).AnyTimes()

			client := newCachedClient(inner, testNamespace, testCacheTTL, clock.Now)

			first, err := client.GetObject("secret")
			require.NoError(t, err)
			first.StringData = map[string]string{"changed": "true"}

			clock.advance(tt.advance)

			second, err := client.GetObject("secret")
			require.NoError(t, err)
			assert.Equal(t, "secret", second.Name)
			assert.Nil(t, second.StringData, "callers must not be able to modify the cached object")
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestCachedClient_Invalidation(t *testing.T) {
	tests := []struct {
		name      string
		expect    func(inner *MockInterface)
		mutate    func(client Interface) error
		wantCalls int
	}{
		{
			name:      "update drops the cached secret",
			expect:    func(inner *MockInterface) { inner.EXPECT().Update("secret", gomock.Any(), gomock.Any()).Return(nil) },
			mutate:    func(client Interface) error { return client.Update("secret", nil, nil) },
			wantCalls: 2,
		},
		{
			name:      "delete drops the cached secret",
			expect:    func(inner *MockInterface) { inner.EXPECT().Delete("secret").Return(nil) },
			mutate:    func(client Interface) error { return client.Delete("secret") },
			wantCalls: 2,
		},
		{
			name:      "apply drops the cached secret",
			expect:    func(inner *MockInterface) { inner.EXPECT().Apply("secret", gomock.Any(), gomock.Any()).Return(nil) },
			mutate:    func(client Interface) error { return client.Apply("secret", nil, nil) },
			wantCalls: 2,
		},
		{
			name: "create drops the cached secret",
			expect: func(inner *MockInterface) {
				inner.EXPECT().Create("secret", gomock.Any(), gomock.Any()).Return(nil)
			},
			mutate:    func(client Interface) error { return client.Create("secret", nil, nil) },
			wantCalls: 2,
		},
		{
			name: "create in another namespace keeps the cached secret",
			expect: func(inner *MockInterface) {
				inner.EXPECT().Create("secret", gomock.Any(), gomock.Any(), "other").Return(nil)
			},
			mutate:    func(client Interface) error { return client.Create("secret", nil, nil, "other") },
			wantCalls: 1,
		},
		{
			name:      "delete all drops the whole cache",
			expect:    func(inner *MockInterface) { inner.EXPECT().DeleteAll("").Return(nil) },
			mutate:    func(client Interface) error { return client.DeleteAll("") },
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := NewMockInterface(gomock.NewController(t))
			clock := &fakeClock{current: time.Now()}
			calls := 0
			inner.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(id string, namespace ...string) (map[string]string, error) {
				calls++
				return map[string]string{"key": "value"}, nil
			}).AnyTimes()
			tt.expect(inner)

			client := newCachedClient(inner, testNamespace, testCacheTTL, clock.Now)

			_, err := client.Get("secret")
			require.NoError(t, err)
			require.NoError(t, tt.mutate(client))

			_, err = client.Get("secret")
			require.NoError(t, err)
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestCachedClient_InvalidationDuringRead(t *testing.T) {
	tests := []struct {
		name   string
		expect func(inner *MockInterface)
		mutate func(client Interface) error
	}{
		{
			name:   "update while a read is in flight discards the read result",
			expect: func(inner *MockInterface) { inner.EXPECT().Update("secret", gomock.Any(), gomock.Any()).Return(nil) },
			mutate: func(client Interface) error { return client.Update("secret", nil, nil) },
		},
		{
			name:   "delete while a read is in flight discards the read result",
			expect: func(inner *MockInterface) { inner.EXPECT().Delete("secret").Return(nil) },
			mutate: func(client Interface) error { return client.Delete("secret") },
		},
		{
			name:   "delete all while a read is in flight discards the read result",
			expect: func(inner *MockInterface) { inner.EXPECT().DeleteAll("").Return(nil) },
			mutate: func(client Interface) error { return client.DeleteAll("") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := NewMockInterface(gomock.NewController(t))
			clock := &fakeClock{current: time.Now()}
			client := newCachedClient(inner, testNamespace, testCacheTTL, clock.Now)

			calls := 0
			inner.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(id string, namespace ...string) (map[string]string, error) {
				calls++
				// The mutation lands between the inner read and the write into the cache.
				if calls == 1 {
					require.NoError(t, tt.mutate(client))
				}

				return map[string]string{"key": "value"}, nil
			}).AnyTimes()
			tt.expect(inner)

			_, err := client.Get("secret")
			require.NoError(t, err)

			_, err = client.Get("secret")
			require.NoError(t, err)
			assert.Equal(t, 2, calls)
		})
	}
}

func TestNewCachedClient(t *testing.T) {
	tests := []struct {
		name      string
		ttl       time.Duration
		wantCalls int
	}{
		{
			name:      "zero ttl bypasses the cache",
			ttl:       0,
			wantCalls: 2,
		},
		{
			name:      "negative ttl bypasses the cache",
			ttl:       -time.Second,
			wantCalls: 2,
		},
		{
			name:      "positive ttl caches reads",
			ttl:       testCacheTTL,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := NewMockInterface(gomock.NewController(t))
			calls := 0
			inner.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(id string, namespace ...string) (map[string]string, error) {
				calls++
				return map[string]string{"key": "value"}, nil
			}).AnyTimes()

			client := NewCachedClient(inner, testNamespace, tt.ttl)
			if tt.ttl <= 0 {
				assert.Same(t, inner, client)
			}

			for range 2 {
				_, err := client.Get("secret")
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

// TestCachedClient_CoalescesConcurrentMisses covers the burst this cache exists for: webhook
// listeners resolve the same parameter at once, so a cold key must produce one read rather
// than one per caller.
func TestCachedClient_CoalescesConcurrentMisses(t *testing.T) {
	const callers = 5

	tests := []struct {
		name   string
		expect func(inner *MockInterface, block func())
		read   func(client Interface) error
	}{
		{
			name: "Get",
			expect: func(inner *MockInterface, block func()) {
				inner.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(id string, namespace ...string) (map[string]string, error) {
					block()
					return map[string]string{"key": "value"}, nil
				}).AnyTimes()
			},
			read: func(client Interface) error {
				_, err := client.Get("secret")
				return err
			},
		},
		{
			name: "GetObject",
			expect: func(inner *MockInterface, block func()) {
				inner.EXPECT().GetObject(gomock.Any()).DoAndReturn(func(id string) (*v1.Secret, error) {
					block()
					return &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: id}}, nil
				}).AnyTimes()
			},
			read: func(client Interface) error {
				_, err := client.GetObject("secret")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			entered := make(chan struct{}, 1)
			release := make(chan struct{})

			inner := NewMockInterface(gomock.NewController(t))
			tt.expect(inner, func() {
				if calls.Add(1) == 1 {
					entered <- struct{}{}
					<-release
				}
			})

			client := newCachedClient(inner, testNamespace, testCacheTTL, time.Now)

			var wg sync.WaitGroup
			read := func() {
				defer wg.Done()
				assert.NoError(t, tt.read(client))
			}

			wg.Add(1)
			go read()
			// The first caller is now inside the inner client, so its read is registered and
			// the callers started below can join it instead of starting their own.
			<-entered

			for range callers - 1 {
				wg.Add(1)
				go read()
			}

			// Let those callers reach the in-flight read before it completes.
			time.Sleep(20 * time.Millisecond)
			close(release)
			wg.Wait()

			assert.Equal(t, int32(1), calls.Load())
		})
	}
}

// TestCachedClient_ReadAfterInvalidationDoesNotJoinOlderFlight pins the ordering that coalescing
// makes possible: a mutation completes while a read is still in flight, and the read that arrives
// afterwards must see the mutation rather than the value the older read is about to return.
func TestCachedClient_ReadAfterInvalidationDoesNotJoinOlderFlight(t *testing.T) {
	var reads atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})

	inner := NewMockInterface(gomock.NewController(t))
	inner.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(id string, namespace ...string) (map[string]string, error) {
		if reads.Add(1) == 1 {
			entered <- struct{}{}
			<-release
			return map[string]string{"key": "before"}, nil
		}

		return map[string]string{"key": "after"}, nil
	}).AnyTimes()
	inner.EXPECT().Update("secret", gomock.Any(), gomock.Any()).Return(nil)

	client := newCachedClient(inner, testNamespace, testCacheTTL, time.Now)

	var first sync.WaitGroup
	first.Add(1)
	go func() {
		defer first.Done()
		data, err := client.Get("secret")
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"key": "before"}, data, "the in-flight read keeps the value it started with")
	}()

	// The first read is now inside the inner client and its flight is registered.
	<-entered
	require.NoError(t, client.Update("secret", nil, nil))

	data, err := client.Get("secret")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"key": "after"}, data, "a read starting after the update must not receive the superseded value")

	close(release)
	first.Wait()
	assert.Equal(t, int32(2), reads.Load())
}
