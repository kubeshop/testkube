package commands

import (
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func Run(run lite.ActionExecute, container lite.LiteActionContainer) {
	state := data.GetState()
	step := state.GetStep(run.Ref)

	// TODO: Run the timeout
	// TODO: Compute the retry

	// Obtain command to run
	command := make([]string, 0)
	if container.Config.Command != nil {
		command = slices.Clone(*container.Config.Command)
	}
	if container.Config.Args != nil {
		command = append(command, *container.Config.Args...)
	}

	// Resolve the command to run
	wd, _ := os.Getwd()
	if wd == "" {
		wd = "/"
	}
	machine := expressions.CombinedMachines(data.RefSuccessMachine, data.AliasMachine, data.StateMachine, libs.NewFsMachine(os.DirFS("/"), wd))
	for i := range command {
		value, err := expressions.CompileAndResolveTemplate(command[i], machine, expressions.FinalizerFail)
		if err != nil {
			panic(fmt.Sprintf("failed to compute argument '%d': %s", i, err.Error()))
		}
		command[i], _ = value.Static().StringValue()
	}

	// Run the operation
	execution := orchestration.Executions.Create(command[0], command[1:])
	result, err := execution.Run()
	if err != nil {
		data.Failf(data.CodeInternal, "failed to execute: %v", err)
	}

	// Initialize local state
	var status data.StepStatus

	success := result.ExitCode == 0

	// Compute the result
	if run.Negative {
		success = !success
	}
	if result.Aborted {
		status = data.StepStatusAborted
	} else if success {
		status = data.StepStatusPassed
	} else {
		status = data.StepStatusFailed
	}

	// TODO: Retry if expected

	// Notify about the status
	step.SetStatus(status).SetExitCode(result.ExitCode)
	orchestration.FinishExecution(step, constants.ExecutionResult{ExitCode: result.ExitCode, Iteration: 0})
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
