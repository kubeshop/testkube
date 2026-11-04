package watchers

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudflare/backoff"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/log"
)

// The delay before the watcher lists the events again after an error. It doubles after each
// error in a row, up to the maximum, because the Kubernetes API is often busy when it fails.
var (
	eventsRetryDelay    = time.Second
	eventsRetryMaxDelay = 30 * time.Second
)

type eventsWatcher struct {
	client    kubernetesClient[corev1.EventList, *corev1.Event]
	opts      metav1.ListOptions
	optsCh    chan struct{}
	started   atomic.Bool
	watching  atomic.Bool
	startedCh chan struct{} // TODO: Ensure there is no memory leak
	listener  func(*corev1.Event)
	// retryUntil closes when no other watcher observes the execution. After that, an error ends
	// the watcher instead of a retry, and Err returns that error.
	retryUntil <-chan struct{}
	// parentCtx outlives the watcher. A read with a limit uses it, so the last list of an execution
	// still works after an error ended the watcher.
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelCauseFunc
	mu        sync.Mutex
	lastTs    time.Time
	// sent holds each version of an event that the listener received. A list can return a version
	// that the watch already sent, and the version of an event is the only way to know it. The set
	// lives as long as the watcher, which watches the events of one execution.
	sent map[string]struct{}
}

type EventsWatcher interface {
	LastAcknowledgedTime() time.Time
	Update(t time.Duration) (int, error)
	Ensure(tsInPast time.Time, timeout time.Duration) (int, error)
	Started() <-chan struct{}
	Done() <-chan struct{}
	Err() error
}

func NewEventsWatcher(parentCtx context.Context, client kubernetesClient[corev1.EventList, *corev1.Event], opts metav1.ListOptions, retryUntil <-chan struct{}, listener func(event *corev1.Event)) EventsWatcher {
	ctx, ctxCancel := context.WithCancelCause(parentCtx)
	watcher := &eventsWatcher{
		client:     client,
		opts:       opts,
		listener:   listener,
		retryUntil: retryUntil,
		optsCh:     make(chan struct{}),
		startedCh:  make(chan struct{}),
		parentCtx:  parentCtx,
		ctx:        ctx,
		cancel:     ctxCancel,
		sent:       make(map[string]struct{}),
	}
	close(watcher.optsCh)
	go watcher.cycle()
	return watcher
}

func NewAsyncEventsWatcher(parentCtx context.Context, client kubernetesClient[corev1.EventList, *corev1.Event], opts <-chan metav1.ListOptions, retryUntil <-chan struct{}, listener func(event *corev1.Event)) EventsWatcher {
	ctx, ctxCancel := context.WithCancelCause(parentCtx)
	watcher := &eventsWatcher{
		client:     client,
		listener:   listener,
		retryUntil: retryUntil,
		optsCh:     make(chan struct{}),
		startedCh:  make(chan struct{}),
		parentCtx:  parentCtx,
		ctx:        ctx,
		cancel:     ctxCancel,
		sent:       make(map[string]struct{}),
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
	case v := <-opts:
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
		defer close(finished)

		// Fetch the data without the lock, so a slow list does not hold back another read. A read with
		// a limit also stops its own list at the limit.
		opts := e.currentOpts()
		opts.ResourceVersion = ""
		if t != 0 {
			opts.TimeoutSeconds = common.Ptr(int64(math.Ceil(t.Seconds())))
		}
		if opts.TimeoutSeconds == nil {
			opts.TimeoutSeconds = common.Ptr(defaultListTimeoutSeconds)
		}
		ctx := e.ctx
		if t != 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(e.parentCtx, t)
			defer cancel()
		}
		list, err := e.client.List(ctx, opts)
		if err != nil {
			started <- readStart{err: err}
			close(started)
			return
		}

		e.mu.Lock()
		defer e.mu.Unlock()

		// Update the latest resource version
		e.opts.ResourceVersion = list.ResourceVersion

		// Omit the event versions that the listener already received. The version of the list does not
		// tell them apart: when the newest event is also the newest write, both have the same version.
		items := list.Items[:0]
		for i := range list.Items {
			if e.markSent(&list.Items[i]) {
				items = append(items, list.Items[i])
			}
		}
		list.Items = items

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

// watch reads the events until the watch ends. It reports whether the watch delivered an event,
// which shows that the API accepts the watch.
func (e *eventsWatcher) watch() (bool, error) {
	// Initialize the watcher
	opts := e.currentOpts()
	if opts.TimeoutSeconds == nil {
		opts.TimeoutSeconds = common.Ptr(defaultWatchTimeoutSeconds)
	}
	opts.AllowWatchBookmarks = true
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
	ch := watcher.ResultChan()
	delivered := false
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
			// Handle closed watcher
			if !ok {
				return delivered, e.ctx.Err()
			}

			// The API reports an error in the stream, for example a version that is too old. The
			// watch cannot continue from that version, so the cycle lists the events again.
			if event.Type == watch.Error {
				return delivered, apierrors.FromObject(event.Object)
			}
			delivered = true

			// Load the current Kubernetes object
			object, ok := event.Object.(*corev1.Event)
			if !ok || object == nil {
				continue
			}

			// Save the latest resource version to recover
			e.mu.Lock()
			e.opts.ResourceVersion = object.ResourceVersion
			if object.CreationTimestamp.After(e.lastTs) {
				e.lastTs = object.CreationTimestamp.Time
			}
			if object.LastTimestamp.After(e.lastTs) {
				e.lastTs = object.LastTimestamp.Time
			}
			e.mu.Unlock()

			// Continue watching if that's just a bookmark, or if a list already sent this version
			if event.Type == watch.Bookmark {
				continue
			}
			// The listener runs under the lock, as in a read, so a final list cannot skip an event that
			// is marked as sent but is not in the store yet.
			e.mu.Lock()
			if e.markSent(object) {
				e.listener(object)
			}
			e.mu.Unlock()
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

	// A busy Kubernetes API can fail a list or a watch. The events keep the cause of a waiting pod,
	// so the watcher lists them again after a delay instead of stopping, while other watchers still
	// observe the execution.
	// A watch that delivered an event resets the delay, because a later error does not follow the
	// earlier ones. A list alone does not, because the API can accept every list and refuse every
	// watch. The delay has no jitter, so a retry never comes at once.
	retry := backoff.NewWithoutJitter(eventsRetryMaxDelay, eventsRetryDelay)
	for e.ctx.Err() == nil {
		watched, err := e.listAndWatch()
		if e.ctx.Err() != nil {
			break
		}

		// The caller does not wait for a list that fails, because the events are not required to start.
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}

		if watched {
			retry.Reset()
		}
		delay := retry.Duration()
		select {
		case <-e.retryUntil:
			e.cancel(err)
			return
		default:
		}
		log.DefaultLogger.Warnw("watching the events failed, listing them again", "fieldSelector", e.currentOpts().FieldSelector, "retryIn", delay.String(), "error", err)
		select {
		case <-e.ctx.Done():
		case <-e.retryUntil:
			e.cancel(err)
			return
		case <-time.After(delay):
		}
	}
	e.cancel(context.Cause(e.ctx))
}

// errWatchClosed is the error of a watch that the API closed with no event. The watch has no
// timeout of its own, so a proxy or a busy API server closed it, and a retry waits first.
var errWatchClosed = errors.New("the watch closed with no event")

// listAndWatch lists the events, then watches them until the watch fails. A watch that the API
// closes without an error after it delivered an event starts again from the last version. It
// reports whether a watch delivered an event.
func (e *eventsWatcher) listAndWatch() (bool, error) {
	started, finished := e.read(time.Time{}, 0)
	if result := <-started; result.err != nil {
		return false, result.err
	}
	<-finished
	watched := false
	for {
		delivered, err := e.watch()
		watched = watched || delivered
		if err != nil {
			return watched, err
		}
		if !delivered {
			return watched, errWatchClosed
		}
	}
}

// currentOpts returns a copy of the list options. A read and a watch change the resource version
// while other code reads the options, so the copy takes the lock.
func (e *eventsWatcher) currentOpts() metav1.ListOptions {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.opts
}

// markSent records the version of the event, and reports false when the listener already received it.
// The caller holds the lock.
func (e *eventsWatcher) markSent(event *corev1.Event) bool {
	key := string(event.UID) + "/" + event.ResourceVersion
	if _, ok := e.sent[key]; ok {
		return false
	}
	e.sent[key] = struct{}{}
	return true
}

// Err returns the error that ended the watcher, or nil while it runs.
func (e *eventsWatcher) Err() error {
	if e.ctx.Err() == nil {
		return nil
	}
	return context.Cause(e.ctx)
}

func (e *eventsWatcher) Done() <-chan struct{} {
	return e.ctx.Done()
}

// Update gets the latest list of the events, to ensure that nothing is missed at that point.
// It returns number of items that have been appended, after the listener received them.
func (e *eventsWatcher) Update(t time.Duration) (int, error) {
	// Wait for readiness
	<-e.optsCh

	// Start reading data
	started, finished := e.read(time.Time{}, t)
	result := <-started
	<-finished
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
	result := <-started
	return result.count, result.err
}
