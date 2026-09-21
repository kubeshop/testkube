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
//     step. The cause wins over the stop, because the cause is what the user fixes. The heal also
//     writes the code of the stop into a step, and that echo belongs to the rules below.
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
		reason = actorReasons[stop.Actor]
	}

	// The heal writes the code of the stop into the step that it stops, so a step can hold a code
	// that the stop already carries. That echo is not a cause of its own, and the rules below own
	// it, so they keep the actor that decided the stop and the order that this list states.
	// A code that differs is a real cause, and Kubernetes or the test process reported it, so it
	// names no actor.
	if ref, step, ok := r.recordedCauseStep(sigSequence); ok && step.ErrorReason != string(r.healedReason(stop, reason)) {
		return NewStatusDetails("", step.ErrorReason, ref, step.ErrorMessage)
	}

	if reason != "" {
		return r.stopDetails(sigSequence, stop.Actor, reason)
	}

	// A step of the test failed, so the test already had a result. The status of the execution can be
	// aborted here, because the stop also ended a later step, and the failure still decides the
	// object. The stop is only the mechanism that ended the rest of the run.
	if ref, step, ok := r.failedStep(sigSequence); ok {
		return NewStatusDetails("", string(StopReasonExitCode), ref, step.ErrorMessage)
	}

	// A caller that does not name itself leaves the job without an annotation, so the stop reads as
	// a deleted job. A step that failed first wins, because the test already had a result.
	if stop.Actor == StopActorSystem {
		return r.stopDetails(sigSequence, stop.Actor, StopReasonJobDeleted)
	}

	// No signal explains the result, so the object still names the step that holds the message. A
	// stop can end a later step while the initialization step passed, and that message is the only
	// text the user has.
	return r.stopDetails(sigSequence, stop.Actor, StopReasonUnknown)
}

// healedReason returns the code that the heal writes into the step that this stop ends. The
// notifier gives a deleted job the code job-deleted, so the echo of that stop carries it too, even
// though rule 5 and not rule 3 reports it.
func (r *TestWorkflowResult) healedReason(stop Stop, reason StopReason) StopReason {
	if reason == "" && stop.Actor == StopActorSystem {
		return StopReasonJobDeleted
	}
	return reason
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
// when it did not pass, else the first leaf in the order of the signature that the stop ended.
// The ref is empty for the initialization step, because the object names a step of the workflow.
func (r *TestWorkflowResult) stoppedStep(sigSequence []TestWorkflowSignature) (string, TestWorkflowStepResult) {
	if r.Initialization != nil && isStopped(r.Initialization.Status) {
		return "", *r.Initialization
	}
	for _, sig := range sigSequence {
		// The heal gives a group the status of its children and never their message, so a group
		// would name a step with no words. Only a leaf holds the message of the stop.
		if len(sig.Children) > 0 {
			continue
		}
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
		// A group holds the status of its children and no cause of its own, so only a leaf counts.
		if len(sig.Children) > 0 {
			continue
		}
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

// failedStep returns the failed step that names the result. It prefers the first failed leaf in the
// order of the signature, and it falls back to a failed group. A step that the user marked optional
// never counts, because an optional step does not fail the execution.
func (r *TestWorkflowResult) failedStep(sigSequence []TestWorkflowSignature) (string, TestWorkflowStepResult, bool) {
	if r.Initialization != nil && r.Initialization.Status.Failed() {
		return "", *r.Initialization, true
	}
	var groupRef string
	var groupStep TestWorkflowStepResult
	var hasGroup bool
	for _, sig := range sigSequence {
		step, ok := r.Steps[sig.Ref]
		if !ok || !step.Status.Failed() || sig.Optional {
			continue
		}
		// A leaf names the failure exactly, so it wins. A group can still fail on its own, because
		// a negative group fails when its children pass, and then it is the only failure there is.
		if len(sig.Children) == 0 {
			return sig.Ref, step, true
		}
		if !hasGroup {
			groupRef, groupStep, hasGroup = sig.Ref, step, true
		}
	}
	return groupRef, groupStep, hasGroup
}
