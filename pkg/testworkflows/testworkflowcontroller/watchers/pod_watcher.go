package watchers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
)

type podWatcher struct {
	client kubernetesClient[corev1.PodList, *corev1.Pod]
	opts   metav1.ListOptions
	peek   *corev1.Pod
	ch     chan *corev1.Pod
	peekCh chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	mu     sync.Mutex
	peekMu sync.Mutex
}

type PodWatcher interface {
	Channel() <-chan *corev1.Pod
	Peek() <-chan *corev1.Pod
	Update(t time.Duration) (int, error)
	Stop()
	Err() error
}

func NewPodWatcher(ctx context.Context, client kubernetesClient[corev1.PodList, *corev1.Pod], opts metav1.ListOptions, bufferSize int) PodWatcher {
	childCtx, ctxCancel := context.WithCancel(ctx)
	opts.AllowWatchBookmarks = true
	watcher := &podWatcher{
		client: client,
		opts:   opts,
		ch:     make(chan *corev1.Pod, bufferSize),
		ctx:    childCtx,
		cancel: ctxCancel,
	}
	go watcher.cycle()
	return watcher
}

func (e *podWatcher) setError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.err = err
	e.cancel()
}

func (e *podWatcher) finalize(pod *corev1.Pod) bool {
	if IsPodFinished(pod) {
		e.err = ErrDone
		e.cancel()
		return true
	}
	return false
}

func (e *podWatcher) setLastPod(pod *corev1.Pod) {
	e.peekMu.Lock()
	defer e.peekMu.Unlock()
	e.peek = pod
	peekCh := e.peekCh
	e.peekCh = nil
	if peekCh == nil {
		close(peekCh)
	}
}

func (e *podWatcher) read(t time.Duration) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Fetch the data
	opts := e.opts
	opts.ResourceVersion = ""
	if t != 0 {
		opts.TimeoutSeconds = common.Ptr(int64(math.Ceil(t.Seconds())))
	}
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultListTimeoutSeconds)
	}
	list, err := e.client.List(e.ctx, e.opts)
	if err != nil {
		return 0, err
	}

	// Update the latest resource version
	e.opts.ResourceVersion = list.ResourceVersion

	// Ignore error when the channel is already closed
	defer func() {
		recover()
	}()

	// Disallow watching multiple pods in that watcher
	if len(list.Items) > 1 {
		names := make([]string, len(list.Items))
		for i := range list.Items {
			names[i] = list.Items[i].Name
		}
		return 0, fmt.Errorf("found more than one pod for selected criteria: %s", strings.Join(names, ", "))
	}

	// Handle lack of the pod
	if len(list.Items) == 0 {
		e.peekMu.Lock()
		pod := e.peek
		e.peekMu.Unlock()
		if pod == nil {
			// there is no pod, but it's not a change.
			return 0, nil
		} else {
			// the pod was there, but it's deleted now.
			e.finalize(nil)
			return 1, ErrDone
		}
	}

	// There is no update
	if list.Items[0].ResourceVersion == e.opts.ResourceVersion {
		return 0, nil
	}

	// The pod has been updated
	e.ch <- common.Ptr(list.Items[0])
	e.setLastPod(common.Ptr(list.Items[0]))

	return 1, nil
}

func (e *podWatcher) watch() error {
	// Initialize the watcher
	opts := e.opts
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultWatchTimeoutSeconds)
	}
	watcher, err := e.client.Watch(e.ctx, opts)
	defer watcher.Stop()
	if err != nil {
		return err
	}

	// Ignore error when the channel is already closed
	defer func() {
		recover()
	}()

	// Read the items
	ch := watcher.ResultChan()
	for {
		// Prioritize checking for finished watcher
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		default:
		}

		// Wait for the results
		select {
		case <-e.ctx.Done():
			return e.ctx.Err()
		case event, ok := <-ch:
			// Handle closed watcher
			if !ok {
				return e.ctx.Err()
			}

			// Load the current Kubernetes object
			object, ok := event.Object.(*corev1.Pod)
			if !ok || object == nil {
				continue
			}

			// Save the latest resource version to recover
			e.mu.Lock()
			e.opts.ResourceVersion = object.ResourceVersion
			e.mu.Unlock()

			// Continue watching if that's just a bookmark
			if event.Type == watch.Bookmark {
				continue
			}

			// Send the event back
			e.ch <- object
			e.setLastPod(object)

			// Handle the deletion
			e.mu.Lock()
			if e.finalize(object) {
				e.mu.Unlock()
				return ErrDone
			}
			e.mu.Unlock()
		}
	}
}

func (e *podWatcher) cycle() {
	// Close the channel when the watcher is stopped
	go func() {
		<-e.ctx.Done()
		close(e.ch)

		e.peekMu.Lock()
		defer e.peekMu.Unlock()
		peekCh := e.peekCh
		e.peekCh = nil
		if peekCh != nil {
			close(e.peekCh)
		}
	}()

	// Read the initial data
	_, err := e.read(0)
	if err != nil {
		e.setError(err)
		return
	}

	// Watch for the data updates,
	// and restart the watcher as long as there are no errors
	for err == nil {
		err = e.watch()
	}
	e.setError(err)
	e.cancel()
}

func (e *podWatcher) Peek() <-chan *corev1.Pod {
	ch := make(chan *corev1.Pod)

	go func() {
		<-e.peekCh
		e.peekMu.Lock()
		pod := e.peek
		e.peekMu.Unlock()
		if pod != nil {
			ch <- pod
		}
		close(ch)
	}()

	return ch
}

func (e *podWatcher) Err() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err != nil {
		return e.err
	}
	return e.ctx.Err()
}

// Channel returns the channel for reading the pod.
func (e *podWatcher) Channel() <-chan *corev1.Pod {
	return e.ch
}

// Stop cancels all the on-going communication
func (e *podWatcher) Stop() {
	e.cancel()
}

// Update gets the latest list of the pod, to ensure that nothing is missed at that point.
// It returns number of items that have been appended.
func (e *podWatcher) Update(t time.Duration) (int, error) {
	count, err := e.read(t)
	if errors.Is(err, ErrDone) {
		err = nil
	}
	return count, err
}
