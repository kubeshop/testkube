package executionworkertypes

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// StartError is an error from one phase of an execution start. Reason names that phase.
type StartError struct {
	Reason testkube.StartReason
	Err    error
}

func (e *StartError) Error() string { return e.Err.Error() }

func (e *StartError) Unwrap() error { return e.Err }

// WithStartReason marks err with the reason of the phase that failed. An inner phase
// names a more specific reason, so an error that already carries one keeps it.
func WithStartReason(err error, reason testkube.StartReason) error {
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
func StartReasonOf(err error) testkube.StartReason {
	var startErr *StartError
	if errors.As(err, &startErr) {
		return startErr.Reason
	}
	return testkube.StartReasonUnknown
}
