package common

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// FailWithReason writes the error as the step message and the reason as its code, then prints the
// error and exits. The code lets a reader act on the failure without the words of the message.
func FailWithReason(reason testkube.StopReason, err error) {
	writeStepError(err.Error())
	writeStepReason(string(reason))
	ui.Fail(err)
}

// ExitOnErrorWithReason writes the item and the error as the step message and the reason as its
// code, then prints them and exits. When the error is nil, only pkg/ui runs.
func ExitOnErrorWithReason(reason testkube.StopReason, item string, err error) {
	if err != nil {
		writeStepError(fmt.Sprintf("%s: %s", item, err.Error()))
		writeStepReason(string(reason))
	}
	ui.ExitOnError(item, err)
}

// Fail writes the error as the step message, then prints it and exits.
func Fail(err error) {
	writeStepError(err.Error())
	ui.Fail(err)
}

// Failf writes the formatted error as the step message, then prints it and exits.
func Failf(format string, params ...interface{}) {
	writeStepError(fmt.Sprintf(format, params...))
	ui.Failf(format, params...)
}

// ExitOnError writes the item and the error as the step message, then prints them and exits.
// When the error is nil, only pkg/ui runs, because it prints the item in verbose mode.
func ExitOnError(item string, err error) {
	if err != nil {
		writeStepError(fmt.Sprintf("%s: %s", item, err.Error()))
	}
	ui.ExitOnError(item, err)
}

// writeStepError writes the message to the file that TK_ERR_FILE names. A write error is not
// important, because the command prints the same error to the log.
func writeStepError(message string) {
	writeStepFile(constants.EnvStepErrorFile, message)
}

// writeStepReason writes the code to the file that TK_REASON_FILE names.
func writeStepReason(reason string) {
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
