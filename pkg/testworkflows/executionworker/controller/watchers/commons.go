package watchers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	constants2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	DefaultErrorMessage = "Job has been aborted"

	defaultListTimeoutSeconds  = int64(240)
	defaultWatchTimeoutSeconds = int64(365 * 24 * 3600)
)

var (
	ErrDone                      = errors.New("resource is done")
	terminatedLogRe              = regexp.MustCompile(`^([^,]),(0|[1-9]\d*)$`)
	involvedFieldPathContainerRe = regexp.MustCompile(`^spec\.(?:initContainers|containers)\{([^]]+)}`)
)

type kubernetesClient[T any, U any] interface {
	List(ctx context.Context, options metav1.ListOptions) (*T, error)
	Watch(ctx context.Context, options metav1.ListOptions) (watch.Interface, error)
}

type readStart struct {
	count int
	err   error
}

func IsJobFinished(job *batchv1.Job) bool {
	if job == nil {
		return false
	}
	if job.DeletionTimestamp != nil {
		return true
	}
	if job.Status.CompletionTime != nil {
		return true
	}
	for i := range job.Status.Conditions {
		if job.Status.Conditions[i].Type == batchv1.JobComplete || job.Status.Conditions[i].Type == batchv1.JobFailed {
			if job.Status.Conditions[i].Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func IsPodFinished(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return true
	}
	if pod.DeletionTimestamp != nil {
		return true
	}
	if pod.Status.Phase == corev1.PodUnknown {
		for i := range pod.Status.Conditions {
			if pod.Status.Conditions[i].Type == corev1.DisruptionTarget && pod.Status.Conditions[i].Status == corev1.ConditionTrue {
				return true
			}
			if pod.Status.Conditions[i].Reason == "PodCompleted" {
				return true
			}
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		if pod.Status.InitContainerStatuses[i].State.Terminated != nil && ((pod.Status.InitContainerStatuses[i].State.Terminated.Reason != "Completed" && pod.Status.InitContainerStatuses[i].State.Terminated.Reason != "") || pod.Status.InitContainerStatuses[i].State.Terminated.ExitCode != 0) {
			return true
		}
	}
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].State.Terminated != nil && (pod.Status.ContainerStatuses[i].State.Terminated.Reason != "Completed" && pod.Status.ContainerStatuses[i].State.Terminated.Reason != "") {
			return true
		}
	}
	return false
}

func IsPodStarted(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.Status.Phase == corev1.PodRunning {
		return true
	}
	return false
}

func GetContainerStatus(pod *corev1.Pod, name string) *corev1.ContainerStatus {
	if pod == nil {
		return nil
	}
	for i := range pod.Status.InitContainerStatuses {
		if pod.Status.InitContainerStatuses[i].Name == name {
			return &pod.Status.InitContainerStatuses[i]
		}
	}
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == name {
			return &pod.Status.ContainerStatuses[i]
		}
	}
	return nil
}

func IsContainerStarted(pod *corev1.Pod, name string) bool {
	if pod == nil {
		return false
	}
	status := GetContainerStatus(pod, name)
	if status == nil {
		return false
	}
	return (status.Started != nil && *status.Started) || status.Ready || status.State.Running != nil
}

func IsContainerFinished(pod *corev1.Pod, name string) bool {
	status := GetContainerStatus(pod, name)
	return status != nil && status.State.Terminated != nil
}

func GetEventTimestamp(event *corev1.Event) time.Time {
	ts := event.CreationTimestamp.Time
	if event.FirstTimestamp.After(ts) {
		ts = event.FirstTimestamp.Time
	}
	if event.LastTimestamp.After(ts) {
		ts = event.LastTimestamp.Time
	}
	return ts
}

func GetFirstEventTimestamp(event *corev1.Event) time.Time {
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	if !event.CreationTimestamp.IsZero() {
		return event.CreationTimestamp.Time
	}
	return event.LastTimestamp.Time
}

func GetJobLastTimestamp(job *batchv1.Job) time.Time {
	if job.DeletionTimestamp != nil {
		return job.DeletionTimestamp.Time
	}
	ts := job.CreationTimestamp.Time
	if job.Status.CompletionTime != nil && job.Status.CompletionTime.After(ts) {
		ts = job.Status.CompletionTime.Time
	}
	for i := range job.Status.Conditions {
		if job.Status.Conditions[i].LastProbeTime.After(ts) {
			ts = job.Status.Conditions[i].LastProbeTime.Time
		}
		if job.Status.Conditions[i].LastTransitionTime.After(ts) {
			ts = job.Status.Conditions[i].LastTransitionTime.Time
		}
	}
	for i := range job.ManagedFields {
		if job.ManagedFields[i].Time != nil && job.ManagedFields[i].Time.After(ts) {
			ts = job.ManagedFields[i].Time.Time
		}
	}
	return ts
}

func GetJobCompletionTimestamp(job *batchv1.Job) time.Time {
	if job == nil {
		return time.Time{}
	}
	if job.Status.CompletionTime != nil {
		return job.Status.CompletionTime.Time
	}
	for i := range job.Status.Conditions {
		if job.Status.Conditions[i].Type == batchv1.JobComplete || job.Status.Conditions[i].Type == batchv1.JobFailed {
			if job.Status.Conditions[i].Status == corev1.ConditionTrue && !job.Status.Conditions[i].LastTransitionTime.IsZero() {
				return job.Status.Conditions[i].LastTransitionTime.Time
			}
		}
	}
	if job.DeletionTimestamp != nil {
		return job.DeletionTimestamp.Time
	}
	return time.Time{}
}

func GetPodLastTimestamp(pod *corev1.Pod) time.Time {
	if pod.DeletionTimestamp != nil {
		return pod.DeletionTimestamp.Time
	}
	ts := pod.CreationTimestamp.Time
	if pod.Status.StartTime != nil && pod.Status.StartTime.After(ts) {
		ts = pod.Status.StartTime.Time
	}
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].LastProbeTime.After(ts) {
			ts = pod.Status.Conditions[i].LastProbeTime.Time
		}
		if pod.Status.Conditions[i].LastTransitionTime.After(ts) {
			ts = pod.Status.Conditions[i].LastTransitionTime.Time
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		if pod.Status.InitContainerStatuses[i].State.Terminated != nil && pod.Status.InitContainerStatuses[i].State.Terminated.FinishedAt.After(ts) {
			ts = pod.Status.InitContainerStatuses[i].State.Terminated.FinishedAt.Time
		}
		if pod.Status.InitContainerStatuses[i].LastTerminationState.Terminated != nil && pod.Status.InitContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.After(ts) {
			ts = pod.Status.InitContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.Time
		}
	}
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].State.Terminated != nil && pod.Status.ContainerStatuses[i].State.Terminated.FinishedAt.After(ts) {
			ts = pod.Status.ContainerStatuses[i].State.Terminated.FinishedAt.Time
		}
		if pod.Status.ContainerStatuses[i].LastTerminationState.Terminated != nil && pod.Status.ContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.After(ts) {
			ts = pod.Status.ContainerStatuses[i].LastTerminationState.Terminated.FinishedAt.Time
		}
	}
	for i := range pod.ManagedFields {
		if pod.ManagedFields[i].Time != nil && pod.ManagedFields[i].Time.After(ts) {
			ts = pod.ManagedFields[i].Time.Time
		}
	}
	return ts
}

func GetPodCompletionTimestamp(pod *corev1.Pod) time.Time {
	if pod == nil {
		return time.Time{}
	}
	ts := time.Time{}
	for _, c := range pod.Status.Conditions {
		// TODO: Filter to only finished values
		if c.LastTransitionTime.After(ts) {
			ts = c.LastTransitionTime.Time
		}
	}
	if pod.DeletionTimestamp != nil && ts.IsZero() {
		ts = pod.DeletionTimestamp.Time
	} else if ts.IsZero() {
		// TODO: Consider getting the latest timestamp from the Pod object
		ts = time.Now()
	}

	return ts
}

type ContainerOperationStatus struct {
	ExitCode int
	Status   testkube.TestWorkflowStepStatus
}

type ContainerResult struct {
	Statuses     []ContainerOperationStatus
	ErrorDetails string
}

func GetPodError(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	if pod.Status.Reason == "DeadlineExceeded" && pod.Spec.ActiveDeadlineSeconds != nil {
		return fmt.Sprintf("Pod timed out after %d seconds", *pod.Spec.ActiveDeadlineSeconds)
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.DisruptionTarget && c.Status == corev1.ConditionTrue {
			if c.Message == "" {
				return c.Reason
			}
			return fmt.Sprintf("%s: %s", c.Reason, c.Message)
		}
	}
	return ""
}

func GetJobError(job *batchv1.Job) string {
	if job == nil {
		return ""
	}
	if job.Spec.ActiveDeadlineSeconds != nil {
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue && c.Reason == "DeadlineExceeded" {
				return fmt.Sprintf("Job timed out after %d seconds", *job.Spec.ActiveDeadlineSeconds)
			}
		}
	}
	var msg string
	if job.DeletionTimestamp != nil {
		msg = "Job has been aborted"
	}
	if job.Annotations != nil {
		if terminationReason, ok := job.Annotations["testkube.io/termination-reason"]; ok && terminationReason != "" {
			msg = terminationReason
		}
	}
	return msg
}

func GetTerminationCode(job *batchv1.Job) string {
	if job == nil || job.Annotations == nil {
		return string(testkube.ABORTED_TestWorkflowStatus)
	}
	if terminationCode, ok := job.Annotations[constants2.AnnotationTerminationCode]; ok && terminationCode != "" {
		return terminationCode
	}
	return string(testkube.ABORTED_TestWorkflowStatus)
}

func GetContainerStateDebug(state corev1.ContainerState) string {
	if state.Running != nil {
		return "running"
	} else if state.Terminated != nil {
		result := fmt.Sprintf("terminated, reason: '%s'", state.Terminated.Reason)
		if state.Terminated.Message != "" {
			result += fmt.Sprintf(", message: '%s'", state.Terminated.Message)
		}
		if state.Terminated.ExitCode != 0 {
			result += fmt.Sprintf(", exit code: '%d'", state.Terminated.ExitCode)
		}
		if state.Terminated.Signal != 0 {
			result += fmt.Sprintf(", signal: %d", state.Terminated.Signal)
		}
		return result
	} else if state.Waiting != nil {
		return fmt.Sprintf("waiting, reason: '%s', message: '%s'",
			state.Waiting.Reason,
			state.Waiting.Message)
	}
	return "unknown"
}

func ReadContainerResult(status *corev1.ContainerStatus, errorFallback string) ContainerResult {
	result := ContainerResult{}

	if status != nil && status.State.Terminated != nil {
		// Fetch the information about non-standard error
		if status.State.Terminated.Reason != "Completed" {
			result.ErrorDetails = status.State.Terminated.Reason
		}

		// Load status for all operations
		for _, message := range strings.Split(status.State.Terminated.Message, "/") {
			match := terminatedLogRe.FindStringSubmatch(message)

			// Stop parsing in case of invalid aborted message
			if match == nil {
				break
			}

			// Gather information
			stepStatus := testkube.TestWorkflowStepStatus(constants.StepStatusFromCode(match[1]))
			exitCode, _ := strconv.Atoi(match[2])

			// Don't trust after there is `aborted` status detected
			if stepStatus == testkube.ABORTED_TestWorkflowStepStatus {
				break
			}

			// Save the status
			result.Statuses = append(result.Statuses, ContainerOperationStatus{
				ExitCode: exitCode,
				Status:   stepStatus,
			})
		}
	}

	// Obtain non-standard error from the other resources too
	if (result.ErrorDetails == "" || result.ErrorDetails == "Error") && errorFallback != "" {
		result.ErrorDetails = errorFallback
	}

	// Re-label generic error
	if result.ErrorDetails == "Error" {
		result.ErrorDetails = "Fatal Error"
	}

	return result
}

func GetEventContainerName(event *corev1.Event) string {
	path := event.InvolvedObject.FieldPath
	if involvedFieldPathContainerRe.Match([]byte(path)) {
		name := involvedFieldPathContainerRe.ReplaceAllString(event.InvolvedObject.FieldPath, "$1")
		return name
	}
	return ""
}
