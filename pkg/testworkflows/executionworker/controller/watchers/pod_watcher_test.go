package watchers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
