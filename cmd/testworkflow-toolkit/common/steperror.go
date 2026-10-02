package common

import (
	"errors"
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// ReasonError is an error that knows the code of the failure. Fail writes that code, so a command
// gives its error to Fail and does not choose the code itself.
type ReasonError interface {
	error
	Reason() testkube.StopReason
}

// WithReason gives the error the code of the failure, so Fail and ExitOnError write that code. The
// message stays the message of the error. A nil error stays nil, so ExitOnError still sees no error.
func WithReason(reason testkube.StopReason, err error) error {
	if err == nil {
		return nil
	}
	return &reasonError{reason: reason, err: err}
}

// reasonError is the error that WithReason returns.
type reasonError struct {
	reason testkube.StopReason
	err    error
}

func (e *reasonError) Error() string               { return e.err.Error() }
func (e *reasonError) Unwrap() error               { return e.err }
func (e *reasonError) Reason() testkube.StopReason { return e.reason }

// Fail writes the error as the step message, then prints it and exits. When the error or an error
// that it wraps is a ReasonError, Fail also writes its code.
func Fail(err error) {
	writeStepError(err.Error())
	writeStepReason(string(ReasonOf(err)))
	ui.Fail(err)
}

// ReasonOf returns the code that Fail writes for the error: the code of the first ReasonError in
// its chain, and an empty code when there is none.
func ReasonOf(err error) testkube.StopReason {
	var reasonErr ReasonError
	if errors.As(err, &reasonErr) {
		return reasonErr.Reason()
	}
	return ""
}

// Failf writes the formatted error as the step message, then prints it and exits.
func Failf(format string, params ...interface{}) {
	writeStepError(fmt.Sprintf(format, params...))
	ui.Failf(format, params...)
}

// ExitOnError writes the item and the error as the step message, then prints them and exits. It
// also writes the code of a ReasonError in the error, as Fail does. When the error is nil, only
// pkg/ui runs, because it prints the item in verbose mode.
func ExitOnError(item string, err error) {
	if err != nil {
		writeStepError(fmt.Sprintf("%s: %s", item, err.Error()))
		writeStepReason(string(ReasonOf(err)))
	}
	ui.ExitOnError(item, err)
}

// writeStepError writes the message to the file that TK_ERR_FILE names. A write error is not
// important, because the command prints the same error to the log.
func writeStepError(message string) {
	writeStepFile(constants.EnvStepErrorFile, message)
}

// writeStepReason writes the code to the file that TK_REASON_FILE names. A failure without a code
// writes no file.
func writeStepReason(reason string) {
	if reason == "" {
		return
	}
	writeStepFile(constants.EnvStepReasonFile, reason)
}

// writeStepFile writes the content to the file that the environment variable names. An init
// process that does not set the variable gets no file, and a write error is not important,
// because the command prints the same error to the log.
func writeStepFile(env, content string) {
	path := os.Getenv(env)
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(content), 0666)
}
