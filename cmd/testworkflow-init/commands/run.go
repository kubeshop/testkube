package commands

import (
	"context"
	"slices"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runtime"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func Run(ctx context.Context, run lite.ActionExecute, container lite.LiteActionContainer) {
	machine := runtime.GetInternalTestWorkflowMachine()
	state := data.GetState()
	step := state.GetStep(run.Ref)

	// Abandon executing if the step was finished before
	if step.IsFinished() {
		return
	}

	// Obtain command to run
	command := make([]string, 0)
	if container.Config.Command != nil {
		command = slices.Clone(*container.Config.Command)
	}
	if container.Config.Args != nil {
		command = append(command, *container.Config.Args...)
	}

	// Ensure the command is not empty
	if len(command) == 0 {
		output.ExitErrorf(constants.CodeInputError, "command is required")
	}

	// Resolve the command to run
	for i := range command {
		value, err := expressions.CompileAndResolveTemplate(command[i], machine, expressions.FinalizerFail)
		if err != nil {
			output.ExitErrorf(constants.CodeInternal, "failed to compute argument '%d': %s", i, err.Error())
		}
		command[i], _ = value.Static().StringValue()
	}

	// Run the operation with context
	execution := orchestration.Executions.CreateWithContext(ctx, command[0], command[1:])
	result, err := execution.Run()
	if err != nil {
		output.ExitErrorf(constants.CodeInternal, "failed to execute: %v", err)
	}

	// Initialize local state
	var status constants.StepStatus

	success := result.ExitCode == 0

	// Compute the result
	if run.Negative {
		success = !success
	}
	if result.Aborted {
		status = constants.StepStatusAborted
	} else if success {
		status = constants.StepStatusPassed
	} else {
		status = constants.StepStatusFailed
	}

	// Abandon saving execution data if the step has been finished before
	if step.IsFinished() {
		return
	}

	// Notify about the status
	step.SetStatus(status).SetExitCode(result.ExitCode)
	orchestration.FinishExecution(step, constants.ExecutionResult{ExitCode: result.ExitCode, Iteration: int(step.Iteration)})
}
