package runner

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
)

// healRecoveredResult finishes a result that the runner recovers after it lost the watch of the execution.
// The caller gives the recorded cause, because a parallel worker takes it from the result of its parent.
// It does not change a result that is already finished, because a second heal marks its aborted steps as skipped
// and nests their earlier messages.
func healRecoveredResult(result *testkube.TestWorkflowResult, sigSequence []testkube.TestWorkflowSignature, scheduledAt time.Time, cause, causeReason string) {
	if result.IsFinished() {
		return
	}
	result.HealAbortedOrCanceled(sigSequence, cause, controller.DefaultErrorMessage, string(testkube.ABORTED_TestWorkflowStatus), causeReason)
	result.HealTimestamps(sigSequence, scheduledAt, time.Time{}, time.Time{}, true)
	result.HealDuration(scheduledAt)
	result.HealMissingPauseStatuses()
	result.HealStatus(sigSequence)
}

// recordedCause returns the message and the reason code of the initialization step, or of the first step in the
// order of the signature, that did not finish normally. A step that passed or failed holds its own result, not the
// cause of the stop. One walk returns both, so the message and the code always come from the same step.
// The signature must belong to the result, because the step refs of the signature index its steps.
func recordedCause(result *testkube.TestWorkflowResult, sigSequence []testkube.TestWorkflowSignature) (message, reason string) {
	if result.Initialization != nil && stopCause(*result.Initialization) {
		return result.Initialization.ErrorMessage, result.Initialization.ErrorReason
	}
	for _, sig := range sigSequence {
		if step := result.Steps[sig.Ref]; stopCause(step) {
			return step.ErrorMessage, step.ErrorReason
		}
	}
	return "", ""
}

// stopCause reports whether the step holds a message and did not finish normally.
func stopCause(step testkube.TestWorkflowStepResult) bool {
	finishedNormally := step.Status.Finished() && !step.Status.Aborted() && !step.Status.Canceled()
	return step.ErrorMessage != "" && !finishedNormally
}
