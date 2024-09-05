package watchers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
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
		list, err := e.client.List(ctx, e.opts)
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

		// There is no update
		if list.Items[0].ResourceVersion == e.opts.ResourceVersion {
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

// TODO: handle resource too old
func (e *podWatcher) watch() error {
	// Initialize the watcher
	opts := e.opts
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultWatchTimeoutSeconds)
	}
	watcher, err := e.client.Watch(e.ctx, opts)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	// Ignore error when the channel is already closed
	e.watching.Store(true)
	defer func() {
		recover()
		e.watching.Store(false)
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
				return ErrDone
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
	result, _ := <-started
	if result.err != nil {
		e.cancel(result.err)
		return
	}
	<-finished

	// Watch for the data updates,
	// and restart the watcher as long as there are no errors
	var err error
	for err == nil {
		err = e.watch()
	}
	e.cancel(err)
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
	result, _ := <-started
	if errors.Is(result.err, ErrDone) {
		result.err = nil
	}
	return result.count, result.err
}
