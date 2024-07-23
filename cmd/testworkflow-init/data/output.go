package data

import (
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
)

func Failf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	output.Std.Printf(message+"\n", args...)

	// Exit
	os.Exit(int(exitCode))
}
