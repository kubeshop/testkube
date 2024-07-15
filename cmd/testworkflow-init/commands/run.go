package commands

import (
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action"
)

func Run(run action.ActionExecute, container action.ActionContainer) {
	state := data.GetState()
	step := state.GetStep(run.Ref)

	// TODO: Run the timeout
	// TODO: Compute the retry

	// TODO: Compute the command/args TODO loop
	command := make([]string, 0)
	if container.Config.Command != nil {
		command = slices.Clone(*container.Config.Command)
	}
	if container.Config.Args != nil {
		command = append(command, *container.Config.Args...)
	}

	// Run the operation
	cmd := exec.Command(command[0], command[1:]...)
	out := data.NewOutputProcessor(run.Ref, os.Stdout)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Initialize local state
	var success bool
	var exitCode uint8
	var status data.StepStatus

	// Run the command
	err := cmd.Start()
	if err == nil {
		success, exitCode = getProcessStatus(cmd.Wait())
	} else {
		success, exitCode = getProcessStatus(err)
	}

	// Compute the result
	if run.Negative {
		success = !success
	}
	if success {
		status = data.StepStatusPassed
	} else {
		status = data.StepStatusFailed
	}

	// TODO: Retry if expected

	// Debug information
	fmt.Printf("Finished step '%s'.\n   Exit code: %d\n   Status: %s\n   Success: %v", run.Ref, exitCode, status, success)

	// Notify about the status
	step.SetStatus(status)
	orchestration.FinishExecution(step, constants.ExecutionResult{ExitCode: exitCode, Iteration: 0})

	// Save the data
	data.SaveState()
	data.SaveTerminationLog()
}

func getProcessStatus(err error) (bool, uint8) {
	if err == nil {
		return true, 0
	}
	if e, ok := err.(*exec.ExitError); ok {
		if e.ProcessState != nil {
			return false, uint8(e.ProcessState.ExitCode())
		}
		return false, 1
	}
	fmt.Println(err.Error())
	return false, 1
}
