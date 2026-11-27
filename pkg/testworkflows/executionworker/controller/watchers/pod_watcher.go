package watchers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/log"
)

const (
	podRetryDelay    = time.Second
	podRetryMaxDelay = 30 * time.Second
)

type podWatcher struct {
	client    kubernetesClient[corev1.PodList, *corev1.Pod]
	opts      metav1.ListOptions
	listener  func(*corev1.Pod)
	started   atomic.Bool
	watching  atomic.Bool
	startedCh chan struct{}
	ctx       context.Context
	cancel    context.CancelCauseFunc
	mu        sync.Mutex
	existed   atomic.Bool
}

type PodWatcher interface {
	Update(ctx context.Context) (int, error)
	Started() <-chan struct{}
	Done() <-chan struct{}
	Err() error
}

func NewPodWatcher(parentCtx context.Context, client kubernetesClient[corev1.PodList, *corev1.Pod], opts metav1.ListOptions, listener func(*corev1.Pod)) PodWatcher {
	ctx, ctxCancel := context.WithCancelCause(parentCtx)
	opts.AllowWatchBookmarks = true
	watcher := &podWatcher{
		client:    client,
		opts:      opts,
		listener:  listener,
		startedCh: make(chan struct{}),
		ctx:       ctx,
		cancel:    ctxCancel,
	}
	go watcher.cycle()
	return watcher
}

func (e *podWatcher) Started() <-chan struct{} {
	ch := make(chan struct{})
	if e.started.Load() || e.ctx.Err() != nil {
		close(ch)
	} else {
		go func() {
			select {
			case <-e.ctx.Done():
			case <-e.startedCh:
			}
			close(ch)
		}()
	}
	return ch
}

// TODO: add readMu lock, work better with mu lock
func (e *podWatcher) read(sideCtx context.Context) (<-chan readStart, <-chan struct{}) {
	started := make(chan readStart, 1)
	finished := make(chan struct{})

	ctx, ctxCancel := context.WithCancelCause(sideCtx)
	go func() {
		select {
		case <-e.ctx.Done():
			ctxCancel(e.ctx.Err())
		case <-ctx.Done():
			ctxCancel(ctx.Err())
		}
	}()

	go func() {
		e.mu.Lock()
		defer func() {
			close(finished)
			ctxCancel(context.Canceled)
			e.mu.Unlock()
		}()

		// Fetch the data
		opts := e.opts
		opts.ResourceVersion = ""
		if opts.TimeoutSeconds == nil {
			opts.TimeoutSeconds = common.Ptr(defaultListTimeoutSeconds)
		}
		list, err := e.client.List(ctx, opts)
		if err != nil {
			started <- readStart{err: err}
			close(started)
			return
		}

		// Disallow watching multiple pods in that watcher
		if len(list.Items) > 1 {
			names := make([]string, len(list.Items))
			for i := range list.Items {
				names[i] = list.Items[i].Name
			}
			started <- readStart{err: fmt.Errorf("found more than one pod for selected criteria: %s", strings.Join(names, ", "))}
			close(started)
			return
		}

		// Handle lack of the pod
		if len(list.Items) == 0 {
			// Update the latest resource version
			e.opts.ResourceVersion = list.ResourceVersion

			if e.existed.Load() {
				// the pod was there, but it's deleted now.
				e.cancel(ErrDone)

				// Inform about start
				started <- readStart{count: 1, err: ErrDone}
				close(started)
			} else {
				// there is no pod, but it's not a change.
				started <- readStart{count: 0}
				close(started)
			}

			// Mark as initial list is starting to propagate
			if e.started.CompareAndSwap(false, true) {
				close(e.startedCh)
			}

			return
		}

		// There is no update. The list version still has to be saved: an object that has not
		// changed carries the version the server just rejected, so keeping it would send the
		// next watch back with the same rejected version.
		if list.Items[0].ResourceVersion == e.opts.ResourceVersion {
			e.opts.ResourceVersion = list.ResourceVersion
			started <- readStart{count: 0}
			close(started)
			return
		}

		// Mark as existing
		e.existed.Store(true)

		// Update the latest resource version
		e.opts.ResourceVersion = list.ResourceVersion

		// Inform about start
		started <- readStart{count: len(list.Items)}
		close(started)

		// Send the item
		e.listener(common.Ptr(list.Items[0]))

		// Mark as initial list is starting to propagate
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}
	}()

	return started, finished
}

// watch reads a single watch stream. It reports whether any event was delivered, so the caller
// can reset its backoff after a watch that worked, and it returns a real error when the stream
// ends. A resourceVersion the API server has aged out arrives here as a watch.Error.
func (e *podWatcher) watch() (bool, error) {
	// Initialize the watcher
	opts := e.opts
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultWatchTimeoutSeconds)
	}
	watcher, err := e.client.Watch(e.ctx, opts)
	if err != nil {
		return false, err
	}
	defer watcher.Stop()

	// Ignore error when the channel is already closed
	e.watching.Store(true)
	defer func() {
		recover()
		e.watching.Store(false)
	}()

	// Read the items
	delivered := false
	ch := watcher.ResultChan()
	for {
		// Prioritize checking for finished watcher
		select {
		case <-e.ctx.Done():
			return delivered, e.ctx.Err()
		default:
		}

		// Wait for the results
		select {
		case <-e.ctx.Done():
			return delivered, e.ctx.Err()
		case event, ok := <-ch:
			// The watch ended without the context being done, so the API server or a proxy closed
			// it. Report it so the caller lists again instead of reconnecting on the same version.
			if !ok {
				return delivered, errWatchClosed
			}

			// The API reports an error in the stream, for example a version that is too old. The
			// watch cannot continue from that version, so the cycle lists the pod again.
			if event.Type == watch.Error {
				return delivered, apierrors.FromObject(event.Object)
			}
			delivered = true

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

			// Try to configure deletion timestamp if Kubernetes engine doesn't support it
			if event.Type == watch.Deleted && object.DeletionTimestamp == nil {
				object.DeletionTimestamp = &metav1.Time{Time: GetPodLastTimestamp(object)}
			}

			// Mark as existing
			e.existed.Store(true)

			// Send the event back
			e.listener(object)

			// Handle the deletion
			if IsPodFinished(object) {
				e.cancel(ErrDone)
				return delivered, ErrDone
			}
		}
	}
}

func (e *podWatcher) cycle() {
	// Close the channel when the watcher is stopped
	go func() {
		<-e.ctx.Done()
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}
	}()

	// Read the initial data
	started, finished := e.read(context.Background())
	result := <-started
	if result.err != nil {
		e.cancel(result.err)
		return
	}
	<-finished

	// Watch for the data updates. A watch can end for a reason a retry recovers from, most often a
	// resourceVersion the API server has aged out after a restart. Recovery re-enters through the
	// list, so every attempt carries a fresh version, and the delay keeps a persistent failure from
	// saturating the API server. A watch that delivered something resets the delay.
	retry := backoff{base: podRetryDelay, max: podRetryMaxDelay}
	var err error
	for e.ctx.Err() == nil {
		var watched bool
		watched, err = e.watch()
		if e.ctx.Err() != nil || errors.Is(err, ErrDone) {
			break
		}

		delay := retry.delay(watched)
		log.DefaultLogger.Warnw("watching the pod failed, listing it again",
			"labelSelector", e.currentOpts().LabelSelector, "retryIn", delay.String(), "error", err)
		select {
		case <-e.ctx.Done():
		case <-time.After(delay):
		}
		if e.ctx.Err() != nil {
			break
		}

		// List again so the next watch starts from a version the API server still knows. A
		// temporary failure here, such as the API server still coming back, must not end the
		// watcher: the loop watches again, fails fast, and waits longer before the next list.
		started, finished = e.read(context.Background())
		result = <-started
		if errors.Is(result.err, ErrDone) {
			err = result.err
			break
		}
		if result.err != nil {
			err = result.err
			continue
		}
		<-finished
	}
	if err == nil {
		err = context.Cause(e.ctx)
	}
	e.cancel(err)
}

// currentOpts returns a copy of the options under the lock, because the watch writes the
// resourceVersion into them while the cycle reads them.
func (e *podWatcher) currentOpts() metav1.ListOptions {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.opts
}

func (e *podWatcher) Err() error {
	return e.ctx.Err()
}

func (e *podWatcher) Done() <-chan struct{} {
	return e.ctx.Done()
}

// Update gets the latest list of the pod, to ensure that nothing is missed at that point.
// It returns number of items that have been appended.
func (e *podWatcher) Update(ctx context.Context) (int, error) {
	started, _ := e.read(ctx)
	result := <-started
	if errors.Is(result.err, ErrDone) {
		result.err = nil
	}
	return result.count, result.err
}
