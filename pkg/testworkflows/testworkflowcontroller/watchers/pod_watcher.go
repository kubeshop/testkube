package watchers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

type podWatcher struct {
	client    kubernetesClient[corev1.PodList, *corev1.Pod]
	opts      metav1.ListOptions
	peek      *corev1.Pod
	hook      func(*corev1.Pod)
	started   atomic.Bool
	startedCh chan struct{} // TODO: Ensure there is no memory leak
	ch        chan *corev1.Pod
	peekable  atomic.Bool
	peekCh    chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	err       error
	mu        sync.Mutex
	peekMu    sync.Mutex
}

type PodWatcher interface {
	Channel() <-chan *corev1.Pod
	Peek(ctx context.Context) <-chan *corev1.Pod
	Update(t time.Duration) (int, error)
	Exists() bool
	IsStarted() bool
	Started() <-chan struct{}
	Stop()
	Done() <-chan struct{}
	Err() error
}

func NewPodWatcher(parentCtx context.Context, client kubernetesClient[corev1.PodList, *corev1.Pod], opts metav1.ListOptions, bufferSize int, hook func(*corev1.Pod)) PodWatcher {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	opts.AllowWatchBookmarks = true
	watcher := &podWatcher{
		client:    client,
		opts:      opts,
		hook:      hook,
		ch:        make(chan *corev1.Pod, bufferSize),
		startedCh: make(chan struct{}),
		peekCh:    make(chan struct{}),
		ctx:       ctx,
		cancel:    ctxCancel,
	}
	go watcher.cycle()
	return watcher
}

func (e *podWatcher) IsStarted() bool {
	return e.started.Load()
}

func (e *podWatcher) Started() <-chan struct{} {
	ch := make(chan struct{})
	if e.started.Load() || e.ctx.Err() != nil || e.startedCh == nil {
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
	if pod != nil {
		e.peek = pod
	}
	if e.peekable.CompareAndSwap(false, true) {
		close(e.peekCh)
	}
}

func (e *podWatcher) read(t time.Duration) (<-chan readStart, <-chan struct{}) {
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

		// Send the item immediately to the hook aside of all the other processing
		if len(list.Items) == 1 {
			e.hook(common.Ptr(list.Items[0]))
		}

		// Ignore error when the channel is already closed
		defer func() {
			recover()
		}()

		// Handle lack of the pod
		if len(list.Items) == 0 {
			e.peekMu.Lock()
			pod := e.peek
			e.peekMu.Unlock()

			// Mark as initial list is starting to propagate
			if e.started.CompareAndSwap(false, true) {
				close(e.startedCh)
			}

			if pod == nil {
				// there is no pod, but it's not a change.
				started <- readStart{count: 0}
				close(started)
				return
			} else {
				// the pod was there, but it's deleted now.
				e.finalize(nil)

				// Inform about start
				started <- readStart{count: 1, err: ErrDone}
				close(started)
				return
			}
		}

		// Store information about the last pod for peeking
		e.setLastPod(common.Ptr(list.Items[0]))

		// Mark as initial list is starting to propagate
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}

		// There is no update
		if list.Items[0].ResourceVersion == e.opts.ResourceVersion {
			started <- readStart{count: 0}
			close(started)
			return
		}

		// Inform about start
		started <- readStart{count: len(list.Items)}
		close(started)

		// Send the item
		e.ch <- common.Ptr(list.Items[0])
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

			// Send the item immediately to the hook aside of all the other processing
			e.hook(object)

			// Try to configure deletion timestamp if Kubernetes engine doesn't support it
			if event.Type == watch.Deleted && object.DeletionTimestamp == nil {
				ts := object.CreationTimestamp.Time
				if object.Status.StartTime != nil && object.Status.StartTime.After(ts) {
					ts = object.Status.StartTime.Time
				}
				for i := range object.Status.Conditions {
					if object.Status.Conditions[i].LastTransitionTime.After(ts) {
						ts = object.Status.Conditions[i].LastTransitionTime.Time
					}
				}
				for i := range object.Status.InitContainerStatuses {
					if object.Status.InitContainerStatuses[i].State.Terminated != nil && object.Status.InitContainerStatuses[i].State.Terminated.FinishedAt.After(ts) {
						ts = object.Status.InitContainerStatuses[i].State.Terminated.FinishedAt.Time
					}
					if object.Status.InitContainerStatuses[i].LastTerminationState.Terminated != nil && object.Status.InitContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.After(ts) {
						ts = object.Status.InitContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.Time
					}
				}
				for i := range object.Status.ContainerStatuses {
					if object.Status.ContainerStatuses[i].State.Terminated != nil && object.Status.ContainerStatuses[i].State.Terminated.FinishedAt.After(ts) {
						ts = object.Status.ContainerStatuses[i].State.Terminated.FinishedAt.Time
					}
					if object.Status.ContainerStatuses[i].LastTerminationState.Terminated != nil && object.Status.ContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.After(ts) {
						ts = object.Status.ContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.Time
					}
				}
				for i := range object.ManagedFields {
					if object.ManagedFields[i].Time != nil && object.ManagedFields[i].Time.After(ts) {
						ts = object.ManagedFields[i].Time.Time
					}
				}
				object.DeletionTimestamp = &metav1.Time{ts}
			}

			// Send the event back
			e.setLastPod(object)
			e.ch <- object

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

		if e.peekable.CompareAndSwap(false, true) {
			close(e.peekCh)
		}

		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}
	}()

	// Read the initial data
	started, finished := e.read(0)
	result, _ := <-started
	if result.err != nil {
		e.setError(result.err)
		return
	}
	<-finished

	// Watch for the data updates,
	// and restart the watcher as long as there are no errors
	var err error
	for err == nil {
		err = e.watch()
	}
	e.setError(err)
	e.cancel()
}

func (e *podWatcher) Exists() bool {
	e.peekMu.Lock()
	defer e.peekMu.Unlock()
	return e.peek != nil
}

func (e *podWatcher) Peek(ctx context.Context) <-chan *corev1.Pod {
	if e.peekable.Load() {
		ch := make(chan *corev1.Pod, 1)

		e.peekMu.Lock()
		pod := e.peek
		e.peekMu.Unlock()

		ch <- pod
		close(ch)
		return ch
	} else if e.ctx.Err() != nil {
		ch := make(chan *corev1.Pod)
		close(ch)
		return ch
	}

	ch := make(chan *corev1.Pod)
	go func() {
		select {
		case <-e.peekCh:
		case <-ctx.Done():
			close(ch)
			return
		}
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

func (e *podWatcher) Done() <-chan struct{} {
	return e.ctx.Done()
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
	fmt.Println(ui.Red("REFRESHING Pod"), e.opts)

	started, _ := e.read(t)
	result, _ := <-started
	if errors.Is(result.err, ErrDone) {
		result.err = nil
	}
	return result.count, result.err
}
