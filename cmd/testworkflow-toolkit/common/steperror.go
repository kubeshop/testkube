package common

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/ui"
)

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
	path := os.Getenv(constants.EnvStepErrorFile)
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(message), 0666)
}
