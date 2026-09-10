package executionworkertypes

import "errors"

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

// StartError is an error from one phase of an execution start. Reason names that phase.
type StartError struct {
	Reason StartReason
	Err    error
}

func (e *StartError) Error() string { return e.Err.Error() }

func (e *StartError) Unwrap() error { return e.Err }

// WithStartReason marks err with the reason of the phase that failed. An inner phase
// names a more specific reason, so an error that already carries one keeps it.
func WithStartReason(err error, reason StartReason) error {
	if err == nil {
		return nil
	}
	var startErr *StartError
	if errors.As(err, &startErr) {
		return err
	}
	return &StartError{Reason: reason, Err: err}
}

// StartReasonOf returns the reason in the error chain, or StartReasonUnknown.
func StartReasonOf(err error) StartReason {
	var startErr *StartError
	if errors.As(err, &startErr) {
		return startErr.Reason
	}
	return StartReasonUnknown
}
