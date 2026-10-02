package watchers

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

var (
	ErrMissingData = errors.New("missing data to fulfill request")
)

type executionState struct {
	job       Job
	pod       Pod
	jobEvents JobEvents
	podEvents PodEvents
	options   *ExecutionStateOptions
}

type ExecutionStateOptions struct {
	ResourceId     string
	RootResourceId string
	Namespace      string
	Signature      []stage.Signature
	ActionGroups   actiontypes.ActionGroups
	ScheduledAt    time.Time
}

type ExecutionState interface {
	Job() Job
	Pod() Pod
	JobEvents() JobEvents
	PodEvents() PodEvents
	JobExists() bool
	PodExists() bool

	Namespace() string
	ResourceId() string
	RootResourceId() string
	RunnerId() string
	PodName() string
	PodNodeName() string
	PodIP() string
	PodDeletionTimestamp() time.Time
	CompletionTimestamp() time.Time
	ContainerStartTimestamp(name string) time.Time
	ContainerStarted(name string) bool
	ContainerFinished(name string) bool
	ContainerFailed(name string) bool
	Signature() ([]stage.Signature, error)
	ActionGroups() (actiontypes.ActionGroups, error)
	InternalConfig() (testworkflowconfig.InternalConfig, error)
	ScheduledAt() time.Time
	InitializationTimeout() time.Duration

	ExecutionError() string
	CurrentCause() *testkube.Cause
	TerminationCause() *testkube.Cause
	JobExecutionError() string
	PodExecutionError() string
	Debug() map[string]string

	PodCreationTimestamp() time.Time
	EstimatedPodCreationTimestamp() time.Time

	PodStartTimestamp() time.Time
	EstimatedPodStartTimestamp() time.Time

	JobCreationTimestamp() time.Time
	EstimatedJobCreationTimestamp() time.Time

	ContainersReady() bool

	PodCreated() bool
	PodStarted() bool
	Completed() bool
}

func NewExecutionState(job Job, pod Pod, jobEvents JobEvents, podEvents PodEvents, opts *ExecutionStateOptions) ExecutionState {
	if opts == nil {
		opts = &ExecutionStateOptions{}
	}
	return &executionState{
		job:       job,
		pod:       pod,
		jobEvents: jobEvents,
		podEvents: podEvents,
		options:   opts,
	}
}

func (e *executionState) Job() Job {
	return e.job
}

func (e *executionState) Pod() Pod {
	return e.pod
}

func (e *executionState) JobExists() bool {
	return e.job != nil
}

func (e *executionState) PodExists() bool {
	return e.pod != nil
}

func (e *executionState) PodName() string {
	if e.pod != nil {
		return e.pod.Name()
	}
	if e.podEvents.Name() != "" {
		return e.podEvents.Name()
	}
	if e.jobEvents.PodName() != "" {
		return e.jobEvents.PodName()
	}
	return ""
}

func (e *executionState) PodNodeName() string {
	if e.pod != nil && e.pod.NodeName() != "" {
		return e.pod.NodeName()
	}
	if e.podEvents.NodeName() != "" {
		return e.podEvents.NodeName()
	}
	return ""
}

func (e *executionState) PodIP() string {
	if e.pod != nil {
		return e.pod.IP()
	}
	return ""
}

func (e *executionState) ContainersReady() bool {
	return e.pod != nil && e.pod.ContainersReady()
}

func (e *executionState) PodDeletionTimestamp() time.Time {
	if e.pod != nil && e.pod.Original().DeletionTimestamp != nil {
		return e.pod.Original().DeletionTimestamp.Time
	}
	if !e.jobEvents.PodDeletionTimestamp().IsZero() {
		return e.jobEvents.PodDeletionTimestamp()
	}
	return time.Time{}
}

func (e *executionState) CompletionTimestamp() time.Time {
	if e.pod != nil && !e.pod.FinishTimestamp().IsZero() {
		return e.pod.FinishTimestamp()
	}
	if !e.PodDeletionTimestamp().IsZero() {
		return e.PodDeletionTimestamp()
	}
	if e.job != nil && !e.job.FinishTimestamp().IsZero() {
		ts := e.job.FinishTimestamp()
		// It may be not accurate, so try to take it from the events too
		if ts.Before(e.PodEvents().LastTimestamp()) {
			return e.PodEvents().LastTimestamp()
		}
		return ts
	}
	if !e.podEvents.FinishTimestamp().IsZero() {
		return e.podEvents.FinishTimestamp()
	}
	if !e.jobEvents.FinishTimestamp().IsZero() {
		return e.jobEvents.FinishTimestamp()
	}
	return time.Time{}
}

func (e *executionState) ContainerStartTimestamp(name string) time.Time {
	if e.pod != nil && !e.pod.ContainerStartTimestamp(name).IsZero() {
		return e.pod.ContainerStartTimestamp(name)
	}
	return e.podEvents.Container(name).StartTimestamp()
}

func (e *executionState) Namespace() string {
	if e.options.Namespace != "" {
		return e.options.Namespace
	}
	if e.job != nil && e.job.Namespace() != "" {
		return e.job.Namespace()
	}
	if e.pod != nil && e.pod.Namespace() != "" {
		return e.pod.Namespace()
	}
	if e.jobEvents.Namespace() != "" {
		return e.jobEvents.Namespace()
	}
	if e.podEvents.Namespace() != "" {
		return e.podEvents.Namespace()
	}
	return ""
}

func (e *executionState) ResourceId() string {
	if e.options.ResourceId != "" {
		return e.options.ResourceId
	}
	if e.job != nil && e.job.ResourceId() != "" {
		return e.job.ResourceId()
	}
	if e.pod != nil && e.pod.ResourceId() != "" {
		return e.pod.ResourceId()
	}
	// Fallback computing that from the pod name
	if e.PodName() != "" {
		return e.PodName()[0:strings.LastIndex(e.PodName(), "-")]
	}
	return ""
}

func (e *executionState) RootResourceId() string {
	if e.options.ResourceId != "" {
		return e.options.ResourceId
	}
	if e.job != nil && e.job.RootResourceId() != "" {
		return e.job.RootResourceId()
	}
	if e.pod != nil && e.pod.RootResourceId() != "" {
		return e.pod.RootResourceId()
	}
	// Fallback computing that from the pod name
	if e.PodName() != "" {
		return e.PodName()[0:strings.Index(e.PodName(), "-")]
	}
	return ""
}

func (e *executionState) RunnerId() string {
	if e.job != nil && e.job.RunnerId() != "" {
		return e.job.RunnerId()
	}
	if e.pod != nil && e.pod.RunnerId() != "" {
		return e.pod.RunnerId()
	}
	return ""
}

func (e *executionState) JobEvents() JobEvents {
	return e.jobEvents
}

func (e *executionState) PodEvents() PodEvents {
	return e.podEvents
}

func (e *executionState) Signature() ([]stage.Signature, error) {
	if e.job != nil {
		return e.job.Signature()
	}
	if e.pod != nil {
		return e.pod.Signature()
	}
	if e.options.Signature != nil {
		return e.options.Signature, nil
	}
	return nil, ErrMissingData
}

func (e *executionState) InternalConfig() (testworkflowconfig.InternalConfig, error) {
	if e.job != nil {
		return e.job.InternalConfig()
	}
	if e.pod != nil {
		return e.pod.InternalConfig()
	}
	return testworkflowconfig.InternalConfig{}, ErrMissingData
}

func (e *executionState) ScheduledAt() time.Time {
	if e.job != nil {
		v, err := e.job.ScheduledAt()
		if err == nil {
			return v
		}
	}
	if e.pod != nil {
		v, err := e.pod.ScheduledAt()
		if err == nil {
			return v
		}
	}
	return e.options.ScheduledAt
}

// InitializationTimeout returns the initialization timeout from the job, then from the pod, or zero.
func (e *executionState) InitializationTimeout() time.Duration {
	if e.job != nil {
		if timeout := e.job.InitializationTimeout(); timeout > 0 {
			return timeout
		}
	}
	if e.pod != nil {
		return e.pod.InitializationTimeout()
	}
	return 0
}

func (e *executionState) ActionGroups() (actiontypes.ActionGroups, error) {
	if e.job != nil {
		return e.job.ActionGroups()
	}
	if e.pod != nil {
		return e.pod.ActionGroups()
	}
	if e.options.ActionGroups != nil {
		return e.options.ActionGroups, nil
	}
	return nil, ErrMissingData
}

func (e *executionState) ContainerStarted(name string) bool {
	return (e.pod != nil && e.pod.ContainerStarted(name)) ||
		(e.podEvents.Container(name).Started())
}

func (e *executionState) ContainerFinished(name string) bool {
	return e.pod != nil && e.pod.ContainerFinished(name)
}

func (e *executionState) ContainerFailed(name string) bool {
	return e.pod != nil && e.pod.ContainerFailed(name)
}

func (e *executionState) JobCreationTimestamp() time.Time {
	if e.job != nil {
		return e.job.CreationTimestamp()
	}
	return time.Time{}
}

func (e *executionState) EstimatedJobCreationTimestamp() time.Time {
	if e.job != nil {
		return e.job.CreationTimestamp()
	}
	if !e.ScheduledAt().IsZero() {
		return e.ScheduledAt()
	}
	ts := e.jobEvents.FirstTimestamp()
	if e.podEvents.FirstTimestamp().Before(ts) {
		ts = e.podEvents.FirstTimestamp()
	}
	if !e.EstimatedPodCreationTimestamp().IsZero() && e.EstimatedPodCreationTimestamp().Before(ts) {
		ts = e.EstimatedPodCreationTimestamp()
	}
	return ts
}

func (e *executionState) PodCreationTimestamp() time.Time {
	if e.pod != nil {
		return e.pod.CreationTimestamp()
	}
	if !e.jobEvents.PodCreationTimestamp().IsZero() {
		return e.jobEvents.PodCreationTimestamp()
	}
	return time.Time{}
}

func (e *executionState) EstimatedPodCreationTimestamp() time.Time {
	if !e.PodCreationTimestamp().IsZero() {
		return e.PodCreationTimestamp()
	}
	if !e.podEvents.FirstTimestamp().IsZero() {
		return e.podEvents.FirstTimestamp()
	}
	return time.Time{}
}

func (e *executionState) PodStartTimestamp() time.Time {
	if e.pod != nil && !e.pod.StartTimestamp().IsZero() {
		return e.pod.StartTimestamp()
	}
	return e.podEvents.StartTimestamp()
}

func (e *executionState) EstimatedPodStartTimestamp() time.Time {
	if !e.PodStartTimestamp().IsZero() {
		return e.PodStartTimestamp()
	}
	for _, ev := range e.podEvents.Original() {
		if GetEventContainerName(ev) != "" {
			return GetFirstEventTimestamp(ev)
		}
	}
	return time.Time{}
}

func (e *executionState) PodCreated() bool {
	return !e.EstimatedPodCreationTimestamp().IsZero()
}

func (e *executionState) PodStarted() bool {
	return !e.EstimatedPodStartTimestamp().IsZero()
}

func (e *executionState) Completed() bool {
	return !e.CompletionTimestamp().IsZero()
}

func (e *executionState) JobExecutionError() string {
	if e.job != nil && e.job.ExecutionError() != "" {
		return e.job.ExecutionError()
	}

	if e.jobEvents.Error() {
		reason := e.jobEvents.ErrorReason()
		message := e.jobEvents.ErrorMessage()
		if message == "" {
			return reason
		}
		return fmt.Sprintf("%s: %s", reason, message)
	}

	return ""
}

func (e *executionState) PodExecutionError() string {
	errorStr := ""
	if e.pod != nil && e.pod.ExecutionError() != "" {
		errorStr = e.pod.ExecutionError()
	}

	if (errorStr == "" || errorStr == "Error") && e.podEvents.Error() {
		reason := e.podEvents.ErrorReason()
		message := e.podEvents.ErrorMessage()
		if message == "" {
			return reason
		}
		return fmt.Sprintf("%s: %s", reason, message)
	}

	if errorStr == "Error" {
		return "Fatal Error"
	}

	return errorStr
}

// imagePullWaitingReasons are the waiting reasons of a container whose image Kubernetes cannot pull.
var imagePullWaitingReasons = []string{"ErrImagePull", "ImagePullBackOff", "InvalidImageName"}

// CurrentCause returns the cause that keeps the pod from running, or nil when there is none.
// Kubernetes retries these causes, so the cause does not stop the execution.
// It does not use PodStarted, because Kubernetes sets the pod start time before it pulls the images.
func (e *executionState) CurrentCause() *testkube.Cause {
	if e.pod == nil {
		// The job reports FailedCreate before a pod exists.
		return e.jobEvents.WaitingCause()
	}
	if e.pod.Finished() {
		return nil
	}
	if message, ok := e.pod.Unschedulable(); ok {
		return &testkube.Cause{Reason: string(testkube.StopReasonUnschedulable), Message: message}
	}
	if reason, message := e.pod.WaitingReason(imagePullWaitingReasons...); reason != "" {
		return &testkube.Cause{Reason: string(testkube.StartReasonImagePullFailed), Message: message}
	}
	if reason, message := e.pod.WaitingReason("CreateContainerConfigError"); reason != "" {
		return &testkube.Cause{Reason: string(testkube.StopReasonConfigMissing), Message: message}
	}
	// A pod exists, so the job created it. An earlier FailedCreate event of the job does not apply.
	return e.podEvents.WaitingCause()
}

const (
	// containerReasonOOMKilled is the container reason for a kill on the memory limit.
	containerReasonOOMKilled = "OOMKilled"
	// The event reasons of a pod and a job that report the end of the pod.
	eventReasonEvicted             = "Evicted"
	eventReasonExceededGracePeriod = "ExceededGracePeriod"
	eventReasonBackoffLimit        = "BackoffLimitExceeded"
)

var (
	// containerErrorReasons are the container reasons that report a container that could not run.
	containerErrorReasons = []string{"Error", "StartError", "ContainerCannotRun"}
	// disruptionCauses maps the reason of the disruption condition of a pod to a reason code.
	disruptionCauses = map[string]testkube.StopReason{
		"EvictionByEvictionAPI":  testkube.StopReasonEvicted,
		"PreemptionByScheduler":  testkube.StopReasonPreempted,
		"TerminationByKubelet":   testkube.StopReasonNodeShutdown,
		"DeletionByTaintManager": testkube.StopReasonNodeShutdown,
		"DeletionByPodGC":        testkube.StopReasonNodeShutdown,
	}
)

// TerminationCause returns the cause that Kubernetes reported when it ended the pod, and nil when
// it reported none. It is the twin of CurrentCause, which reads a pod that still waits.
// The message is the text that ExecutionError returns, so the words of the result do not change.
func (e *executionState) TerminationCause() *testkube.Cause {
	reason := e.terminationReason()
	if reason == "" {
		return nil
	}
	return &testkube.Cause{Reason: string(reason), Message: e.ExecutionError()}
}

// terminationReason returns the code for the signal that ended the pod. The order of the checks
// decides which signal wins when Kubernetes reports more than one.
func (e *executionState) terminationReason() testkube.StopReason {
	var containerReason, disruption string
	// The condition of the job and the event of the job both report the deadline, and the watch can
	// read one before the other. The message reads the condition, so the code reads it too.
	deadline := e.job != nil && IsJobDeadlineExceeded(e.job.Original())
	if e.pod != nil {
		original := e.pod.Original()
		containerReason = e.pod.ContainerError()
		disruption, _ = GetPodDisruption(original)
		deadline = deadline || (original != nil && original.Status.Reason == ReasonDeadlineExceeded)
	}
	podEventReason := e.podEvents.ErrorReason()
	jobEventReason := e.jobEvents.ErrorReason()

	// A kill on the memory limit wins over a disruption of the pod, because the limit is what the
	// user raises. Kubernetes reports both when it takes a node away from a container that grew.
	if containerReason == containerReasonOOMKilled {
		return testkube.StopReasonOOMKilled
	}
	if cause, ok := disruptionCauses[disruption]; ok {
		return cause
	}
	if podEventReason == eventReasonEvicted {
		return testkube.StopReasonEvicted
	}
	// The deadline wins over a container error, because the container fails as a result of it.
	if deadline || jobEventReason == ReasonDeadlineExceeded {
		return testkube.StopReasonDeadlineExceeded
	}
	if slices.Contains(containerErrorReasons, containerReason) ||
		jobEventReason == eventReasonBackoffLimit || podEventReason == eventReasonExceededGracePeriod {
		return testkube.StopReasonContainerError
	}
	return ""
}

func (e *executionState) ExecutionError() string {
	podErr := e.PodExecutionError()
	jobErr := e.JobExecutionError()
	if podErr == "" && strings.HasPrefix(jobErr, "BackoffLimitExceeded") {
		return "Fatal Error"
	}
	if podErr == "" || (podErr == "Fatal Error" && jobErr != "" && !strings.HasPrefix(jobErr, "BackoffLimitExceeded")) {
		return jobErr
	}
	return podErr
}

func (e *executionState) Debug() map[string]string {
	result := map[string]string{
		"pod":       "unknown",
		"job":       "unknown",
		"podEvents": "unknown",
		"jobEvents": "unknown",
	}
	if e.pod != nil {
		result["pod"] = e.pod.Debug()
	}
	if e.job != nil {
		result["job"] = e.job.Debug()
	}
	if e.podEvents != nil {
		result["podEvents"] = e.podEvents.Debug()
	}
	if e.jobEvents != nil {
		result["jobEvents"] = e.jobEvents.Debug()
	}
	return result
}
