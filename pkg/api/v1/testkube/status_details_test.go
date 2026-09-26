package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusDetailsTypeOf(t *testing.T) {
	tests := []struct {
		name   string
		actor  StopActor
		reason string
		want   StatusDetailsType
	}{
		{name: "definition invalid", reason: string(StartReasonDefinitionInvalid), want: StatusDetailsTypeInitFailure},
		{name: "template missing", reason: string(StopReasonTemplateMissing), want: StatusDetailsTypeInitFailure},
		{name: "queue limit exceeded", reason: string(StopReasonQueueLimitExceeded), want: StatusDetailsTypeInitFailure},
		{name: "image pull failed", reason: string(StartReasonImagePullFailed), want: StatusDetailsTypeInitFailure},
		{name: "job create failed", reason: string(StartReasonJobCreateFailed), want: StatusDetailsTypeInitFailure},
		{name: "resource create failed", reason: string(StartReasonResourceFailed), want: StatusDetailsTypeInitFailure},
		{name: "start failed", reason: string(StartReasonUnknown), want: StatusDetailsTypeInitFailure},
		{name: "admission denied", reason: string(StopReasonAdmissionDenied), want: StatusDetailsTypeInitFailure},
		{name: "config missing", reason: string(StopReasonConfigMissing), want: StatusDetailsTypeInitFailure},
		{name: "unschedulable", reason: string(StopReasonUnschedulable), want: StatusDetailsTypeInitFailure},
		{name: "volume mount failed", reason: string(StopReasonVolumeMountFailed), want: StatusDetailsTypeInitFailure},
		{name: "initialization timeout", reason: string(StopReasonInitTimeout), want: StatusDetailsTypeInitFailure},
		{name: "git auth failed", reason: string(StopReasonGitAuthFailed), want: StatusDetailsTypeInitFailure},
		{name: "git clone failed", reason: string(StopReasonGitCloneFailed), want: StatusDetailsTypeInitFailure},

		{name: "evicted", reason: string(StopReasonEvicted), want: StatusDetailsTypeExecutionFailure},
		{name: "preempted", reason: string(StopReasonPreempted), want: StatusDetailsTypeExecutionFailure},
		{name: "node shutdown", reason: string(StopReasonNodeShutdown), want: StatusDetailsTypeExecutionFailure},
		{name: "out of memory kill", reason: string(StopReasonOOMKilled), want: StatusDetailsTypeExecutionFailure},
		{name: "process killed", reason: string(StopReasonProcessKilled), want: StatusDetailsTypeExecutionFailure},
		{name: "container error", reason: string(StopReasonContainerError), want: StatusDetailsTypeExecutionFailure},
		{name: "deadline exceeded", reason: string(StopReasonDeadlineExceeded), want: StatusDetailsTypeExecutionFailure},
		{name: "step timeout", reason: string(StopReasonStepTimeout), want: StatusDetailsTypeExecutionFailure},
		{name: "service not ready", reason: string(StopReasonServiceNotReady), want: StatusDetailsTypeExecutionFailure},
		{name: "artifact upload failed", reason: string(StopReasonArtifactUploadFailed), want: StatusDetailsTypeExecutionFailure},
		{name: "execution stuck", reason: string(StopReasonExecutionStuck), want: StatusDetailsTypeExecutionFailure},
		{name: "execution timeout", reason: string(StopReasonExecutionTimeout), want: StatusDetailsTypeExecutionFailure},
		{name: "transition timeout", reason: string(StopReasonTransitionTimeout), want: StatusDetailsTypeExecutionFailure},
		{name: "stop not confirmed", reason: string(StopReasonStopNotConfirmed), want: StatusDetailsTypeExecutionFailure},
		{name: "queue timeout", reason: string(StopReasonQueueTimeout), want: StatusDetailsTypeExecutionFailure},
		{name: "queued too long", reason: string(StopReasonQueuedTooLong), want: StatusDetailsTypeExecutionFailure},
		{name: "worker resume failed", reason: string(StopReasonWorkerResumeFailed), want: StatusDetailsTypeExecutionFailure},
		{name: "superseded", reason: string(StopReasonSuperseded), want: StatusDetailsTypeExecutionFailure},
		{name: "fail fast", reason: string(StopReasonFailFast), want: StatusDetailsTypeExecutionFailure},
		{name: "trigger abort", reason: string(StopReasonTriggerAbort), want: StatusDetailsTypeExecutionFailure},
		{name: "job deleted", reason: string(StopReasonJobDeleted), want: StatusDetailsTypeExecutionFailure},
		{name: "abort all without a person", reason: string(StopReasonAbortAll), want: StatusDetailsTypeExecutionFailure},

		{name: "exit code", reason: string(StopReasonExitCode), want: StatusDetailsTypeStepFailure},
		{name: "child workflow failed", reason: string(StopReasonChildWorkflowFailed), want: StatusDetailsTypeStepFailure},

		{name: "user cancel", reason: string(StopReasonUserCancel), want: StatusDetailsTypeUserCancel},
		{name: "force cancel", reason: string(StopReasonForceCancel), want: StatusDetailsTypeUserCancel},

		{name: "unknown", reason: string(StopReasonUnknown), want: StatusDetailsTypeUnknown},
		{name: "code from a newer control plane", reason: "later-added", want: StatusDetailsTypeUnknown},
		{name: "no reason at all", reason: "", want: StatusDetailsTypeUnknown},

		{name: "a person who stops all executions still cancels", actor: StopActorUser, reason: string(StopReasonAbortAll), want: StatusDetailsTypeUserCancel},
		{name: "a person who cancels an unschedulable execution still cancels", actor: StopActorUser, reason: string(StopReasonUnschedulable), want: StatusDetailsTypeUserCancel},
		{name: "the abort endpoint acts for a person", actor: StopActorAPI, reason: string(StopReasonAbortAll), want: StatusDetailsTypeUserCancel},
		{name: "the control plane is not a person", actor: StopActorControlPlane, reason: string(StopReasonQueueTimeout), want: StatusDetailsTypeExecutionFailure},
		{name: "the runner is not a person", actor: StopActorRunner, reason: string(StopReasonExecutionStuck), want: StatusDetailsTypeExecutionFailure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StatusDetailsTypeOf(tt.actor, tt.reason))
		})
	}
}

func TestStatusDetailsTypes_Sentence(t *testing.T) {
	// A code that reaches a user without words reads as a raw token, so every code in the
	// table has a sentence in one of the two reason types.
	for reason := range statusDetailsTypes {
		t.Run(reason, func(t *testing.T) {
			words := StopReason(reason).Sentence()
			if words == "" {
				words = StartReason(reason).Sentence()
			}
			assert.NotEmpty(t, words)
		})
	}
}

func TestNewStatusDetails(t *testing.T) {
	tests := []struct {
		name    string
		actor   StopActor
		reason  string
		step    string
		message string
		want    TestWorkflowStatusDetails
	}{
		{
			name:    "a cause that Kubernetes reported",
			reason:  string(StopReasonUnschedulable),
			step:    "rinit",
			message: "no node can run the pod: 0/2 nodes are available",
			want: TestWorkflowStatusDetails{
				Type_:   string(StatusDetailsTypeInitFailure),
				Reason:  string(StopReasonUnschedulable),
				Message: "no node can run the pod: 0/2 nodes are available",
				Step:    "rinit",
			},
		},
		{
			name:   "a stop that the control plane decided",
			actor:  StopActorControlPlane,
			reason: string(StopReasonQueueTimeout),
			want: TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeExecutionFailure),
				Reason: string(StopReasonQueueTimeout),
				Actor:  string(StopActorControlPlane),
			},
		},
		{
			name:  "a cancel by a person without a reason",
			actor: StopActorUser,
			want: TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeUserCancel),
				Reason: string(StopReasonUserCancel),
				Actor:  string(StopActorUser),
			},
		},
		{
			name:  "a cancel through the API without a reason",
			actor: StopActorAPI,
			want: TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeUserCancel),
				Reason: string(StopReasonUserCancel),
				Actor:  string(StopActorAPI),
			},
		},
		{
			name:  "a control plane stop without a reason",
			actor: StopActorControlPlane,
			want: TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeUnknown),
				Reason: string(StopReasonUnknown),
				Actor:  string(StopActorControlPlane),
			},
		},
		{
			name: "a stop without an actor and without a reason",
			want: TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeUnknown),
				Reason: string(StopReasonUnknown),
			},
		},
		{
			name:   "a cancel by a person keeps the reason it carries",
			actor:  StopActorUser,
			reason: string(StopReasonAbortAll),
			want: TestWorkflowStatusDetails{
				Type_:  string(StatusDetailsTypeUserCancel),
				Reason: string(StopReasonAbortAll),
				Actor:  string(StopActorUser),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, *NewStatusDetails(tt.actor, tt.reason, tt.step, tt.message))
		})
	}
}

func TestStatusDetailsType_DisplayName(t *testing.T) {
	tests := []struct {
		name string
		t    StatusDetailsType
		want string
	}{
		{name: "a known type has its words", t: StatusDetailsTypeExecutionFailure, want: "Infrastructure failure"},
		{name: "a type without words stays the type", t: "a-newer-type", want: "a-newer-type"},
		{name: "an empty type stays empty", t: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.t.DisplayName())
		})
	}
}

func TestStatusDetailsType_DisplayName_EveryType(t *testing.T) {
	// A type without words reads as a raw token, so every known type has words.
	for _, detailsType := range []StatusDetailsType{
		StatusDetailsTypeInitFailure, StatusDetailsTypeExecutionFailure, StatusDetailsTypeStepFailure,
		StatusDetailsTypeUserCancel, StatusDetailsTypeUnknown,
	} {
		assert.NotEqual(t, string(detailsType), detailsType.DisplayName(), detailsType)
	}
}
