package watchers

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	store2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/store"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	ReadLatestBufferingTimeframe = 5 * time.Millisecond
	ReadCriticalGracefullyTime   = 750 * time.Millisecond
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
	update      store2.Update
	mu          sync.RWMutex
}

type ExecutionWatcher interface {
	State() ExecutionState
	Commit()

	JobEventsErr() error
	PodEventsErr() error
	JobErr() error
	PodErr() error

	RefreshPod(ctx context.Context)
	RefreshJob(ctx context.Context)

	Started() <-chan struct{}
	Updated(ctx context.Context) <-chan struct{}
	Next() <-chan struct{}
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
	e.state = common.Ptr(*e.uncommitted)
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

func (e *executionWatcher) RefreshPod(ctx context.Context) {
	e.podWatcher.Update(ctx)
}

func (e *executionWatcher) RefreshJob(ctx context.Context) {
	e.jobWatcher.Update(ctx)
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

func (e *executionWatcher) Updated(ctx context.Context) <-chan struct{} {
	return e.update.Channel(ctx)
}

func (e *executionWatcher) Next() <-chan struct{} {
	return e.update.Next()
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
		update:          store2.NewUpdate(),
		podEventsOptsCh: make(chan metav1.ListOptions, 1),
		initialCommitCh: make(chan struct{}),
	}

	update := store2.NewUpdate()
	job := store2.NewValue[batchv1.Job](ctx, update)
	pod := store2.NewValue[corev1.Pod](ctx, update)
	jobEvents := store2.NewList[corev1.Event](ctx, update)
	podEvents := store2.NewList[corev1.Event](ctx, update)

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
	}()
	go func() {
		<-watcher.podWatcher.Done()
	}()
	go func() {
		<-watcher.podEventsWatcher.Done()
	}()
	go func() {
		<-watcher.jobEventsWatcher.Done()
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
		time.Sleep(ReadLatestBufferingTimeframe)

		if job.Latest() != nil {
			watcher.uncommitted.job = NewJob(job.Latest())
		}
		if pod.Latest() != nil {
			watcher.uncommitted.pod = NewPod(pod.Latest())
		}
		for ok := true; ok; {
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
			hasMissingCriticalPod := false
			hasMissingCriticalJob := false

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
						watcher.jobWatcher.Update(ctx)
						wg.Done()
					}()
				}
				if hasMissingCriticalPod {
					wg.Add(1)
					go func() {
						watcher.podWatcher.Update(ctx)
						wg.Done()
					}()
				}
				wg.Wait()
				readLatestData()
			} else {
				timer := time.After(ReadCriticalGracefullyTime)

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
				if hasMissingCriticalPod {
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

		next(false)
		if watcher.podEventsInitialized.Load() {
			<-watcher.podEventsWatcher.Started()
			next(false)
		}
		watcher.Commit()
		for {
			_, ok := <-update.Channel(ctx)
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
