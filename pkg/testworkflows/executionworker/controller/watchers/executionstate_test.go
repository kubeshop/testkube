package watchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestExecutionState_CurrentCause(t *testing.T) {
	waiting := func(name, reason, message string) corev1.ContainerStatus {
		return corev1.ContainerStatus{Name: name, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}}}
	}
	unschedulable := corev1.PodCondition{
		Type:    corev1.PodScheduled,
		Status:  corev1.ConditionFalse,
		Reason:  corev1.PodReasonUnschedulable,
		Message: "0/1 nodes are available: 1 Insufficient cpu.",
	}

	warning := func(reason, message string) *corev1.Event {
		return &corev1.Event{Type: corev1.EventTypeWarning, Reason: reason, Message: message}
	}
	normal := func(reason string) *corev1.Event {
		return &corev1.Event{Type: corev1.EventTypeNormal, Reason: reason}
	}
	at := func(event *corev1.Event, second int) *corev1.Event {
		event.LastTimestamp = metav1.Time{Time: time.Date(2026, 1, 1, 10, 0, second, 0, time.UTC)}
		return event
	}
	pending := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}

	tests := []struct {
		name      string
		pod       *corev1.Pod
		podEvents []*corev1.Event
		jobEvents []*corev1.Event
		want      *testkube.Cause
	}{
		{
			name: "returns nil without a pod",
			want: nil,
		},
		{
			name: "returns unschedulable with the scheduler message",
			pod:  &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, Conditions: []corev1.PodCondition{unschedulable}}},
			want: &testkube.Cause{Reason: "unschedulable", Message: "0/1 nodes are available: 1 Insufficient cpu."},
		},
		{
			name: "returns image-pull-failed for an init container",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, InitContainerStatuses: []corev1.ContainerStatus{
				waiting("tktw-init", "ErrImagePull", "pull access denied"),
			}}},
			want: &testkube.Cause{Reason: "image-pull-failed", Message: "pull access denied"},
		},
		{
			name: "returns config-missing for a container that waits for a secret",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{
				waiting("1", "CreateContainerConfigError", `secret "does-not-exist" not found`),
			}}},
			want: &testkube.Cause{Reason: "config-missing", Message: `secret "does-not-exist" not found`},
		},
		{
			name: "ignores a container that waits for its normal start",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{
				waiting("1", "ContainerCreating", ""),
			}}},
			want: nil,
		},
		{
			name: "ignores a container that is not a step container",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{
				waiting("sidecar", "ImagePullBackOff", "Back-off pulling image"),
			}}},
			want: nil,
		},
		{
			name:      "returns volume-mount-failed from the latest mount warning",
			pod:       pending,
			podEvents: []*corev1.Event{normal("Scheduled"), warning("FailedMount", `secret "missing" not found`)},
			want:      &testkube.Cause{Reason: "volume-mount-failed", Message: `secret "missing" not found`},
		},
		{
			name:      "ignores a scheduling warning that a later progress event passed",
			pod:       pending,
			podEvents: []*corev1.Event{warning("FailedScheduling", "0/1 nodes are available"), normal("Scheduled")},
			want:      nil,
		},
		{
			name:      "keeps a newer failure when an older progress event comes later in the list",
			pod:       pending,
			podEvents: []*corev1.Event{at(warning("FailedScheduling", "0/1 nodes are available"), 20), at(normal("Scheduled"), 10)},
			want:      &testkube.Cause{Reason: "unschedulable", Message: "0/1 nodes are available"},
		},
		{
			name:      "ignores an older failure when a newer progress event comes earlier in the list",
			pod:       pending,
			podEvents: []*corev1.Event{at(normal("Scheduled"), 20), at(warning("FailedScheduling", "0/1 nodes are available"), 10)},
			want:      nil,
		},
		{
			name:      "ignores an event with a cause reason that is not a warning",
			pod:       pending,
			podEvents: []*corev1.Event{normal("FailedMount")},
			want:      nil,
		},
		{
			name:      "returns admission-denied from the job before the pod exists",
			jobEvents: []*corev1.Event{warning("FailedCreate", `admission webhook "policy" denied the request: privileged containers are not allowed`)},
			want:      &testkube.Cause{Reason: "admission-denied", Message: `admission webhook "policy" denied the request: privileged containers are not allowed`},
		},
		{
			name:      "returns nil when the job created the pod after an earlier create warning, before the pod shows",
			jobEvents: []*corev1.Event{at(warning("FailedCreate", "denied"), 1), at(normal("SuccessfulCreate"), 2)},
			want:      nil,
		},
		{
			name:      "ignores a job create warning when the pod exists",
			pod:       pending,
			jobEvents: []*corev1.Event{warning("FailedCreate", "denied")},
			want:      nil,
		},
		{
			name: "returns nil for a finished pod",
			pod:  &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed, Conditions: []corev1.PodCondition{unschedulable}}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pod Pod
			if tt.pod != nil {
				pod = NewPod(tt.pod)
			}
			state := NewExecutionState(nil, pod, NewJobEvents(tt.jobEvents), NewPodEvents(tt.podEvents), nil)

			assert.Equal(t, tt.want, state.CurrentCause())
		})
	}
}

func TestExecutionState_InitializationTimeout(t *testing.T) {
	annotated := func(value string) map[string]string {
		return map[string]string{constants.InitializationTimeoutAnnotation: value}
	}

	tests := []struct {
		name           string
		jobAnnotations map[string]string
		podAnnotations map[string]string
		want           time.Duration
	}{
		{
			name:           "reads the job first",
			jobAnnotations: annotated("2m0s"),
			podAnnotations: annotated("3m0s"),
			want:           2 * time.Minute,
		},
		{
			name:           "reads the pod when the job has no value",
			podAnnotations: annotated("3m0s"),
			want:           3 * time.Minute,
		},
		{
			name: "returns zero without an annotation",
			want: 0,
		},
		{
			name:           "returns zero for a value that is not a duration",
			jobAnnotations: annotated("soon"),
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &batchv1.Job{Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: tt.jobAnnotations}}}}
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: tt.podAnnotations}}
			state := NewExecutionState(NewJob(job), NewPod(pod), NewJobEvents(nil), NewPodEvents(nil), nil)

			assert.Equal(t, tt.want, state.InitializationTimeout())
		})
	}
}

func TestExecutionState_TerminationCause(t *testing.T) {
	terminated := func(name, reason string) corev1.ContainerStatus {
		return corev1.ContainerStatus{
			Name:  name,
			State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: reason}},
		}
	}
	// The step containers carry a numeric name, so the reader tells them from a sidecar.
	stepContainer := func(reason string) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{
			Phase:             corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{terminated("1", reason)},
		}}
	}
	disrupted := func(reason, message string) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			Conditions: []corev1.PodCondition{{
				Type:    corev1.DisruptionTarget,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: message,
			}},
		}}
	}
	warning := func(reason, message string) *corev1.Event {
		return &corev1.Event{Type: corev1.EventTypeWarning, Reason: reason, Message: message}
	}

	deadlineJob := &batchv1.Job{
		Spec: batchv1.JobSpec{ActiveDeadlineSeconds: common.Ptr(int64(60))},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
			Type:   batchv1.JobFailed,
			Status: corev1.ConditionTrue,
			Reason: "DeadlineExceeded",
		}}},
	}

	tests := []struct {
		name      string
		job       *batchv1.Job
		pod       *corev1.Pod
		podEvents []*corev1.Event
		jobEvents []*corev1.Event
		want      *testkube.Cause
	}{
		{
			name: "reports nothing when Kubernetes reported no signal",
			pod:  &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
			want: nil,
		},
		{
			name: "reports an out-of-memory kill of a step container",
			pod:  stepContainer("OOMKilled"),
			want: &testkube.Cause{Reason: "oom-killed", Message: "OOMKilled"},
		},
		{
			name: "an out-of-memory kill wins over a disruption of the pod",
			pod: &corev1.Pod{Status: corev1.PodStatus{
				Phase:             corev1.PodFailed,
				ContainerStatuses: []corev1.ContainerStatus{terminated("1", "OOMKilled")},
				Conditions: []corev1.PodCondition{{
					Type:   corev1.DisruptionTarget,
					Status: corev1.ConditionTrue,
					Reason: "TerminationByKubelet",
				}},
			}},
			want: &testkube.Cause{Reason: "oom-killed", Message: "TerminationByKubelet"},
		},
		{
			name: "reports an eviction from the condition of the pod",
			pod:  disrupted("EvictionByEvictionAPI", "the node was low on memory"),
			want: &testkube.Cause{Reason: "evicted", Message: "EvictionByEvictionAPI: the node was low on memory"},
		},
		{
			name:      "reports an eviction from the event of the pod",
			pod:       &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			podEvents: []*corev1.Event{warning("Evicted", "The node was low on resource: ephemeral-storage")},
			want:      &testkube.Cause{Reason: "evicted", Message: "Evicted: The node was low on resource: ephemeral-storage"},
		},
		{
			name: "reports a preemption",
			pod:  disrupted("PreemptionByScheduler", "preempted for a pod with a higher priority"),
			want: &testkube.Cause{Reason: "preempted", Message: "PreemptionByScheduler: preempted for a pod with a higher priority"},
		},
		{
			name: "reports a shutdown of the node from the kubelet",
			pod:  disrupted("TerminationByKubelet", "the node is shutting down"),
			want: &testkube.Cause{Reason: "node-shutdown", Message: "TerminationByKubelet: the node is shutting down"},
		},
		{
			name: "reports a shutdown of the node from the taint manager",
			pod:  disrupted("DeletionByTaintManager", "the node has an unreachable taint"),
			want: &testkube.Cause{Reason: "node-shutdown", Message: "DeletionByTaintManager: the node has an unreachable taint"},
		},
		{
			name: "reports a shutdown of the node from the pod garbage collector",
			pod:  disrupted("DeletionByPodGC", "the node no longer exists"),
			want: &testkube.Cause{Reason: "node-shutdown", Message: "DeletionByPodGC: the node no longer exists"},
		},
		{
			name: "reports the deadline of the pod",
			pod: &corev1.Pod{
				Spec:   corev1.PodSpec{ActiveDeadlineSeconds: common.Ptr(int64(60))},
				Status: corev1.PodStatus{Phase: corev1.PodFailed, Reason: "DeadlineExceeded"},
			},
			want: &testkube.Cause{Reason: "deadline-exceeded", Message: "Pod timed out after 60 seconds"},
		},
		{
			name:      "reports the deadline of the job",
			pod:       &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			jobEvents: []*corev1.Event{warning("DeadlineExceeded", "Job was active longer than specified deadline")},
			want:      &testkube.Cause{Reason: "deadline-exceeded", Message: "DeadlineExceeded: Job was active longer than specified deadline"},
		},
		{
			name: "the deadline of the job wins over a container that could not run",
			pod:  stepContainer("Error"),
			// A container that the deadline stops reports its own error, so the deadline names the cause.
			jobEvents: []*corev1.Event{warning("DeadlineExceeded", "Job was active longer than specified deadline")},
			want:      &testkube.Cause{Reason: "deadline-exceeded", Message: "DeadlineExceeded: Job was active longer than specified deadline"},
		},
		{
			name: "reports a container that could not run",
			pod:  stepContainer("ContainerCannotRun"),
			want: &testkube.Cause{Reason: "container-error", Message: "ContainerCannotRun"},
		},
		{
			name: "reports a container that could not start",
			pod:  stepContainer("StartError"),
			want: &testkube.Cause{Reason: "container-error", Message: "StartError"},
		},
		{
			name:      "reports the backoff limit of the job",
			pod:       &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			jobEvents: []*corev1.Event{warning("BackoffLimitExceeded", "Job has reached the specified backoff limit")},
			want:      &testkube.Cause{Reason: "container-error", Message: "Fatal Error"},
		},
		{
			name:      "reports a pod that passed its grace period",
			pod:       &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			podEvents: []*corev1.Event{warning("ExceededGracePeriod", "Container runtime did not kill the pod within specified grace period")},
			want:      &testkube.Cause{Reason: "container-error", Message: "ExceededGracePeriod: Container runtime did not kill the pod within specified grace period"},
		},
		{
			name: "reports nothing without a pod",
			want: nil,
		},
		{
			// The watch can read the terminal update of the job before its event arrives, so the
			// condition alone must still give the code that the message names.
			name: "reports the deadline from the condition of the job without its event",
			job:  deadlineJob,
			pod:  &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			want: &testkube.Cause{Reason: "deadline-exceeded", Message: "Job timed out after 60 seconds"},
		},
		{
			name: "the condition of the job wins over a container that could not run",
			job:  deadlineJob,
			pod:  stepContainer("Error"),
			want: &testkube.Cause{Reason: "deadline-exceeded", Message: "Job timed out after 60 seconds"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pod Pod
			if tt.pod != nil {
				pod = NewPod(tt.pod)
			}
			var job Job
			if tt.job != nil {
				job = NewJob(tt.job)
			}
			state := NewExecutionState(job, pod, NewJobEvents(tt.jobEvents), NewPodEvents(tt.podEvents), nil)

			assert.Equal(t, tt.want, state.TerminationCause())
		})
	}
}
