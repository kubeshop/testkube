package watchers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// A watch the API server has hung up on, which is what a client sees once its
// resourceVersion has been aged out. Every Watch returns an already-closed channel.
type closingPodClient struct {
	mu         sync.Mutex
	watchCalls int32
	listCalls  int32
	watchRVs   []string
	listRVs    []string
}

func (c *closingPodClient) List(_ context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	atomic.AddInt32(&c.listCalls, 1)
	c.mu.Lock()
	c.listRVs = append(c.listRVs, opts.ResourceVersion)
	c.mu.Unlock()
	return &corev1.PodList{
		ListMeta: metav1.ListMeta{ResourceVersion: "1000"},
		Items: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", ResourceVersion: "1000"},
		}},
	}, nil
}

func (c *closingPodClient) Watch(_ context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	atomic.AddInt32(&c.watchCalls, 1)
	c.mu.Lock()
	c.watchRVs = append(c.watchRVs, opts.ResourceVersion)
	c.mu.Unlock()
	ch := make(chan watch.Event)
	close(ch) // the server hung up, exactly as it does on "too old resourceVersion"
	return &closedPodWatcher{ch: ch}, nil
}

type closedPodWatcher struct{ ch chan watch.Event }

func (w *closedPodWatcher) Stop()                          {}
func (w *closedPodWatcher) ResultChan() <-chan watch.Event { return w.ch }

func TestPodWatcher_RecoversFromRejectedResourceVersion(t *testing.T) {
	client := &closingPodClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	NewPodWatcher(ctx, client, metav1.ListOptions{ResourceVersion: "500"}, func(*corev1.Pod) {})

	// Longer than the one second base delay, so the window covers a real recovery list rather
	// than only the first failed watch.
	const window = 1400 * time.Millisecond
	time.Sleep(window)
	cancel()
	time.Sleep(20 * time.Millisecond)

	watches := atomic.LoadInt32(&client.watchCalls)
	lists := atomic.LoadInt32(&client.listCalls)
	rate := float64(watches) / window.Seconds()

	client.mu.Lock()
	distinctWatchRV := map[string]int{}
	for _, rv := range client.watchRVs {
		distinctWatchRV[rv]++
	}
	listRVs := append([]string(nil), client.listRVs...)
	client.mu.Unlock()

	t.Logf("watch calls in %v: %d  (~%.0f/s)", window, watches, rate)
	t.Logf("list calls (the relist that should recover): %d", lists)
	t.Logf("resourceVersions sent on watch: %v", distinctWatchRV)
	t.Logf("resourceVersions sent on list:  %v", listRVs)

	// A dead watch must be paced, not respun as fast as the loop runs.
	assert.Less(t, watches, int32(10),
		"watch was recreated %d times in %v (~%.0f/s) with no backoff", watches, window, rate)

	// Recovery must re-enter through the list with a fresh version, or the next watch is
	// rejected on the same one.
	assert.Greater(t, lists, int32(1), "the watcher never listed again after the watch died")
	for _, rv := range listRVs[1:] {
		assert.Empty(t, rv, "a recovery list reused resourceVersion %q instead of resetting it", rv)
	}
}

// A resourceVersion the API server has aged out does not close the stream: it arrives as a
// watch.Error carrying a Status. The object fails the *corev1.Pod assertion, so before this was
// handled the rejection was skipped and the same version was watched again.
type expiredRVPodClient struct {
	mu       sync.Mutex
	listRVs  []string
	watchRVs []string
}

func (c *expiredRVPodClient) List(_ context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	c.mu.Lock()
	c.listRVs = append(c.listRVs, opts.ResourceVersion)
	c.mu.Unlock()
	return &corev1.PodList{
		ListMeta: metav1.ListMeta{ResourceVersion: "1000"},
		Items:    []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", ResourceVersion: "1000"}}},
	}, nil
}

func (c *expiredRVPodClient) Watch(_ context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	c.mu.Lock()
	c.watchRVs = append(c.watchRVs, opts.ResourceVersion)
	c.mu.Unlock()
	ch := make(chan watch.Event, 1)
	ch <- watch.Event{
		Type:   watch.Error,
		Object: &apierrors.NewResourceExpired("too old resource version").ErrStatus,
	}
	return &closedPodWatcher{ch: ch}, nil
}

func TestPodWatcher_TreatsExpiredResourceVersionAsAFailure(t *testing.T) {
	client := &expiredRVPodClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	NewPodWatcher(ctx, client, metav1.ListOptions{ResourceVersion: "500"}, func(*corev1.Pod) {})

	const window = 1400 * time.Millisecond
	time.Sleep(window)
	cancel()
	time.Sleep(20 * time.Millisecond)

	client.mu.Lock()
	watchRVs := append([]string(nil), client.watchRVs...)
	listRVs := append([]string(nil), client.listRVs...)
	client.mu.Unlock()

	t.Logf("watch attempts: %d, list attempts: %d", len(watchRVs), len(listRVs))

	// The rejection must be paced like any other failure, not respun.
	assert.Less(t, len(watchRVs), 10,
		"a rejected resourceVersion was retried %d times in %v", len(watchRVs), window)

	// And it must send the watcher back through the list for a fresh version.
	assert.Greater(t, len(listRVs), 1, "the watcher never listed again after the version was rejected")
	for _, rv := range listRVs[1:] {
		assert.Empty(t, rv, "a recovery list reused resourceVersion %q", rv)
	}
}

// An object that has not changed keeps its own version while the list moves on. The recovery
// list must still adopt the list version, or the next watch returns to the rejected one.
type unchangedPodClient struct {
	mu       sync.Mutex
	watchRVs []string
	listRV   int32
}

func (c *unchangedPodClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.PodList, error) {
	// The list version advances on every call; the object's own version never does.
	n := atomic.AddInt32(&c.listRV, 1)
	return &corev1.PodList{
		ListMeta: metav1.ListMeta{ResourceVersion: fmt.Sprintf("%d", 9000+n)},
		Items:    []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", ResourceVersion: "500"}}},
	}, nil
}

func (c *unchangedPodClient) Watch(_ context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	c.mu.Lock()
	c.watchRVs = append(c.watchRVs, opts.ResourceVersion)
	c.mu.Unlock()
	ch := make(chan watch.Event, 1)
	ch <- watch.Event{Type: watch.Error, Object: &apierrors.NewResourceExpired("too old resource version").ErrStatus}
	return &closedPodWatcher{ch: ch}, nil
}

func TestPodWatcher_RecoveryDoesNotReuseTheRejectedVersion(t *testing.T) {
	client := &unchangedPodClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	NewPodWatcher(ctx, client, metav1.ListOptions{ResourceVersion: "500"}, func(*corev1.Pod) {})

	time.Sleep(1400 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)

	client.mu.Lock()
	rvs := append([]string(nil), client.watchRVs...)
	client.mu.Unlock()

	t.Logf("resourceVersions sent on watch: %v", rvs)
	require.GreaterOrEqual(t, len(rvs), 2, "expected a retry after the rejection")

	// The object still reports 500, the version the server rejected. Every watch after the
	// first must use the version the list reported instead.
	for i, rv := range rvs[1:] {
		assert.NotEqual(t, "500", rv,
			"watch %d returned to the rejected resourceVersion instead of the list version", i+2)
	}
}

// An API server coming back from a restart can fail the recovery list for a moment. That must
// not end the watcher, or the execution stops being followed for the rest of its run.
type flakyListPodClient struct {
	calls int32
}

func (c *flakyListPodClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.PodList, error) {
	// First call succeeds so the watcher starts; the next fails as the server restarts.
	if n := atomic.AddInt32(&c.calls, 1); n == 2 {
		return nil, apierrors.NewServiceUnavailable("the server is currently unable to handle the request")
	}
	return &corev1.PodList{
		ListMeta: metav1.ListMeta{ResourceVersion: "1000"},
		Items:    []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", ResourceVersion: "1000"}}},
	}, nil
}

func (c *flakyListPodClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	ch := make(chan watch.Event)
	close(ch)
	return &closedPodWatcher{ch: ch}, nil
}

func TestPodWatcher_SurvivesATemporaryListFailure(t *testing.T) {
	client := &flakyListPodClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := NewPodWatcher(ctx, client, metav1.ListOptions{ResourceVersion: "500"}, func(*corev1.Pod) {})

	// One second to the failing list, two more to the one that succeeds after it.
	time.Sleep(4 * time.Second)

	select {
	case <-w.Done():
		t.Fatalf("the watcher ended on a temporary list failure: %v", w.Err())
	default:
	}

	assert.GreaterOrEqual(t, int(atomic.LoadInt32(&client.calls)), 3,
		"the watcher stopped listing after the failure instead of recovering")
}
