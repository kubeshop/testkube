package testkube

// StartReason is the token the runner sends to the control plane when it declines
// an execution. The values are stable, because filters and telemetry use them.
type StartReason string

const (
	StartReasonImagePullFailed   StartReason = "image-pull-failed"
	StartReasonDefinitionInvalid StartReason = "definition-invalid"
	StartReasonResourceFailed    StartReason = "resource-create-failed"
	StartReasonJobCreateFailed   StartReason = "job-create-failed"
	StartReasonUnknown           StartReason = "start-failed"
)

// Sentence returns the words for the reason in a message for people, as the tail of
// "Failed to run execution: ...". It returns an empty string for a value it does not
// know, so a reader can keep the raw token.
func (r StartReason) Sentence() string {
	switch r {
	case StartReasonImagePullFailed:
		return "the image could not be pulled"
	case StartReasonDefinitionInvalid:
		return "the workflow definition is invalid"
	case StartReasonResourceFailed:
		return "a secret, config map, or volume claim could not be created"
	case StartReasonJobCreateFailed:
		return "the job could not be created"
	case StartReasonUnknown:
		return "the runner could not start the execution"
	default:
		return ""
	}
}
