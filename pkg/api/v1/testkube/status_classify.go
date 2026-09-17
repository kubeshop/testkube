package testkube

// ClassifyStatus returns why the execution did not pass, and nil for an execution that passed.
// The caller runs it on the final result, after the heal functions, and gives what it knows about
// the stop. The signature sets the order of the steps, so the classifier reads them as the user
// wrote them.
//
// The first rule that matches wins:
//
//  1. A person decided the stop, so the result is a cancel and not a failure.
//  2. A step holds a recorded cause, for example a signal of Kubernetes or a code of a toolkit
//     step. The cause wins over the stop, because the cause is what the user fixes.
//  3. The stop names a reason, or an actor that stops without one.
//  4. A step of the test failed.
//  5. A component that does not name itself stopped the execution, so the job is gone.
//  6. No signal explains the result.
//
// Rule 5 follows rule 4, because a deleted job does not change a test that already failed.
func (r *TestWorkflowResult) ClassifyStatus(sigSequence []TestWorkflowSignature, stop Stop) *TestWorkflowStatusDetails {
	if r == nil || r.IsPassed() {
		return nil
	}

	if stop.Actor == StopActorUser || stop.Actor == StopActorAPI {
		reason := stop.Reason
		if reason == "" {
			reason = StopReasonUserCancel
		}
		return r.stopDetails(sigSequence, stop.Actor, reason)
	}

	reason := stop.Reason
	if reason == "" {
		// These actors stop an execution without a reason, so the actor names the cause.
		reason = actorReasons[stop.Actor]
	}

	// The heal writes the code of the stop into the step that it stops, so a step can hold the code
	// that the stop already carries. Only a code that differs is a cause of its own, and a code that
	// matches belongs to the stop and keeps the actor that decided it.
	if ref, step, ok := r.recordedCauseStep(sigSequence); ok && step.ErrorReason != string(reason) {
		// A cause of its own names no actor, because Kubernetes and the test process report it and
		// no component decided it.
		return NewStatusDetails("", step.ErrorReason, ref, step.ErrorMessage)
	}

	if reason != "" {
		return r.stopDetails(sigSequence, stop.Actor, reason)
	}

	if r.IsFailed() {
		if ref, step, ok := r.failedStep(sigSequence); ok {
			return NewStatusDetails("", string(StopReasonExitCode), ref, step.ErrorMessage)
		}
	}

	// A caller that does not name itself leaves the job without an annotation, so the stop reads as
	// a deleted job. A step that failed first wins, because the test already had a result.
	if stop.Actor == StopActorSystem {
		return r.stopDetails(sigSequence, stop.Actor, StopReasonJobDeleted)
	}

	message := ""
	if r.Initialization != nil {
		message = r.Initialization.ErrorMessage
	}
	return NewStatusDetails(stop.Actor, string(StopReasonUnknown), "", message)
}

// actorReasons holds the actors that stop an execution and send no reason code with it.
var actorReasons = map[StopActor]StopReason{
	StopActorFailFast: StopReasonFailFast,
	StopActorTrigger:  StopReasonTriggerAbort,
}

// stopDetails builds the object for a stop that an actor or a reason explains. The message comes
// from the step that the stop ended, so a reader gets the text of the step next to the code.
func (r *TestWorkflowResult) stopDetails(sigSequence []TestWorkflowSignature, actor StopActor, reason StopReason) *TestWorkflowStatusDetails {
	ref, step := r.stoppedStep(sigSequence)
	return NewStatusDetails(actor, string(reason), ref, step.ErrorMessage)
}

// stoppedStep returns the step that holds the message of the stop. That is the initialization step
// when it did not pass, else the first step in the order of the signature that the stop ended.
// The ref is empty for the initialization step, because the object names a step of the workflow.
func (r *TestWorkflowResult) stoppedStep(sigSequence []TestWorkflowSignature) (string, TestWorkflowStepResult) {
	if r.Initialization != nil && isStopped(r.Initialization.Status) {
		return "", *r.Initialization
	}
	for _, sig := range sigSequence {
		if step, ok := r.Steps[sig.Ref]; ok && isStopped(step.Status) {
			return sig.Ref, step
		}
	}
	// No step of the workflow holds the stop, so the initialization step gives the message.
	if r.Initialization != nil {
		return "", *r.Initialization
	}
	return "", TestWorkflowStepResult{}
}

// isStopped reports whether the stop ended the step. A timeout and a cancel end a step as an abort
// does, so all three carry the message of the stop.
func isStopped(status *TestWorkflowStepStatus) bool {
	return status.AnyAborted() || status.Canceled()
}

// recordedCauseStep returns the step that holds a reason code of its own. It reads the
// initialization step first, then the steps in the order of the signature. A step that the user
// marked optional counts only when no other step holds a code, because an optional step does not
// decide the result of the execution.
func (r *TestWorkflowResult) recordedCauseStep(sigSequence []TestWorkflowSignature) (string, TestWorkflowStepResult, bool) {
	if r.Initialization != nil && r.Initialization.ErrorReason != "" && !r.Initialization.Status.Passed() {
		return "", *r.Initialization, true
	}
	var optionalRef string
	var optionalStep TestWorkflowStepResult
	var hasOptional bool
	for _, sig := range sigSequence {
		step, ok := r.Steps[sig.Ref]
		if !ok || step.ErrorReason == "" || step.Status.Passed() {
			continue
		}
		if !sig.Optional {
			return sig.Ref, step, true
		}
		if !hasOptional {
			optionalRef, optionalStep, hasOptional = sig.Ref, step, true
		}
	}
	return optionalRef, optionalStep, hasOptional
}

// failedStep returns the first step in the order of the signature that failed and that the user did
// not mark optional. An optional step that failed does not fail the execution.
func (r *TestWorkflowResult) failedStep(sigSequence []TestWorkflowSignature) (string, TestWorkflowStepResult, bool) {
	if r.Initialization != nil && r.Initialization.Status.Failed() {
		return "", *r.Initialization, true
	}
	for _, sig := range sigSequence {
		if step, ok := r.Steps[sig.Ref]; ok && step.Status.Failed() && !sig.Optional {
			return sig.Ref, step, true
		}
	}
	return "", TestWorkflowStepResult{}, false
}
