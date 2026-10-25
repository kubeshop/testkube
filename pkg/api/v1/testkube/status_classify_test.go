package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// healed stands for the termination sentence that the heal writes. The classifier must not read
	// it for a step that the stop ended, because the runner gives the plain cause in stop.Causes.
	const healed = "the message that the heal wrote"
	fatal := termination{code: aborted, reason: "Fatal Error"}.message()

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
			initialization: step(CANCELED_TestWorkflowStepStatus, healed, ""),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop:           Stop{Code: canceled, Actor: StopActorUser, Causes: map[string]string{"": ""}},
			want:           details(StatusDetailsTypeUserCancel, string(StopReasonUserCancel), "", "", StopActorUser),
		},
		{
			name:           "a person who cancels an unschedulable execution still cancels",
			status:         CANCELED_TestWorkflowStatus,
			initialization: step(CANCELED_TestWorkflowStepStatus, healed, string(StopReasonUnschedulable)),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop:           Stop{Code: canceled, Actor: StopActorUser, Causes: map[string]string{"": "0/1 nodes are available"}},
			want: details(StatusDetailsTypeUserCancel, string(StopReasonUserCancel), "",
				"0/1 nodes are available", StopActorUser),
		},
		{
			name:           "a person who stops all executions of a workflow still cancels",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorUser, Reason: StopReasonAbortAll, Causes: map[string]string{"": ""}},
			want:           details(StatusDetailsTypeUserCancel, string(StopReasonAbortAll), "", "", StopActorUser),
		},
		{
			name:           "the abort endpoint of a standalone agent acts for a person",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorAPI, Causes: map[string]string{"": ""}},
			want:           details(StatusDetailsTypeUserCancel, string(StopReasonUserCancel), "", "", StopActorAPI),
		},
		{
			// A code that differs from the reason of the stop is a cause of its own, so it wins and
			// it names no actor.
			name:           "a cause that the pod recorded wins over the reason of the stop",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonUnschedulable)),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop: Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonInitTimeout, Causes: map[string]string{
				"": "the first step did not start before the initialization timeout of the workflow: 0/1 nodes are available",
			}},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonUnschedulable), "",
				"the first step did not start before the initialization timeout of the workflow: 0/1 nodes are available", ""),
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
			name:           "a step that ran out of memory names the step",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonOOMKilled)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a", Name: "Run test"}},
			stop:        Stop{Code: aborted, Causes: map[string]string{"a": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonOOMKilled), "a",
				`The step "Run test" ran out of memory.`, ""),
		},
		{
			name:           "a step timeout names the step in a group",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"leaf": step(ABORTED_TestWorkflowStepStatus, "the step did not finish within its timeout", string(StopReasonStepTimeout)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "group", Children: []TestWorkflowSignature{{Ref: "leaf", Name: "Run test"}}}, {Ref: "leaf", Name: "Run test"}},
			stop:        Stop{Code: aborted, Causes: map[string]string{"leaf": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonStepTimeout), "leaf",
				`The step "Run test" did not finish within its timeout.`, ""),
		},
		{
			name:           "a step message that the runner gives no plain cause for stays as it is",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"leaf": step(ABORTED_TestWorkflowStepStatus, "the step did not finish within its timeout", string(StopReasonStepTimeout)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "leaf", Name: "Run test"}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonStepTimeout), "leaf",
				"the step did not finish within its timeout", ""),
		},
		{
			name:           "an empty plain cause of a code without a phrase stays empty",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonVolumeMountFailed)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a", Name: "Run test"}},
			stop:        Stop{Code: aborted, Causes: map[string]string{"a": ""}},
			want:        details(StatusDetailsTypeInitFailure, string(StopReasonVolumeMountFailed), "a", "", ""),
		},
		{
			name:           "a step without a name falls back to its category",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonOOMKilled)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a", Category: "Run shell command"}},
			stop:        Stop{Code: aborted, Causes: map[string]string{"a": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonOOMKilled), "a",
				`The step "Run shell command" ran out of memory.`, ""),
		},
		{
			name:           "a cause with words of its own stays",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, "the test process was killed by the kernel (signal: killed)", string(StopReasonProcessKilled)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a", Name: "Run test"}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonProcessKilled), "a",
				"the test process was killed by the kernel (signal: killed)", ""),
		},
		{
			name:           "an out-of-memory kill without a step gives the words of the code",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonOOMKilled)),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a", Name: "Run test"}},
			stop:           Stop{Code: aborted, Causes: map[string]string{"": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonOOMKilled), "",
				"the container exceeded its memory limit", ""),
		},
		{
			name:           "a failed initialization step keeps its message",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(FAILED_TestWorkflowStepStatus, "the init process failed", ""),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			want: details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "",
				"the init process failed", ""),
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
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorControlPlane, Reason: StopReasonExecutionTimeout, Causes: map[string]string{"": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonExecutionTimeout), "",
				"", StopActorControlPlane),
		},
		{
			name:           "a fail-fast actor without a reason reads fail-fast",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			stop:        Stop{Code: aborted, Actor: StopActorFailFast, Causes: map[string]string{"a": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonFailFast), "a",
				"", StopActorFailFast),
		},
		{
			name:           "a trigger actor without a reason reads trigger-abort",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorTrigger, Causes: map[string]string{"": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonTriggerAbort), "",
				"", StopActorTrigger),
		},
		{
			name:           "a failed step reads exit-code and names the step",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(PASSED_TestWorkflowStepStatus, "", ""),
				"b": {Status: common.Ptr(FAILED_TestWorkflowStepStatus), ExitCode: 2},
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}, {Ref: "b", Name: "Run tests"}},
			want: details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "b",
				`The step "Run tests" exited with code 2.`, ""),
		},
		{
			// A negative group fails when its children pass, so the group is the only failure there
			// is. Skipping every group would report unknown and lose the test failure.
			name:           "a negative group that failed names the group",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"group": step(FAILED_TestWorkflowStepStatus, "", ""),
				"leaf":  step(PASSED_TestWorkflowStepStatus, "", ""),
			},
			sigSequence: []TestWorkflowSignature{
				{Ref: "group", Negative: true, Children: []TestWorkflowSignature{{Ref: "leaf"}}},
				{Ref: "leaf"},
			},
			want: details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "group",
				`The step "group" passed, but it must fail.`, ""),
		},
		{
			// A failed leaf names the failure more precisely than the group that holds it.
			name:           "a failed leaf wins over the group that also failed",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"group": step(FAILED_TestWorkflowStepStatus, "", ""),
				"leaf":  step(FAILED_TestWorkflowStepStatus, "exit status 3", ""),
			},
			sigSequence: []TestWorkflowSignature{
				{Ref: "group", Children: []TestWorkflowSignature{{Ref: "leaf"}}},
				{Ref: "leaf"},
			},
			want: details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "leaf", "exit status 3", ""),
		},
		{
			name:           "an optional step that failed does not decide the result",
			status:         FAILED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"optional": step(FAILED_TestWorkflowStepStatus, "", ""),
				"required": step(FAILED_TestWorkflowStepStatus, "", ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "optional", Optional: true}, {Ref: "required", Category: "Run shell command"}},
			want: details(StatusDetailsTypeStepFailure, string(StopReasonExitCode), "required",
				`The step "Run shell command" failed.`, ""),
		},
		{
			// The heal writes the code of the stop into the step it stops, so the classifier reads a
			// step that already holds it. The actor of the stop must survive that.
			name:           "an initialization timeout keeps the actor of the runner",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonInitTimeout)),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonInitTimeout, Causes: map[string]string{"": ""}},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonInitTimeout), "",
				"", StopActorRunner),
		},
		{
			name:           "a fail-fast stop keeps its actor after the heal wrote the code",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonFailFast)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			stop:        Stop{Code: aborted, Actor: StopActorFailFast, Causes: map[string]string{"a": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonFailFast), "a",
				"", StopActorFailFast),
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
			// The signature puts a group before its children, and the heal can stop both. The leaf
			// holds the message and the code, so the object must name the leaf and not the group.
			name:           "a nested step keeps the leaf and its message",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"group": step(ABORTED_TestWorkflowStepStatus, "", ""),
				"leaf":  step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonExecutionStuck)),
			},
			sigSequence: []TestWorkflowSignature{
				{Ref: "group", Children: []TestWorkflowSignature{{Ref: "leaf"}}},
				{Ref: "leaf"},
			},
			stop: Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonExecutionStuck, Causes: map[string]string{"leaf": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonExecutionStuck), "leaf",
				"", StopActorRunner),
		},
		{
			// An actor with no reason and no rule of its own still leaves a message on the step it
			// stopped. The unknown rule must name that step, or the only text is lost.
			name:           "an unknown stop in a later step keeps that step and its message",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			stop:        Stop{Code: aborted, Actor: StopActorControlPlane, Causes: map[string]string{"a": ""}},
			want: details(StatusDetailsTypeUnknown, string(StopReasonUnknown), "a",
				"", StopActorControlPlane),
		},
		{
			name:           "no signal reads unknown with the message of the initialization step",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, ""),
			steps:          map[string]TestWorkflowStepResult{},
			stop:           Stop{Code: aborted, Causes: map[string]string{"": ""}},
			want:           details(StatusDetailsTypeUnknown, string(StopReasonUnknown), "", "", ""),
		},
		{
			name:           "a step that the stop skipped does not carry the cause",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, healed, ""),
				"b": step(SKIPPED_TestWorkflowStepStatus, "The execution was aborted before. (by the runner)", ""),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}, {Ref: "b"}},
			stop:        Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonExecutionStuck, Causes: map[string]string{"a": ""}},
			want: details(StatusDetailsTypeExecutionFailure, string(StopReasonExecutionStuck), "a",
				"", StopActorRunner),
		},
		{
			name:           "a waiting cause after a job deadline keeps the deadline and the cause",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(ABORTED_TestWorkflowStepStatus, healed, string(StopReasonVolumeMountFailed)),
			steps:          map[string]TestWorkflowStepResult{"a": step(SKIPPED_TestWorkflowStepStatus, "", "")},
			sigSequence:    []TestWorkflowSignature{{Ref: "a"}},
			stop: Stop{Code: aborted, Causes: map[string]string{
				"": `Job timed out after 60 seconds: MountVolume.SetUp failed for volume "data" : secret "absent" not found`,
			}},
			want: details(StatusDetailsTypeInitFailure, string(StopReasonVolumeMountFailed), "",
				`Job timed out after 60 seconds: MountVolume.SetUp failed for volume "data" : secret "absent" not found`, ""),
		},
		{
			// Without the plain causes, for example on the recovery path, the object keeps the step
			// message as the heal wrote it, which is longer but still true.
			name:           "without plain causes the message is the step message",
			status:         ABORTED_TestWorkflowStatus,
			initialization: step(PASSED_TestWorkflowStepStatus, "", ""),
			steps: map[string]TestWorkflowStepResult{
				"a": step(ABORTED_TestWorkflowStepStatus, fatal, string(StopReasonContainerError)),
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a", Name: "Run test"}},
			stop:        Stop{Code: aborted},
			want:        details(StatusDetailsTypeExecutionFailure, string(StopReasonContainerError), "a", fatal, ""),
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

// TestTestWorkflowResult_ClassifyStatusAfterHeal drives the classifier through the heal instead of a
// hand-built result. The heal writes the code of the stop into the step that it stops, so only this
// shape shows what the classifier reads in a running deployment.
func TestTestWorkflowResult_ClassifyStatusAfterHeal(t *testing.T) {
	const defaultErrorStr = "Job has been aborted"
	aborted := string(ABORTED_TestWorkflowStatus)

	tests := []struct {
		name        string
		steps       map[string]TestWorkflowStepResult
		sigSequence []TestWorkflowSignature
		errorStr    string
		stop        Stop
		// cause is the plain cause that the runner gives for each step that the heal ended.
		cause       string
		wantType    StatusDetailsType
		wantReason  StopReason
		wantStep    string
		wantActor   StopActor
		wantMessage string
	}{
		{
			// The heal gives the group the status of its child and no message, so the object must
			// name the leaf that carries the words.
			name: "a fail-fast stop inside a group names the leaf",
			steps: map[string]TestWorkflowStepResult{
				"group": {Status: common.Ptr(RUNNING_TestWorkflowStepStatus)},
				"leaf":  {Status: common.Ptr(RUNNING_TestWorkflowStepStatus)},
			},
			sigSequence: []TestWorkflowSignature{
				{Ref: "group", Children: []TestWorkflowSignature{{Ref: "leaf"}}},
				{Ref: "leaf"},
			},
			errorStr:    "because another parallel worker failed",
			stop:        Stop{Code: aborted, Actor: StopActorFailFast},
			wantType:    StatusDetailsTypeExecutionFailure,
			wantReason:  StopReasonFailFast,
			wantStep:    "leaf",
			wantActor:   StopActorFailFast,
			wantMessage: "",
		},
		{
			// The job went away after the test already had a result, so the exit code wins and the
			// deleted job stays the mechanism.
			name: "a deleted job after a failed step keeps exit-code",
			steps: map[string]TestWorkflowStepResult{
				"failed": {Status: common.Ptr(FAILED_TestWorkflowStepStatus), ExitCode: 3},
				"next":   {Status: common.Ptr(QUEUED_TestWorkflowStepStatus)},
			},
			sigSequence: []TestWorkflowSignature{{Ref: "failed"}, {Ref: "next"}},
			errorStr:    "by the system",
			stop:        Stop{Code: aborted, Actor: StopActorSystem},
			wantType:    StatusDetailsTypeStepFailure,
			wantReason:  StopReasonExitCode,
			wantStep:    "failed",
			wantMessage: `The step "failed" exited with code 3.`,
		},
		{
			name: "a deleted job with no failed step keeps the actor of the system",
			steps: map[string]TestWorkflowStepResult{
				"a": {Status: common.Ptr(RUNNING_TestWorkflowStepStatus)},
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			errorStr:    "by the system",
			stop:        Stop{Code: aborted, Actor: StopActorSystem},
			wantType:    StatusDetailsTypeExecutionFailure,
			wantReason:  StopReasonJobDeleted,
			wantStep:    "a",
			wantActor:   StopActorSystem,
			wantMessage: "",
		},
		{
			// A real cause differs from the code of the stop, so it wins and it names no actor.
			name: "a cause that the pod recorded still wins over the stop",
			steps: map[string]TestWorkflowStepResult{
				"a": {Status: common.Ptr(RUNNING_TestWorkflowStepStatus), ErrorReason: string(StopReasonUnschedulable), ErrorMessage: "no node can run the pod: 0/12 nodes are available."},
			},
			sigSequence: []TestWorkflowSignature{{Ref: "a"}},
			errorStr:    "by the runner: the first step did not start before the initialization timeout of the workflow",
			stop:        Stop{Code: aborted, Actor: StopActorRunner, Reason: StopReasonInitTimeout},
			cause:       "the first step did not start before the initialization timeout of the workflow: 0/12 nodes are available",
			wantType:    StatusDetailsTypeInitFailure,
			wantReason:  StopReasonUnschedulable,
			wantStep:    "a",
			wantMessage: "the first step did not start before the initialization timeout of the workflow: 0/12 nodes are available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TestWorkflowResult{
				Status:         common.Ptr(RUNNING_TestWorkflowStatus),
				Initialization: &TestWorkflowStepResult{Status: common.Ptr(PASSED_TestWorkflowStepStatus)},
				Steps:          tt.steps,
			}
			reasonCode := string(tt.stop.Reason)
			if reasonCode == "" {
				reasonCode = string(actorReasons[tt.stop.Actor])
			}
			if reasonCode == "" && tt.stop.Actor == StopActorSystem {
				reasonCode = string(StopReasonJobDeleted)
			}

			stop := tt.stop
			stop.Causes = map[string]string{}
			for ref := range r.HealAbortedOrCanceled(tt.sigSequence, tt.errorStr, defaultErrorStr, tt.stop.Code, reasonCode, &StopCauses{Stop: tt.stop}) {
				stop.Causes[ref] = tt.cause
			}
			r.HealStatus(tt.sigSequence)

			got := r.ClassifyStatus(tt.sigSequence, stop)
			require.NotNil(t, got)
			assert.Equal(t, string(tt.wantType), got.Type_)
			assert.Equal(t, string(tt.wantReason), got.Reason)
			assert.Equal(t, tt.wantStep, got.Step)
			assert.Equal(t, string(tt.wantActor), got.Actor)
			assert.Equal(t, tt.wantMessage, got.Message)
		})
	}
}
