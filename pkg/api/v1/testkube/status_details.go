package testkube

// StatusDetailsType is the layer that made an execution fail. It is part of the API, so
// the values are stable. The dashboard reads it for the label and the icon, and a filter
// and a webhook condition select on it.
type StatusDetailsType string

const (
	// StatusDetailsTypeInitFailure is a failure before the test ran: the definition, the
	// cluster, or the setup of the pod.
	StatusDetailsTypeInitFailure StatusDetailsType = "init-failure"
	// StatusDetailsTypeExecutionFailure is a failure of the infrastructure while the test ran.
	StatusDetailsTypeExecutionFailure StatusDetailsType = "execution-failure"
	// StatusDetailsTypeStepFailure is a failure of the test itself.
	StatusDetailsTypeStepFailure StatusDetailsType = "step-failure"
	// StatusDetailsTypeUserCancel is a stop that a person decided. It is not a failure.
	StatusDetailsTypeUserCancel StatusDetailsType = "user-cancel"
	// StatusDetailsTypeUnknown is a status that no signal explains.
	StatusDetailsTypeUnknown StatusDetailsType = "unknown"
)

// statusDetailsTypeNames holds the words that the dashboard shows for each type, so that every
// place that shows a type to people uses the same words.
var statusDetailsTypeNames = map[StatusDetailsType]string{
	StatusDetailsTypeInitFailure:      "Configuration error",
	StatusDetailsTypeExecutionFailure: "Infrastructure failure",
	StatusDetailsTypeStepFailure:      "Test failure",
	StatusDetailsTypeUserCancel:       "Canceled by the user",
	StatusDetailsTypeUnknown:          "Unknown cause",
}

// DisplayName returns the words for the type, and the type itself for a type without words.
func (t StatusDetailsType) DisplayName() string {
	if name, ok := statusDetailsTypeNames[t]; ok {
		return name
	}
	return string(t)
}

// statusDetailsTypes maps each reason code to the layer that it belongs to. A reason that
// is not in the table gives StatusDetailsTypeUnknown.
var statusDetailsTypes = map[string]StatusDetailsType{
	// The execution failed before the test ran.
	string(StartReasonDefinitionInvalid): StatusDetailsTypeInitFailure,
	string(StopReasonTemplateMissing):    StatusDetailsTypeInitFailure,
	string(StopReasonQueueLimitExceeded): StatusDetailsTypeInitFailure,
	string(StartReasonImagePullFailed):   StatusDetailsTypeInitFailure,
	string(StartReasonJobCreateFailed):   StatusDetailsTypeInitFailure,
	string(StartReasonResourceFailed):    StatusDetailsTypeInitFailure,
	string(StartReasonUnknown):           StatusDetailsTypeInitFailure,
	string(StopReasonAdmissionDenied):    StatusDetailsTypeInitFailure,
	string(StopReasonConfigMissing):      StatusDetailsTypeInitFailure,
	string(StopReasonUnschedulable):      StatusDetailsTypeInitFailure,
	string(StopReasonVolumeMountFailed):  StatusDetailsTypeInitFailure,
	string(StopReasonInitTimeout):        StatusDetailsTypeInitFailure,
	string(StopReasonGitAuthFailed):      StatusDetailsTypeInitFailure,
	string(StopReasonGitCloneFailed):     StatusDetailsTypeInitFailure,

	// The infrastructure stopped the execution while the test ran.
	string(StopReasonEvicted):              StatusDetailsTypeExecutionFailure,
	string(StopReasonPreempted):            StatusDetailsTypeExecutionFailure,
	string(StopReasonNodeShutdown):         StatusDetailsTypeExecutionFailure,
	string(StopReasonOOMKilled):            StatusDetailsTypeExecutionFailure,
	string(StopReasonProcessKilled):        StatusDetailsTypeExecutionFailure,
	string(StopReasonContainerError):       StatusDetailsTypeExecutionFailure,
	string(StopReasonDeadlineExceeded):     StatusDetailsTypeExecutionFailure,
	string(StopReasonStepTimeout):          StatusDetailsTypeExecutionFailure,
	string(StopReasonServiceNotReady):      StatusDetailsTypeExecutionFailure,
	string(StopReasonArtifactUploadFailed): StatusDetailsTypeExecutionFailure,
	string(StopReasonExecutionStuck):       StatusDetailsTypeExecutionFailure,
	string(StopReasonExecutionTimeout):     StatusDetailsTypeExecutionFailure,
	string(StopReasonTransitionTimeout):    StatusDetailsTypeExecutionFailure,
	string(StopReasonStopNotConfirmed):     StatusDetailsTypeExecutionFailure,
	string(StopReasonQueueTimeout):         StatusDetailsTypeExecutionFailure,
	string(StopReasonQueuedTooLong):        StatusDetailsTypeExecutionFailure,
	string(StopReasonWorkerResumeFailed):   StatusDetailsTypeExecutionFailure,
	string(StopReasonSuperseded):           StatusDetailsTypeExecutionFailure,
	string(StopReasonFailFast):             StatusDetailsTypeExecutionFailure,
	string(StopReasonTriggerAbort):         StatusDetailsTypeExecutionFailure,
	string(StopReasonJobDeleted):           StatusDetailsTypeExecutionFailure,
	string(StopReasonAbortAll):             StatusDetailsTypeExecutionFailure,

	// The test itself failed.
	string(StopReasonExitCode):            StatusDetailsTypeStepFailure,
	string(StopReasonChildWorkflowFailed): StatusDetailsTypeStepFailure,

	// A person stopped the execution.
	string(StopReasonUserCancel):  StatusDetailsTypeUserCancel,
	string(StopReasonForceCancel): StatusDetailsTypeUserCancel,
}

// StatusDetailsTypeOf returns the layer for a stop. A person as the actor gives
// user-cancel for every reason, because a cancel is never a failure. A person who stops
// all executions of a workflow makes the same decision as a person who cancels one.
func StatusDetailsTypeOf(actor StopActor, reason string) StatusDetailsType {
	if actor.IsPerson() {
		return StatusDetailsTypeUserCancel
	}
	if detailsType, ok := statusDetailsTypes[reason]; ok {
		return detailsType
	}
	return StatusDetailsTypeUnknown
}

// NewStatusDetails builds the object for a stop. The step and the message identify the
// step that holds the cause, and both stay empty when no step holds one. The reason is
// required, so an empty reason becomes user-cancel for a person and unknown for other actors.
func NewStatusDetails(actor StopActor, reason, step, message string) *TestWorkflowStatusDetails {
	if reason == "" {
		reason = string(StopReasonUnknown)
		if actor.IsPerson() {
			reason = string(StopReasonUserCancel)
		}
	}
	return &TestWorkflowStatusDetails{
		Type_:   string(StatusDetailsTypeOf(actor, reason)),
		Reason:  reason,
		Message: message,
		Step:    step,
		Actor:   string(actor),
	}
}
