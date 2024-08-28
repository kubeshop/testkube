package watchers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/store"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type executionWatcher struct {
	jobWatcher               JobWatcher
	podWatcher               PodWatcher
	jobEventsWatcher         EventsWatcher
	podEventsWatcher         EventsWatcher
	podEventsOptsCh          chan metav1.ListOptions
	initialCommitCh          chan struct{}
	initialCommitInitialized atomic.Bool
	podEventsInitialized     atomic.Bool

	state       *executionState
	uncommitted *executionState
	update      store.Update
	mu          sync.RWMutex
}

type ExecutionWatcher interface {
	State() ExecutionState
	Commit()

	JobEventsErr() error
	PodEventsErr() error
	JobErr() error
	PodErr() error

	ReadJobEventsAt(ts time.Time, timeout time.Duration)
	ReadPodEventsAt(ts time.Time, timeout time.Duration)
	RefreshPod(timeout time.Duration)
	RefreshJob(timeout time.Duration)

	Started() <-chan struct{}
	Updated() <-chan struct{}
}

func (e *executionWatcher) initializePodEventsWatcher() {
	name := e.uncommitted.PodName()
	if name == "" {
		name = e.state.PodName()
	}
	if name != "" && e.podEventsInitialized.CompareAndSwap(false, true) {
		e.podEventsOptsCh <- metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + name,
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		}
		close(e.podEventsOptsCh)
	}
}

func (e *executionWatcher) State() ExecutionState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

func (e *executionWatcher) Commit() {
	e.mu.Lock()
	defer e.mu.Unlock()
	// FIXME
	uncommited := *e.uncommitted
	uncommitedCopy := uncommited
	e.state = common.Ptr(uncommitedCopy)
	if e.initialCommitInitialized.CompareAndSwap(false, true) {
		close(e.initialCommitCh)
	}
	e.update.Emit()
}

func (e *executionWatcher) JobEventsErr() error {
	return e.jobEventsWatcher.Err()
}

func (e *executionWatcher) PodEventsErr() error {
	return e.podEventsWatcher.Err()
}

func (e *executionWatcher) JobErr() error {
	return e.jobWatcher.Err()
}

func (e *executionWatcher) PodErr() error {
	return e.podWatcher.Err()
}

func (e *executionWatcher) ReadJobEventsAt(ts time.Time, timeout time.Duration) {
	e.jobEventsWatcher.Ensure(ts, timeout)
}

func (e *executionWatcher) ReadPodEventsAt(ts time.Time, timeout time.Duration) {
	e.podEventsWatcher.Ensure(ts, timeout)
}

func (e *executionWatcher) RefreshPod(timeout time.Duration) {
	e.podWatcher.Update(timeout)
}

func (e *executionWatcher) RefreshJob(timeout time.Duration) {
	e.jobWatcher.Update(timeout)
}

func (e *executionWatcher) baseStarted() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-e.jobWatcher.Started()
		<-e.jobEventsWatcher.Started()
		<-e.podWatcher.Started()
		if e.podEventsInitialized.Load() {
			<-e.podEventsWatcher.Started()
		}
		close(ch)
	}()
	return ch
}

func (e *executionWatcher) Started() <-chan struct{} {
	return e.initialCommitCh
}

func (e *executionWatcher) Updated() <-chan struct{} {
	return e.update.Channel()
}

func NewExecutionWatcher(parentCtx context.Context, clientSet kubernetes.Interface, namespace, id string, signature []stage.Signature, scheduledAt time.Time) ExecutionWatcher {
	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Build initial data
	opts := ExecutionStateOptions{
		ResourceId:  id,
		Namespace:   namespace,
		Signature:   signature,
		ScheduledAt: scheduledAt,
	}

	// Prepare initial execution watcher
	watcher := &executionWatcher{
		state:           NewExecutionState(nil, nil, NewJobEvents(nil), NewPodEvents(nil), &opts).(*executionState),
		uncommitted:     NewExecutionState(nil, nil, NewJobEvents(nil), NewPodEvents(nil), &opts).(*executionState),
		update:          store.NewUpdate(),
		podEventsOptsCh: make(chan metav1.ListOptions, 1),
		initialCommitCh: make(chan struct{}),
	}

	//update := store.NewBatchUpdate(5 * time.Millisecond)
	update := store.NewUpdate()
	job := store.NewValue[batchv1.Job](ctx, update)
	pod := store.NewValue[corev1.Pod](ctx, update)
	jobEvents := store.NewList[corev1.Event](ctx, update)
	podEvents := store.NewList[corev1.Event](ctx, update)

	// Optimistically, start watching all the easily reachable resources
	watcher.jobWatcher = NewJobWatcher(ctx, clientSet.BatchV1().Jobs(namespace), metav1.ListOptions{
		FieldSelector: "metadata.name=" + id,
	}, job.Put)
	watcher.podWatcher = NewPodWatcher(ctx, clientSet.CoreV1().Pods(namespace), metav1.ListOptions{
		LabelSelector: constants.ResourceIdLabelName + "=" + id,
	}, pod.Put)
	watcher.jobEventsWatcher = NewEventsWatcher(ctx, clientSet.CoreV1().Events(namespace), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + id,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
	}, jobEvents.Put)

	watcher.podEventsWatcher = NewAsyncEventsWatcher(ctx, clientSet.CoreV1().Events(namespace), watcher.podEventsOptsCh, podEvents.Put)

	// Watch for errors
	go func() {
		<-watcher.jobWatcher.Done()
		if !errors.Is(watcher.jobWatcher.Err(), context.Canceled) {
			fmt.Println("DEBUG: JOB WATCHER ERROR", watcher.jobWatcher.Err())
		}
	}()
	go func() {
		<-watcher.podWatcher.Done()
		if !errors.Is(watcher.podWatcher.Err(), context.Canceled) {
			fmt.Println("DEBUG: POD WATCHER ERROR", watcher.podWatcher.Err())
		}
	}()
	go func() {
		<-watcher.podEventsWatcher.Done()
		if !errors.Is(watcher.podEventsWatcher.Err(), context.Canceled) {
			fmt.Println("DEBUG: POD EVENTS WATCHER ERROR", watcher.podEventsWatcher.Err())
		}
	}()
	go func() {
		<-watcher.jobEventsWatcher.Done()
		if !errors.Is(watcher.jobEventsWatcher.Err(), context.Canceled) {
			fmt.Println("DEBUG: JOB EVENTS WATCHER ERROR", watcher.jobEventsWatcher.Err())
		}
	}()

	// Close updates channel when all individual watchers are complete
	go func() {
		<-watcher.jobWatcher.Done()
		<-watcher.podWatcher.Done()
		<-watcher.jobEventsWatcher.Done()
		if watcher.podEventsInitialized.Load() {
			<-watcher.podEventsWatcher.Done()
		}
		watcher.update.Close()
		update.Close()
		if watcher.podEventsInitialized.CompareAndSwap(false, true) {
			close(watcher.podEventsOptsCh)
		}
		watcher.podEventsOptsCh = nil
		if watcher.initialCommitInitialized.CompareAndSwap(false, true) {
			close(watcher.initialCommitCh)
		}
	}()

	// Create helper to read the latest data
	podEventsCh := podEvents.Channel()
	jobEventsCh := jobEvents.Channel()
	readLatestData := func() {
		time.Sleep(5 * time.Millisecond)

		if job.Latest() != nil {
			watcher.uncommitted.job = NewJob(job.Latest())
		}
		if pod.Latest() != nil {
			watcher.uncommitted.pod = NewPod(pod.Latest())
		}
		for ok := true; ok; { // TODO?
			var event *corev1.Event
			select {
			case event, ok = <-podEventsCh:
				if ok {
					watcher.uncommitted.podEvents = NewPodEvents(append(watcher.uncommitted.podEvents.Original(), event))
				}
			case event, ok = <-jobEventsCh:
				if ok {
					watcher.uncommitted.jobEvents = NewJobEvents(append(watcher.uncommitted.jobEvents.Original(), event))
				}
			default:
				ok = false
			}
		}
		watcher.initializePodEventsWatcher()
	}

	// Load the data
	// TODO: handle stream errors
	go func() {
		defer func() {
			ctxCancel()
		}()
		var next func(bool)
		next = func(force bool) {
			// Read the latest data
			readLatestData()

			// TODO: Avoid in the next iteration if it's not new and non-critical
			//hasMissingPodEvents := false
			//hasMissingPod := false
			//hasMissingJob := false
			//hasMissingJobEvents := false
			hasMissingCriticalPod := false
			hasMissingCriticalJob := false

			// TODO Determine if there are missing pod events
			// TODO Determine if there are missing job events

			// TODO: Container Started event is there, pod is not

			// Determine if there is missing pod state after critical error
			if watcher.uncommitted.podEvents.Error() && (watcher.uncommitted.pod == nil || !watcher.uncommitted.pod.Finished()) {
				hasMissingCriticalPod = true
			}

			// Determine if there is missing pod state after job's success
			if (watcher.uncommitted.job != nil && watcher.uncommitted.job.Finished() && watcher.uncommitted.job.ExecutionError() == "") && (watcher.uncommitted.pod == nil || !watcher.uncommitted.pod.Finished()) {
				hasMissingCriticalPod = true
			}

			// Determine if there is missing pod state after job's success (based on job events)
			if watcher.uncommitted.jobEvents.Success() && (watcher.uncommitted.pod == nil || !watcher.uncommitted.pod.Finished()) {
				hasMissingCriticalPod = true
			}

			// Determine if there is missing job state after pod's error
			if watcher.uncommitted.podEvents.Error() && (watcher.uncommitted.job == nil || !watcher.uncommitted.job.Finished()) {
				hasMissingCriticalJob = true
			}

			// Determine if there is missing job state after pod's error (based on pod events)
			if watcher.uncommitted.podEvents.Error() && (watcher.uncommitted.job == nil || !watcher.uncommitted.job.Finished()) {
				hasMissingCriticalJob = true
			}

			// Load missing job updates gracefully
			if force {
				var wg sync.WaitGroup
				if hasMissingCriticalJob {
					wg.Add(1)
					go func() {
						watcher.jobWatcher.Update(2 * time.Second)
						wg.Done()
					}()
				}
				if hasMissingCriticalPod {
					wg.Add(1)
					go func() {
						watcher.podWatcher.Update(2 * time.Second)
						wg.Done()
					}()
				}
				wg.Wait()
				readLatestData()
			} else {
				timer := time.After(500 * time.Millisecond)

				if hasMissingCriticalJob {
					select {
					case <-job.Next():
					case <-timer:
						t := make(chan time.Time)
						timer = t
						close(t)
					}
				}

				// Load missing pod updates gracefully
				if hasMissingCriticalPod { // TODO: Check if that's  already proper
					if watcher.uncommitted.pod == nil || pod.Latest() != watcher.uncommitted.pod.Original() {
						select {
						case <-pod.Next():
						case <-timer:
						}
					}
				}
			}

			// Reiterate checking the status
			if !force && (hasMissingCriticalJob || hasMissingCriticalPod) {
				next(true)
			}
		}

		<-watcher.baseStarted()

		next(false) // TODO: don't wait for baseStarted, as pod events are delayed then
		if watcher.podEventsInitialized.Load() {
			<-watcher.podEventsWatcher.Started()
			next(false)
		}
		watcher.Commit()
		for {
			_, ok := <-update.Channel()
			if !ok {
				break
			}

			next(false)
			watcher.Commit()

			if watcher.State().Completed() {
				return
			}
		}
		next(false)
		watcher.Commit()
	}()

	return watcher
}
