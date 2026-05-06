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
	expandedCommand := make([]string, 0, len(command))
	for i := range command {
		// Check if this argument is a pure template expression that may resolve to an array
		if innerExpr, isPure := expressions.ExtractPureTemplateExpression(command[i]); isPure {
			expr, err := expressions.CompileAndResolve(innerExpr, machine, expressions.FinalizerFail)
			if err != nil {
				output.ExitErrorf(constants.CodeInternal, "failed to compute argument '%d': %s", i, err.Error())
			}
			if expr.Static() != nil {
				// Array result: expand into individual arguments
				if items, sliceErr := expr.Static().SliceValue(); sliceErr == nil {
					for _, item := range items {
						sv := expressions.NewValue(item)
						s, _ := sv.StringValue()
						expandedCommand = append(expandedCommand, s)
					}
					continue
				}
				// Non-array result: reuse the already-resolved value
				s, _ := expr.Static().StringValue()
				expandedCommand = append(expandedCommand, s)
				continue
			}
		}
		value, err := expressions.CompileAndResolveTemplate(command[i], machine, expressions.FinalizerFail)
		if err != nil {
			output.ExitErrorf(constants.CodeInternal, "failed to compute argument '%d': %s", i, err.Error())
		}
		s, _ := value.Static().StringValue()
		expandedCommand = append(expandedCommand, s)
	}
	command = expandedCommand

	// Ensure the command is not empty after expansion
	if len(command) == 0 {
		output.ExitErrorf(constants.CodeInputError, "command is required")
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
