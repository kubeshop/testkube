package testworkflowcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

var (
	ErrMissingPod = errors.New("missing pod information")
)

type executionState struct {
	ctx       context.Context
	jobEvents []*corev1.Event
	podEvents []*corev1.Event
	pod       *corev1.Pod
	job       *batchv1.Job
	mu        sync.RWMutex
	jobMu     sync.Mutex
	updatesCh chan struct{}
}

func (e *executionState) emit() {
	defer func() {
		recover()
	}()

	select {
	case e.updatesCh <- struct{}{}:
	default:
	}
}

func (e *executionState) registerJobEvent(event *corev1.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.jobEvents = append(e.jobEvents, event)
}

func (e *executionState) registerPodEvent(event *corev1.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.podEvents = append(e.podEvents, event)
}

func (e *executionState) registerPod(pod *corev1.Pod) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pod = pod
}

func (e *executionState) registerJob(job *batchv1.Job) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.job = job
}

func (e *executionState) JobEvents() []*corev1.Event {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.jobEvents
}

func (e *executionState) PodEvents() []*corev1.Event {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.podEvents
}

func (e *executionState) Pod() *corev1.Pod {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.pod
}

func (e *executionState) Job() *batchv1.Job {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.job
}

func (e *executionState) Updated() <-chan struct{} {
	ch := make(chan struct{})
	if e.ctx.Err() != nil {
		close(ch)
		return ch
	}
	return e.updatesCh
}

func (e *executionState) ActionGroups() (actions actiontypes.ActionGroups, err error) {
	if e.pod != nil {
		err = json.Unmarshal([]byte(e.pod.Annotations[constants.SpecAnnotationName]), &actions)
		return
	}
	if e.job != nil {
		err = json.Unmarshal([]byte(e.job.Spec.Template.Annotations[constants.SpecAnnotationName]), &actions)
		return
	}
	return nil, ErrMissingPod
}

func (e *executionState) CompletionTimestamp() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get from pod if it's possible
	if e.pod != nil {
		return watchers.GetPodCompletionTimestamp(e.pod)
	}

	// Get from job if it's possible
	if e.job != nil {
		return watchers.GetJobCompletionTimestamp(e.job)
	}

	// Get the information based on the Job events
	for _, event := range e.jobEvents {
		if event.Reason == "BackoffLimitExceeded" || event.Reason == "Completed" {
			// (BackoffLimitExceeded) Job has reached the specified backoff limit
			// (Completed) Job completed
			return watchers.GetEventTimestamp(event)
		}
	}

	return time.Time{}
}

// TODO: Prepare something like WatcherSet, that will have hooks upon watchers, to build info like that
//
//	also, the Pod Events Watcher inside will be able to use job events pod name fallback
func (e *executionState) PodName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get directly from the pod if it's possible
	if e.pod != nil {
		return e.pod.Name
	}

	// Get the information based on the Job events
	for _, event := range e.jobEvents {
		if event.Reason == "SuccessfulCreate" {
			// (SuccessfulCreate) Created pod: 66c49ca3284bce9380023421-78fmp
			return event.Message[strings.LastIndex(event.Message, " ")+1:]
		}
	}

	// Get the information based on the Pod events
	for _, event := range e.podEvents {
		if event.Reason == "Scheduled" {
			// (Scheduled) Successfully assigned distributed-tests/66c49ca3284bce9380023421-78fmp to homelab
			match := regexp.MustCompile(`/(\S+)`).FindStringSubmatch(event.Message)
			if match != nil {
				return match[1]
			}
		}
	}

	return ""
}

func (e *executionState) PodCreationTimestamp() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get directly from the pod if it's possible
	if e.pod != nil {
		return e.pod.CreationTimestamp.Time
	}

	// Get the information based on the Job events
	for _, event := range e.jobEvents {
		if event.Reason == "SuccessfulCreate" {
			// (SuccessfulCreate) Created pod: 66c49ca3284bce9380023421-78fmp
			return watchers.GetEventTimestamp(event)
		}
	}

	// Get the information based on the Pod events
	if len(e.podEvents) > 0 {
		return e.podEvents[0].CreationTimestamp.Time
	}

	return time.Time{}
}

func (e *executionState) PodNodeName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get directly from the pod if it's possible
	if e.pod != nil {
		nodeName := e.pod.Status.NominatedNodeName
		if nodeName == "" {
			nodeName = e.pod.Spec.NodeName
		}
		return nodeName
	}

	// Get the information based on the Pod events
	for _, event := range e.podEvents {
		if event.Reason == "Scheduled" {
			// (Scheduled) Successfully assigned distributed-tests/66c49ca3284bce9380023421-78fmp to homelab
			return event.Message[strings.LastIndex(event.Message, " ")+1:]
		}
	}

	return ""
}

func (e *executionState) Namespace() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.pod != nil {
		return e.pod.Namespace
	}
	if e.job != nil {
		return e.job.Namespace
	}
	if len(e.podEvents) > 0 {
		return e.podEvents[0].Namespace
	}
	if len(e.jobEvents) > 0 {
		return e.jobEvents[0].Namespace
	}

	return ""
}

func readImmediate[T any](ch <-chan T, process func(T), end func()) int {
	count := 0
	for {
		select {
		case v, ok := <-ch:
			if ok {
				process(v)
				count++
			} else {
				end()
				return count
			}
		default:
			return count
		}
	}
}

func NewExecutionState(parentCtx context.Context, watcher watchers.ExecutionWatcher) *executionState {
	ctx, ctxCancel := context.WithCancel(parentCtx)
	state := &executionState{
		ctx:       ctx,
		updatesCh: make(chan struct{}, 1),
	}

	// Get all data source channels
	var chMu sync.Mutex
	jobEventsCh := watcher.JobEvents()
	podEventsCh := watcher.PodEvents()
	podCh := watcher.Pod()
	jobCh := watcher.Job()

	log := func(args ...interface{}) {
		// FIXME: delete?
		//if state.Job() != nil {
		//	args = append([]interface{}{ui.LightBlue("x: " + state.Job().Name)}, args...)
		//}
		//fmt.Println(args...)
	}

	// Compute status
	isRunning := func() bool {
		chMu.Lock()
		defer chMu.Unlock()
		return jobEventsCh != nil || podEventsCh != nil || podCh != nil || jobCh != nil
	}

	// Declare the functions used for reading
	var readImmediateJobEvents, readImmediatePodEvents, readImmediatePod, readImmediateJob func() int

	// Configure helpers to close the channels
	closeJobEventsChannel := func() {
		log("close job events channel")
		defer log("close job events channel: END")
		chMu.Lock()
		defer chMu.Unlock()
		if jobEventsCh != nil {
			jobEventsCh = nil
		}
	}
	closePodEventsChannel := func() {
		log("close pod events channel")
		defer log("close pod events channel: END")
		chMu.Lock()
		defer chMu.Unlock()
		if podEventsCh != nil {
			podEventsCh = nil
		}
	}
	closePodChannel := func() {
		log("close pod channel")
		defer log("close pod channel: END")
		chMu.Lock()
		if podCh == nil {
			chMu.Unlock()
			return
		}
		podCh = nil
		chMu.Unlock()

		// Load the missing pod events
		readImmediatePodEvents()
		watcher.ReadPodEventsAt(watcher.PodCompletionTimestamp(true), 1*time.Second) // FIXME
		readImmediatePodEvents()

		// Load missing job
		if state.Pod() != nil {
			// Wait a moment if there won't be maybe finished job
			if state.Pod().DeletionTimestamp != nil && !watchers.IsJobFinished(state.Job()) {
				time.Sleep(300 * time.Millisecond)
				readImmediateJob()
			}

			// Try to obtain the latest Job data
			if state.Pod().DeletionTimestamp != nil && !watchers.IsJobFinished(state.Job()) && !watcher.JobFinished() {
				watcher.RefreshJob(2 * time.Second)
				readImmediateJob()
			}
		}

		// Close the pod events channel
		closePodEventsChannel()
	}
	closeJobChannel := func() {
		log("close job channel")
		defer log("close job channel: END")
		chMu.Lock()
		if jobCh == nil {
			chMu.Unlock()
			return
		}
		jobCh = nil
		chMu.Unlock()

		// Load the missing pod information
		readImmediatePod()

		// Wait a moment if there won't be maybe finished job
		if !watchers.IsPodFinished(state.Pod()) {
			time.Sleep(300 * time.Millisecond)
			readImmediatePod()
		}

		// Try to obtain the latest Job data
		if !watchers.IsPodFinished(state.Pod()) && !watcher.PodFinished() {
			watcher.RefreshPod(2 * time.Second)
			readImmediatePod()
		}

		// Load the missing job events
		readImmediateJobEvents()
		watcher.ReadJobEventsAt(watcher.JobCompletionTimestamp(true), 1*time.Second) // FIXME
		readImmediateJobEvents()

		// Close the job events channel
		closeJobEventsChannel()
	}

	// Configure helpers to register the data
	registerPod := func(pod *corev1.Pod) {
		log("register pod", watchers.IsPodFinished(pod), watchers.IsJobFinished(state.Job()), pod.DeletionTimestamp)
		defer log("register pod: END")
		state.registerPod(pod)
		if watchers.IsPodFinished(pod) {
			// Check if there is more details Pod information waiting
			if readImmediatePod() > 0 {
				return
			}
			closePodChannel()
		}
		readImmediatePodEvents()

		// Load the missing job information in case the pod is deleted
		readImmediateJob()
	}
	registerJob := func(job *batchv1.Job) {
		log("register job", watchers.IsPodFinished(state.Pod()), watchers.IsJobFinished(job))
		defer log("register job: END")

		state.registerJob(job)
		if watchers.IsJobFinished(job) {
			// Check if there is more details Job information waiting
			if readImmediateJob() > 0 {
				return
			}
			closeJobChannel()
		}
		readImmediateJobEvents()

		// Load the missing pod information in case the job is finished
		readImmediatePod()
		if !watchers.IsPodFinished(state.Pod()) && watchers.IsJobFinished(job) {
			watcher.RefreshPod(2 * time.Second) // TODO: Use Ensure?
			readImmediatePod()
		}
	}

	registerJobEvent := func(event *corev1.Event) {
		log("register job event", event.Reason)
		defer log("register job event: END")
		state.registerJobEvent(event)
	}

	registerPodEvent := func(event *corev1.Event) {
		log("register pod event", event.Reason)
		defer log("register pod event: END")
		state.registerPodEvent(event)
	}

	// Configure helpers to read the most recent data
	readImmediateJobEvents = func() int {
		log("read immediate job events")
		defer log("read immediate job events: END")
		return readImmediate(jobEventsCh, registerJobEvent, closeJobEventsChannel)
	}
	readImmediatePodEvents = func() int {
		log("read immediate pod events")
		defer log("read immediate pod events: END")
		return readImmediate(podEventsCh, registerPodEvent, closePodEventsChannel)
	}
	readImmediatePod = func() int {
		log("read immediate pod")
		defer log("read immediate pod: END")
		return readImmediate(podCh, registerPod, closePodChannel)
	}
	readImmediateJob = func() int {
		log("read immediate job")
		defer log("read immediate job: END")
		return readImmediate(jobCh, registerJob, closeJobChannel)
	}

	go func() {
		<-ctx.Done()
		log("closing updates channel")
		close(state.updatesCh)
	}()

	readImmediateJobEvents()
	readImmediatePodEvents()
	readImmediateJob()
	readImmediatePod()

	// Fast-track
	if state.Pod() != nil && watchers.IsPodFinished(state.Pod()) && watchers.IsJobFinished(state.Job()) {
		log("immediate finish")
		ctxCancel()
		state.emit()
		return state
	}

	// Watch for changes
	go func() {
		defer func() {
			ctxCancel()
			state.emit()
			log("finishing the executionstate iteration")
		}()

		for isRunning() {
			// Prioritize loading events that are on hold

			if readImmediateJobEvents()+readImmediatePodEvents() > 0 {
				state.emit()
			}

			// Load next details
			select {
			case <-ctx.Done():
				return
			case event, ok := <-jobEventsCh:
				if ok {
					state.registerJobEvent(event)
				} else {
					closeJobEventsChannel()
				}
				state.emit()
			case event, ok := <-podEventsCh:
				if ok {
					state.registerPodEvent(event)
				} else {
					closePodEventsChannel()
				}
				state.emit()
			case pod, ok := <-podCh:
				if ok {
					registerPod(pod)
				} else {
					closePodChannel()
				}
				state.emit()
			case job, ok := <-jobCh:
				if ok {
					registerJob(job)
				} else {
					closeJobChannel()
				}
				state.emit()
			}
		}
	}()

	return state
}
