package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/common"
)

func TestTestWorkflowResult_ClassifyStatus(t *testing.T) {
	aborted := string(ABORTED_TestWorkflowStatus)
	canceled := string(CANCELED_TestWorkflowStatus)
	step := func(status TestWorkflowStepStatus, message, reason string) TestWorkflowStepResult {
		return TestWorkflowStepResult{Status: common.Ptr(status), ErrorMessage: message, ErrorReason: reason}
	}
	details := func(detailsType StatusDetailsType, reason, ref, message string, actor StopActor) *TestWorkflowStatusDetails {
		return &TestWorkflowStatusDetails{
			Type_:   string(detailsType),
			Reason:  reason,
			Message: message,
			Step:    ref,
			Actor:   string(actor),
		}
	}

	tests := []struct {
		name           string
		nilResult      bool
		status         TestWorkflowStatus
		initialization TestWorkflowStepResult
		steps          map[string]TestWorkflowStepResult
		sigSequence    []TestWorkflowSignature
		stop           Stop
		want           *TestWorkflowStatusDetails
	}{
		{
			name:      "a nil result carries no object",
			nilResult: true,
			want:      nil,
		},
		{
			name:           "an execution that passed carries no object",
			status:         PASSED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps:          map[string]TestWorkflowStepResult{"a": step(PASSED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			want:           nil,
		},
		{
			name:           "a cancel by a person carries no reason of its own",
			status:         CANCELED_TestWorkflowStatus,
			initialization: step(CANCELED_TestWorkflowStepStatus, "The execution has been canceled. (by the user)", ""),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop:           Stop{Code: canceled, Actor: StopActorUser},
			want: details(StatusDetailsTypeUserCancel, string(StopReasonUserCancel), "",
				"The execution has been canceled. (by the user)", StopActorUser),
		},
		{
			name:           "a person who cancels an unschedulable execution still cancels",
			status:         CANCELED_TestWorkflowStatus,
			initialization: step(CANCELED_TestWorkflowStepStatus, "The execution has been canceled. (by the user: no node can run the pod)", string(StopReasonUnschedulable)),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop:           Stop{Code: canceled, Actor: StopActorUser},
			want: details(StatusDetailsTypeUserCancel, string(StopReasonUserCancel), "",
				"The execution has been canceled. (by the user: no node can run the pod)", StopActorUser),
		},
		{
			name:           "a person who stops all executions of a workflow still cancels",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the user)", ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorUser, Reason: StopReasonAbortAll},
			want: details(StatusDetailsTypeUserCancel, string(StopReasonAbortAll), "",
				"The execution has been aborted. (by the user)", StopActorUser),
		},
		{
			name:           "the abort endpoint of a standalone agent acts for a person",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (through the API)", ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorAPI},
			want: details(StatusDetailsTypeUserCancel, string(StopReasonUserCancel), "",
				"The execution has been aborted. (through the API)", StopActorAPI),
		},
		{
			name:           "a cause that the pod recorded wins over the reason of the stop",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (no node can run the pod)", string(StopReasonUnschedulable)),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop:           Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonInitTimeout},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonUnschedulable), "",
				"The execution has been aborted. (no node can run the pod)", ""),
		},
		{
			name:           "a code that a toolkit step reported names the step",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"clone": step(FAILED_TestWorkflowStepStatus, "fatal: could not read Username", string(StopReasonGitAuthFailed)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "clone"}},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonGitAuthFailed), "clone",
				"fatal: could not read Username", ""),
		},
		{
			name:           "a required step with a code wins over an optional step with a code",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"optional": step(FAILED_TestWorkflowStepStatus, "upload failed", string(StopReasonArtifactUploadFailed)),
				"required": step(FAILED_TestWorkflowStepStatus, "the service did not start", string(StopReasonServiceNotReady)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "optional", Optional: true}, {Ref: "required"}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonServiceNotReady), "required",
				"the service did not start", ""),
		},
		{
			name:           "an optional step with a code counts when no required step has one",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"optional": step(FAILED_TestWorkflowStepStatus, "upload failed", string(StopReasonArtifactUploadFailed)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "optional", Optional: true}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonArtifactUploadFailed), "optional",
				"upload failed", ""),
		},
		{
			name:           "a step that passed on a later attempt holds no cause",
			status:         PASSED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps:          map[string]TestWorkflowStepResult{"a": {Status: common.Ptr(PASSED_TestWorkflowStepStatus), Attempts: 3}},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			want:           nil,
		},
		{
			name:           "a stop that the control plane decided reads its reason",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the control plane: the execution ran for too long)", ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorControlPlane, Reason: StopReasonExecutionTimeout},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonExecutionTimeout), "",
				"The execution has been aborted. (by the control plane: the execution ran for too long)", StopActorControlPlane),
		},
		{
			name:           "a fail-fast actor without a reason reads fail-fast",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (because another parallel worker failed)", ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			stop:        Stop{Code: aborted, Actor: StopActorFailFast},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonFailFast), "a",
				"The execution has been aborted. (because another parallel worker failed)", StopActorFailFast),
		},
		{
			name:           "a trigger actor without a reason reads trigger-abort",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (because its trigger was deleted)", ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorTrigger},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonTriggerAbort), "",
				"The execution has been aborted. (because its trigger was deleted)", StopActorTrigger),
		},
		{
			name:           "a failed step reads exit-code and names the step",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(PASSED_TestWorkflowStepStatus, "", ""),
				"b": step(FAILED_TestWorkflowStepStatus, "", ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}, {Ref: "b"}},
			want:        details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "b", "", ""),
		},
		{
			name:           "an optional step that failed does not decide the result",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"optional": step(FAILED_TestWorkflowStepStatus, "", ""),
				"required": step(FAILED_TestWorkflowStepStatus, "", ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "optional", Optional: true}, {Ref: "required"}},
			want:        details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "required", "", ""),
		},
		{
			name:           "a deleted job with no annotation reads job-deleted",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the system)", ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorSystem},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonJobDeleted), "",
				"The execution has been aborted. (by the system)", StopActorSystem),
		},
		{
			name:           "a step that failed before the job was deleted keeps exit-code",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps:          map[string]TestWorkflowStepResult{"a": step(FAILED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop:           Stop{Code: aborted, Actor: StopActorSystem},
			want:           details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "a", "", ""),
		},
		{
			// The heal writes the code of the stop into the step it stops, so the classifier reads a
			// step that already holds it. The actor of the stop must survive that.
			name:           "an initialization timeout keeps the actor of the runner",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the runner: the first step did not start before the initialization timeout of the workflow)", string(StopReasonInitTimeout)),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonInitTimeout},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonInitTimeout), "",
				"The execution has been aborted. (by the runner: the first step did not start before the initialization timeout of the workflow)", StopActorRunner),
		},
		{
			name:           "a fail-fast stop keeps its actor after the heal wrote the code",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (because another parallel worker failed)", string(StopReasonFailFast)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			stop:        Stop{Code: aborted, Actor: StopActorFailFast},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonFailFast), "a",
				"The execution has been aborted. (because another parallel worker failed)", StopActorFailFast),
		},
		{
			name:           "a declined start keeps the actor of the runner",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "Failed to run execution: the image could not be pulled\npull access denied", string(StartReasonImagePullFailed)),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReason(StartReasonImagePullFailed)},
			want: details(StatusDetailsTypeInitFailure, string(StartReasonImagePullFailed), "",
				"Failed to run execution: the image could not be pulled\npull access denied", StopActorRunner),
		},
		{
			// A cause that differs from the reason of the stop is a cause of its own, so it wins and
			// it names no actor.
			name:           "a Kubernetes cause still wins over the reason of the stop",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the runner: no node can run the pod)", string(StopReasonUnschedulable)),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonInitTimeout},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonUnschedulable), "",
				"The execution has been aborted. (by the runner: no node can run the pod)", ""),
		},
		{
			name:           "no signal reads unknown with the message of the initialization step",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted.", ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted},
			want: details(StatusDetailsTypeUnknown, string(StopReasonUnknown), "",
				"The execution has been aborted.", ""),
		},
		{
			name:           "a step that the stop skipped does not carry the cause",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, "The execution has been aborted. (by the runner)", ""),
				"b": step(SKIPPED_TestWorkflowStepStatus, "The execution was aborted before. (by the runner)", ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}, {Ref: "b"}},
			stop:        Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonExecutionStuck},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonExecutionStuck), "a",
				"The execution has been aborted. (by the runner)", StopActorRunner),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *TestWorkflowResult
			if !tt.nilResult {
				result = &TestWorkflowResult{
					Status:         common.Ptr(tt.status),
					Initialization: common.Ptr(tt.initialization),
					Steps:          tt.steps,
				}
			}

			assert.Equal(t, tt.want, result.ClassifyStatus(tt.sigSequence, tt.stop))
		})
	}
}
