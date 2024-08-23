package watchers

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

var (
	ErrMissingJob = errors.New("missing job information")
)

type executionWatcher struct {
	id                     string
	namespace              string
	podName                string
	podNodeName            string
	podIp                  string
	podExecutionError      string
	jobExecutionError      string
	actionsSerialized      *[]byte
	signatureSerialized    *[]byte
	podEventTimestamp      *time.Time
	jobEventTimestamp      *time.Time
	podCreationTimestamp   *time.Time
	jobCreationTimestamp   *time.Time
	podCompletionTimestamp *time.Time
	jobCompletionTimestamp *time.Time
	podFinished            bool
	jobFinished            bool
	jobWatcher             JobWatcher
	podWatcher             PodWatcher
	jobEventsWatcher       EventsWatcher
	podEventsWatcher       EventsWatcher

	podListOptionsCh     chan metav1.ListOptions
	podEventsInitialized atomic.Bool

	updatesCh chan struct{}

	latestPod *corev1.Pod
	latestJob *batchv1.Job

	ctx       context.Context
	ctxCancel context.CancelFunc

	mu        sync.RWMutex
	receiveMu sync.Mutex
}

type ExecutionWatcher interface {
	Id() string
	Namespace() string
	ExecutionError() string
	Signature() ([]stage.Signature, error)
	ActionGroups() (actiontypes.ActionGroups, error)
	PodName() string
	PodNodeName() string
	PodIP() string
	PodCreationTimestamp(loose bool) time.Time
	JobCreationTimestamp(loose bool) time.Time
	CompletionTimestamp() time.Time
	PodCompletionTimestamp(loose bool) time.Time
	JobCompletionTimestamp(loose bool) time.Time
	PodFinished() bool
	JobFinished() bool

	JobExists() bool
	PodExists() bool
	JobEvents() <-chan *corev1.Event
	JobEventsErr() error
	PodEvents() <-chan *corev1.Event
	PodEventsErr() error
	LatestJob() *batchv1.Job
	Job() <-chan *batchv1.Job
	JobErr() error
	LatestPod() *corev1.Pod
	Pod() <-chan *corev1.Pod
	PodErr() error
	ReadJobEventsAt(ts time.Time, timeout time.Duration)
	ReadPodEventsAt(ts time.Time, timeout time.Duration)
	RefreshPod(timeout time.Duration)
	RefreshJob(timeout time.Duration)
	Started() <-chan struct{}

	Updated() <-chan struct{}
}

func (e *executionWatcher) initializePodEventsWatcher(name string) {
	if name != "" && e.podEventsInitialized.CompareAndSwap(false, true) {
		e.podListOptionsCh <- metav1.ListOptions{
			FieldSelector: "involvedObject.name=" + name,
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		}
		close(e.podListOptionsCh)
	}
}

func (e *executionWatcher) receiveJob(job *batchv1.Job) {
	e.receiveMu.Lock()
	defer e.receiveMu.Unlock()

	e.registerJob(job)
}

func (e *executionWatcher) receivePod(pod *corev1.Pod) {
	e.receiveMu.Lock()
	defer e.receiveMu.Unlock()

	e.initializePodEventsWatcher(pod.Name)
	e.registerPod(pod)
}

func (e *executionWatcher) receiveJobEvent(event *corev1.Event) {
	e.receiveMu.Lock()
	defer e.receiveMu.Unlock()

	e.registerJobEvent(event)
	e.initializePodEventsWatcher(e.PodName())
}

func (e *executionWatcher) receivePodEvent(event *corev1.Event) {
	e.receiveMu.Lock()
	defer e.receiveMu.Unlock()

	e.registerPodEvent(event)
	e.initializePodEventsWatcher(e.PodName())
}

func (e *executionWatcher) registerJob(job *batchv1.Job) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Save the job information
	e.latestJob = job
	e.jobCreationTimestamp = common.Ptr(job.CreationTimestamp.Time)
	if IsJobFinished(job) {
		e.jobFinished = true
		e.jobCompletionTimestamp = common.Ptr(GetJobCompletionTimestamp(job))
	}
	if e.actionsSerialized == nil {
		e.actionsSerialized = common.Ptr([]byte(job.Spec.Template.Annotations[constants.SpecAnnotationName]))
	}
	if e.signatureSerialized == nil {
		e.signatureSerialized = common.Ptr([]byte(job.Annotations[constants.SignatureAnnotationName]))
	}
	e.jobExecutionError = GetJobError(job)
}

func (e *executionWatcher) registerPod(pod *corev1.Pod) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Save the pod information
	e.latestPod = pod
	e.podName = pod.Name
	e.podCreationTimestamp = common.Ptr(pod.CreationTimestamp.Time)
	if IsPodFinished(pod) {
		e.podFinished = true
		e.podCompletionTimestamp = common.Ptr(GetPodCompletionTimestamp(pod))
	}
	if e.actionsSerialized == nil {
		e.actionsSerialized = common.Ptr([]byte(pod.Annotations[constants.SpecAnnotationName]))
	}
	if pod.Status.PodIP != "" {
		e.podIp = pod.Status.PodIP
	}
	e.podExecutionError = GetPodError(pod)
}

func (e *executionWatcher) registerJobEvent(event *corev1.Event) {
	// Detect the dependant Pod creation
	if event.Reason == "SuccessfulCreate" {
		// (SuccessfulCreate) Created pod: 66c49ca3284bce9380023421-78fmp
		e.mu.Lock()
		e.podName = event.Message[strings.LastIndex(event.Message, " ")+1:]
		e.podCreationTimestamp = common.Ptr(GetEventTimestamp(event))
		e.mu.Unlock()
	}

	// Detect the job end
	if event.Reason == "BackoffLimitExceeded" || event.Reason == "Completed" {
		// (BackoffLimitExceeded) Job has reached the specified backoff limit
		// (Completed) Job completed
		e.mu.Lock()
		e.jobFinished = true
		e.jobCompletionTimestamp = common.Ptr(GetEventTimestamp(event))
		e.mu.Unlock()
	}
}

func (e *executionWatcher) registerPodEvent(event *corev1.Event) {
	// Save the first pod timestamp
	e.mu.Lock()
	if e.podEventTimestamp == nil || e.podEventTimestamp.After(GetEventTimestamp(event)) {
		e.podEventTimestamp = common.Ptr(GetEventTimestamp(event))
	}
	e.mu.Unlock()

	// Detect the Pod's name and the Node's name
	if event.Reason == "Scheduled" {
		// (Scheduled) Successfully assigned distributed-tests/66c49ca3284bce9380023421-78fmp to homelab
		match := regexp.MustCompile(`/\s+(\S+)`).FindStringSubmatch(event.Message)
		if match != nil {
			e.mu.Lock()
			e.podName = match[1]
			e.podNodeName = event.Message[strings.LastIndex(event.Message, " ")+1:]
			e.mu.Unlock()
		}
	}
}

func (e *executionWatcher) Id() string {
	return e.id
}

func (e *executionWatcher) Namespace() string {
	return e.namespace
}

func (e *executionWatcher) ExecutionError() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.podExecutionError != "" {
		return e.podExecutionError
	}
	if e.jobExecutionError != "" {
		return e.jobExecutionError
	}
	return ""
}

func (e *executionWatcher) PodName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.podName
}

func (e *executionWatcher) PodNodeName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.podNodeName
}

func (e *executionWatcher) PodIP() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.podIp
}

func (e *executionWatcher) ActionGroups() (actions actiontypes.ActionGroups, err error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.actionsSerialized != nil {
		err = json.Unmarshal(*e.actionsSerialized, &actions)
		return
	}
	return nil, ErrMissingJob
}

func (e *executionWatcher) Signature() ([]stage.Signature, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.signatureSerialized != nil {
		return stage.GetSignatureFromJSON(*e.signatureSerialized)
	}
	return nil, ErrMissingJob
}

func (e *executionWatcher) PodCreationTimestamp(loose bool) time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.podCreationTimestamp != nil {
		return *e.podCreationTimestamp
	}
	if loose && e.podEventTimestamp != nil {
		return *e.podEventTimestamp
	}
	return time.Time{}
}

func (e *executionWatcher) JobCreationTimestamp(loose bool) time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.jobCreationTimestamp != nil {
		return *e.jobCreationTimestamp
	}
	if loose && e.jobEventTimestamp != nil {
		return *e.jobEventTimestamp
	}
	return time.Time{}
}

func (e *executionWatcher) PodCompletionTimestamp(loose bool) time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.podCompletionTimestamp != nil {
		return *e.podCompletionTimestamp
	}
	if loose && e.jobCompletionTimestamp != nil {
		return *e.jobCompletionTimestamp
	}
	return time.Time{}
}

func (e *executionWatcher) JobCompletionTimestamp(loose bool) time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.jobCompletionTimestamp != nil {
		return *e.jobCompletionTimestamp
	}
	if loose && e.podCompletionTimestamp != nil {
		return *e.podCompletionTimestamp
	}
	return time.Time{}
}

func (e *executionWatcher) CompletionTimestamp() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.podCompletionTimestamp != nil {
		return *e.podCompletionTimestamp
	}
	if e.jobCompletionTimestamp != nil {
		return *e.jobCompletionTimestamp
	}
	return time.Time{}
}

func (e *executionWatcher) PodFinished() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.podFinished
}

func (e *executionWatcher) JobFinished() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.jobFinished
}

func (e *executionWatcher) JobEvents() <-chan *corev1.Event {
	return e.jobEventsWatcher.Channel()
}

func (e *executionWatcher) PodEvents() <-chan *corev1.Event {
	return e.podEventsWatcher.Channel()
}

func (e *executionWatcher) Job() <-chan *batchv1.Job {
	return e.jobWatcher.Channel()
}

func (e *executionWatcher) Pod() <-chan *corev1.Pod {
	return e.podWatcher.Channel()
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

func (e *executionWatcher) Started() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-e.jobWatcher.Started()
		<-e.jobEventsWatcher.Started()
		<-e.podWatcher.Started()
		if e.PodName() != "" {
			<-e.podEventsWatcher.Started()
		}
		close(ch)
	}()
	return ch
}

func (e *executionWatcher) LatestJob() *batchv1.Job {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.latestJob
}

func (e *executionWatcher) LatestPod() *corev1.Pod {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.latestPod
}

func (e *executionWatcher) JobExists() bool {
	return e.Job() != nil
}

func (e *executionWatcher) PodExists() bool {
	return e.Pod() != nil
}

func (e *executionWatcher) Updated() <-chan struct{} {
	ch := make(chan struct{})
	if e.ctx.Err() != nil {
		close(ch)
		return ch
	}
	return e.updatesCh
}

func (e *executionWatcher) emit() {
	defer func() {
		recover()
	}()

	select {
	case e.updatesCh <- struct{}{}:
	default:
	}
}

func NewExecutionWatcher(parentCtx context.Context, clientSet kubernetes.Interface, namespace, id string, signature []stage.Signature) ExecutionWatcher {
	// Create local context for stopping all the processes
	ctx, ctxCancel := context.WithCancel(parentCtx)

	// Prepare initial execution watcher
	watcher := &executionWatcher{
		ctx:              ctx,
		ctxCancel:        ctxCancel,
		id:               id,
		namespace:        namespace,
		updatesCh:        make(chan struct{}, 1),
		podListOptionsCh: make(chan metav1.ListOptions),
	}

	// Pre-configure signature if it's known
	if signature != nil {
		signatureSerialized, err := json.Marshal(signature)
		if err == nil {
			watcher.signatureSerialized = &signatureSerialized
		}
	}

	// Optimistically, start watching all the easily reachable resources
	watcher.jobWatcher = NewJobWatcher(ctx, clientSet.BatchV1().Jobs(namespace), metav1.ListOptions{
		FieldSelector: "metadata.name=" + id,
	}, 1, watcher.receiveJob)
	watcher.podWatcher = NewPodWatcher(ctx, clientSet.CoreV1().Pods(namespace), metav1.ListOptions{
		LabelSelector: constants.ResourceIdLabelName + "=" + id,
	}, 1, watcher.receivePod)
	watcher.jobEventsWatcher = NewEventsWatcher(ctx, clientSet.CoreV1().Events(namespace), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + id,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
	}, 10, watcher.receiveJobEvent)

	watcher.podEventsWatcher = NewAsyncEventsWatcher(ctx, clientSet.CoreV1().Events(namespace), watcher.podListOptionsCh, 10, watcher.receivePodEvent)

	// Close updates channel when all individual watchers are complete
	go func() {
		<-watcher.jobWatcher.Done()
		<-watcher.podWatcher.Done()
		<-watcher.jobEventsWatcher.Done()
		if watcher.podEventsInitialized.Load() {
			<-watcher.podEventsWatcher.Done()
		}
		close(watcher.updatesCh)
		if !watcher.podEventsInitialized.Load() {
			close(watcher.podListOptionsCh)
		}
		watcher.podListOptionsCh = nil
	}()

	return watcher
}
