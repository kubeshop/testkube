package watchers

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
)

type eventsWatcher struct {
	client    kubernetesClient[corev1.EventList, *corev1.Event]
	opts      metav1.ListOptions
	optsCh    chan struct{}
	started   atomic.Bool
	watching  atomic.Bool
	startedCh chan struct{} // TODO: Ensure there is no memory leak
	listener  func(*corev1.Event)
	ctx       context.Context
	cancel    context.CancelCauseFunc
	mu        sync.Mutex
	lastTs    time.Time
}

type EventsWatcher interface {
	LastAcknowledgedTime() time.Time
	Update(t time.Duration) (int, error)
	Ensure(tsInPast time.Time, timeout time.Duration) (int, error)
	Started() <-chan struct{}
	Done() <-chan struct{}
	Err() error
}

func NewEventsWatcher(parentCtx context.Context, client kubernetesClient[corev1.EventList, *corev1.Event], opts metav1.ListOptions, listener func(event *corev1.Event)) EventsWatcher {
	ctx, ctxCancel := context.WithCancelCause(parentCtx)
	watcher := &eventsWatcher{
		client:    client,
		opts:      opts,
		listener:  listener,
		optsCh:    make(chan struct{}),
		startedCh: make(chan struct{}),
		ctx:       ctx,
		cancel:    ctxCancel,
	}
	close(watcher.optsCh)
	go watcher.cycle()
	return watcher
}

func NewAsyncEventsWatcher(parentCtx context.Context, client kubernetesClient[corev1.EventList, *corev1.Event], opts <-chan metav1.ListOptions, listener func(event *corev1.Event)) EventsWatcher {
	ctx, ctxCancel := context.WithCancelCause(parentCtx)
	watcher := &eventsWatcher{
		client:    client,
		listener:  listener,
		optsCh:    make(chan struct{}),
		startedCh: make(chan struct{}),
		ctx:       ctx,
		cancel:    ctxCancel,
	}
	go watcher.waitForOpts(opts)
	go watcher.cycle()
	return watcher
}

func (e *eventsWatcher) LastAcknowledgedTime() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastTs
}

func (e *eventsWatcher) Started() <-chan struct{} {
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

func (e *eventsWatcher) waitForOpts(opts <-chan metav1.ListOptions) {
	select {
	case v, _ := <-opts:
		e.mu.Lock()
		e.opts = v
		e.mu.Unlock()
	case <-e.ctx.Done():
	}
	close(e.optsCh)
}

func (e *eventsWatcher) read(tsInPast time.Time, t time.Duration) (<-chan readStart, <-chan struct{}) {
	started := make(chan readStart, 1)
	finished := make(chan struct{})

	go func() {
		e.mu.Lock()
		defer func() {
			close(finished)
			e.mu.Unlock()
		}()

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
			started <- readStart{err: err}
			close(started)
			return
		}

		// Update the latest resource version
		e.opts.ResourceVersion = list.ResourceVersion

		// Omit the events that have been already sent
		for i := range list.Items {
			if list.Items[i].ResourceVersion == e.opts.ResourceVersion {
				list.Items = list.Items[i+1:]
				break
			}
		}

		if len(list.Items) == 0 {
			if e.started.CompareAndSwap(false, true) {
				close(e.startedCh)
			}
			started <- readStart{count: 0}
			close(started)
			return
		}

		// Update the last acknowledged timestamp
		if tsInPast.After(e.lastTs) {
			e.lastTs = tsInPast
		}
		for i := range list.Items {
			if GetEventTimestamp(&list.Items[i]).After(e.lastTs) {
				e.lastTs = GetEventTimestamp(&list.Items[i])
			}
		}

		// Inform about start
		started <- readStart{count: len(list.Items)}
		close(started)

		// Send the received events
		for i := range list.Items {
			e.listener(common.Ptr(list.Items[i]))
		}

		// Mark as initial list is starting to propagate
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}
	}()

	return started, finished
}

// TODO: handle resource too old
func (e *eventsWatcher) watch() error {
	// Initialize the watcher
	opts := e.opts
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultWatchTimeoutSeconds)
	}
	opts.AllowWatchBookmarks = true
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
			object, ok := event.Object.(*corev1.Event)
			if !ok || object == nil {
				continue
			}

			// Save the latest resource version to recover
			e.mu.Lock()
			e.opts.ResourceVersion = object.ResourceVersion
			if object.CreationTimestamp.Time.After(e.lastTs) {
				e.lastTs = object.CreationTimestamp.Time
			}
			if object.LastTimestamp.Time.After(e.lastTs) {
				e.lastTs = object.LastTimestamp.Time
			}
			e.mu.Unlock()

			// Continue watching if that's just a bookmark
			if event.Type == watch.Bookmark {
				continue
			}

			// Send the item immediately to the listener aside of all the other processing
			e.listener(object)
		}
	}
}

func (e *eventsWatcher) cycle() {
	// Close the channel when the watcher is stopped
	go func() {
		<-e.ctx.Done()
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}
	}()

	// Wait for readiness
	<-e.optsCh

	// Read the initial data
	started, finished := e.read(time.Time{}, 0)
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

func (e *eventsWatcher) Err() error {
	return e.ctx.Err()
}

func (e *eventsWatcher) Done() <-chan struct{} {
	return e.ctx.Done()
}

// Update gets the latest list of the events, to ensure that nothing is missed at that point.
// It returns number of items that have been appended.
func (e *eventsWatcher) Update(t time.Duration) (int, error) {
	// Wait for readiness
	<-e.optsCh

	// Start reading data
	started, _ := e.read(time.Time{}, t)
	result, _ := <-started
	return result.count, result.err
}

// Ensure checks if there are already acknowledged events for particular timestamp
func (e *eventsWatcher) Ensure(tsInPast time.Time, timeout time.Duration) (int, error) {
	// Wait for readiness
	<-e.optsCh

	// Fast-track when the timestamp is already acknowledged
	e.mu.Lock()
	if tsInPast.Before(e.lastTs) {
		e.mu.Unlock()
		return 0, nil
	}
	e.mu.Unlock()

	// Start reading data
	started, _ := e.read(tsInPast.Truncate(time.Second).Add(-1), timeout)
	result, _ := <-started
	return result.count, result.err
}
