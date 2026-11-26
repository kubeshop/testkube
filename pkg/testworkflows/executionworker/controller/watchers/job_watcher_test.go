package watchers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// A watch the API server has hung up on, which is what a client sees once its
// resourceVersion has been aged out. Every Watch returns an already-closed channel.
type closingJobClient struct {
	mu         sync.Mutex
	watchCalls int32
	listCalls  int32
	watchRVs   []string
	listRVs    []string
}

func (c *closingJobClient) List(_ context.Context, opts metav1.ListOptions) (*batchv1.JobList, error) {
	atomic.AddInt32(&c.listCalls, 1)
	c.mu.Lock()
	c.listRVs = append(c.listRVs, opts.ResourceVersion)
	c.mu.Unlock()
	return &batchv1.JobList{
		ListMeta: metav1.ListMeta{ResourceVersion: "1000"},
		Items: []batchv1.Job{{
			ObjectMeta: metav1.ObjectMeta{Name: "job-1", ResourceVersion: "1000"},
		}},
	}, nil
}

func (c *closingJobClient) Watch(_ context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	atomic.AddInt32(&c.watchCalls, 1)
	c.mu.Lock()
	c.watchRVs = append(c.watchRVs, opts.ResourceVersion)
	c.mu.Unlock()
	ch := make(chan watch.Event)
	close(ch) // the server hung up, exactly as it does on "too old resourceVersion"
	return &closedWatcher{ch: ch}, nil
}

type closedWatcher struct{ ch chan watch.Event }

func (w *closedWatcher) Stop()                          {}
func (w *closedWatcher) ResultChan() <-chan watch.Event { return w.ch }

func TestJobWatcher_RecoversFromRejectedResourceVersion(t *testing.T) {
	client := &closingJobClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	NewJobWatcher(ctx, client, metav1.ListOptions{ResourceVersion: "500"}, func(*batchv1.Job) {})

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
// watch.Error carrying a Status. The object fails the *batchv1.Job assertion, so before this was
// handled the rejection was skipped and the same version was watched again.
type expiredRVJobClient struct {
	mu       sync.Mutex
	listRVs  []string
	watchRVs []string
}

func (c *expiredRVJobClient) List(_ context.Context, opts metav1.ListOptions) (*batchv1.JobList, error) {
	c.mu.Lock()
	c.listRVs = append(c.listRVs, opts.ResourceVersion)
	c.mu.Unlock()
	return &batchv1.JobList{
		ListMeta: metav1.ListMeta{ResourceVersion: "1000"},
		Items:    []batchv1.Job{{ObjectMeta: metav1.ObjectMeta{Name: "job-1", ResourceVersion: "1000"}}},
	}, nil
}

func (c *expiredRVJobClient) Watch(_ context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	c.mu.Lock()
	c.watchRVs = append(c.watchRVs, opts.ResourceVersion)
	c.mu.Unlock()
	ch := make(chan watch.Event, 1)
	ch <- watch.Event{
		Type:   watch.Error,
		Object: &apierrors.NewResourceExpired("too old resource version").ErrStatus,
	}
	return &closedWatcher{ch: ch}, nil
}

func TestJobWatcher_TreatsExpiredResourceVersionAsAFailure(t *testing.T) {
	client := &expiredRVJobClient{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	NewJobWatcher(ctx, client, metav1.ListOptions{ResourceVersion: "500"}, func(*batchv1.Job) {})

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
