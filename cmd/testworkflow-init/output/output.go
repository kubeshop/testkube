package output

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
)

func Failf(exitCode uint8, message string, args ...interface{}) {
	// Print message
	fmt.Printf(message+"\n", args...)

	// Kill the sub-process
	data.Step.Kill()

	// Exit
	os.Exit(int(exitCode))
}
