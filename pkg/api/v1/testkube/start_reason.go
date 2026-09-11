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
