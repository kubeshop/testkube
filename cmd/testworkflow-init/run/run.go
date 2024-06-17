package run

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
)

const (
	defaultBinPath = "/.tktw/bin"
)

func execute(cmd string, args ...string) {
	data.Step.Run(data.Config.Negative, cmd, args...)

	if data.Config.Negative {
		fmt.Printf("Expected to fail: finished with exit code %d.\n", data.Step.ExitCode)
	} else if data.Config.Debug {
		fmt.Printf("Exit code: %d.\n", data.Step.ExitCode)
	}
}

func Run(cmd string, args []string) {
	// Ensure the built-in binaries are available
	if os.Getenv("PATH") == "" {
		_ = os.Setenv("PATH", defaultBinPath)
	} else {
		_ = os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), defaultBinPath))
	}

	// Instantiate the command and run
	execute(cmd, args...)

	// Retry if it's expected
	// TODO: Support nested retries
	step := data.State.GetStep(data.Step.Ref)
	for step.Iteration <= uint64(data.Config.RetryCount) {
		expr, err := data.Expression(data.Config.RetryUntil, data.LocalMachine)
		if err != nil {
			fmt.Printf("Failed to execute retry condition: %s: %s\n", data.Config.RetryUntil, err.Error())
			data.Finish()
		}
		v, _ := expr.BoolValue()
		if v {
			break
		}
		step.Next()
		fmt.Printf("\nExit code: %d • Retrying: attempt #%d (of %d):\n", data.Step.ExitCode, step.Iteration-1, data.Config.RetryCount)
		execute(cmd, args...)
	}

	// Finish
	data.Finish()
}
