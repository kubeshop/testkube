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
	listener  func(*corev1.Pod)
	started   atomic.Bool
	startedCh chan struct{} // TODO: Ensure there is no memory leak
	ch        chan *corev1.Pod
	ctx       context.Context
	cancel    context.CancelCauseFunc
	mu        sync.Mutex
	existed   atomic.Bool
}

type PodWatcher interface {
	Channel() <-chan *corev1.Pod
	Update(t time.Duration) (int, error)
	Started() <-chan struct{}
	Done() <-chan struct{}
	Err() error
}

func NewPodWatcher(parentCtx context.Context, client kubernetesClient[corev1.PodList, *corev1.Pod], opts metav1.ListOptions, bufferSize int, listener func(*corev1.Pod)) PodWatcher {
	ctx, ctxCancel := context.WithCancelCause(parentCtx)
	opts.AllowWatchBookmarks = true
	watcher := &podWatcher{
		client:    client,
		opts:      opts,
		listener:  listener,
		ch:        make(chan *corev1.Pod, bufferSize),
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

		// Send the item immediately to the listener aside of all the other processing
		if len(list.Items) == 1 {
			e.listener(common.Ptr(list.Items[0]))
		}

		// Ignore error when the channel is already closed
		defer func() {
			recover()
		}()

		// Handle lack of the pod
		if len(list.Items) == 0 {

			// Mark as initial list is starting to propagate
			if e.started.CompareAndSwap(false, true) {
				close(e.startedCh)
			}

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
			return
		}

		// Mark as existing
		e.existed.Store(true)

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

			// Send the item immediately to the listener aside of all the other processing
			e.listener(object)

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

			// Mark as existing
			e.existed.Store(true)

			// Send the event back
			e.ch <- object

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
		close(e.ch)
		if e.started.CompareAndSwap(false, true) {
			close(e.startedCh)
		}
	}()

	// Read the initial data
	started, finished := e.read(0)
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

// Channel returns the channel for reading the pod.
func (e *podWatcher) Channel() <-chan *corev1.Pod {
	return e.ch
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
