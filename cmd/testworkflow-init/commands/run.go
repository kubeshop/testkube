package commands

import (
	"context"
	"slices"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runtime"
	"github.com/kubeshop/testkube/pkg/executiondata"
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

	workingDir := ""
	if container.Config.WorkingDir != nil {
		workingDir = *container.Config.WorkingDir
	}

	// Narrow the run to specific test cases, before the command is resolved so
	// that the selection is visible to its arguments. This reads the report a
	// previous attempt left behind; the verdict below reads the one this attempt
	// is about to write.
	selection := &testCasesSelection{}
	if run.TestCases != nil && run.TestCases.Select != nil {
		resolved, err := resolveTestCaseSelection(run.TestCases, workingDir, state.InternalConfig.Execution.Rerun)
		if err != nil {
			output.ExitErrorf(constants.CodeInputError, "test case selection: %s", err.Error())
		}
		selection = resolved
		if err = selection.Export(); err != nil {
			output.ExitErrorf(constants.CodeInternal, "test case selection: %s", err.Error())
		}
		// The selection machine goes first so that testCases.* resolves before
		// anything else claims the name.
		machine = expressions.CombinedMachines(selection.Machine(), machine)

		// Nothing selected means either the first attempt of a narrowing retry,
		// which must run everything, or a re-run step with nothing left to do.
		if !selection.Narrowed {
			if finished := applyEmptySelectionPolicy(step, run.TestCases.Select.Empty); finished {
				return
			}
		}
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
		if innerExpr, isPure := expressions.ExtractPureTemplateExpression(command[i]); isPure && !expressions.IsWildcardAccessorOnly(innerExpr) {
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

	// An output another workflow withheld resolves to a marker instead of the value it
	// was meant to carry. Running the command would hand the marker to the tool as if
	// it were that value, so fail while the cause is still visible.
	if markers := executiondata.WithheldMarkersIn(command); len(markers) > 0 {
		output.ExitErrorf(constants.CodeInputError, "%s", executiondata.WithheldError("the command of this step", markers).Error())
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

	// A testCases policy replaces the exit code as the source of the verdict:
	// the tool exits non-zero for a failure the workflow may have declared
	// acceptable. Only a Pro preset can put a policy here - see StubTestCases.
	var outcome testCasesOutcome
	if run.TestCases != nil {
		outcome = applyTestCases(run.Ref, run.TestCases, workingDir, int(result.ExitCode), selection.Narrowed)
		success = outcome.Success
	}

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

	// Notify about the status. The counters go out before the execution result,
	// so a reader sees what the report said before the verdict built from it.
	if outcome.Results != nil {
		instructions.PrintHintDetails(step.Ref, constants.InstructionTestResults, outcome.Results)
	}
	step.SetStatus(status).SetExitCode(result.ExitCode)
	orchestration.FinishExecution(step, constants.ExecutionResult{ExitCode: result.ExitCode, Details: outcome.Details, Iteration: int(step.Iteration)})
}
