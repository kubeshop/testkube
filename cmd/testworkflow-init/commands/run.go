package commands

import (
	"fmt"
	"os"
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

	// List all the parents
	leaf := []*data.StepData{step}
	for i := range step.Parents {
		leaf = append(leaf, state.GetStep(step.Parents[i]))
	}

	// TODO: Consider moving timeout to main.go
	// Create timeout finalizer
	finalizeTimeout := func() {
		// Check timed out steps in leaf
		timedOut := orchestration.GetTimedOut(leaf...)
		if timedOut == nil {
			return
		}

		// Iterate over timed out step
		for _, r := range timedOut {
			r.SetStatus(data.StepStatusTimeout)
			sub := state.GetSubSteps(r.Ref)
			for i := range sub {
				if sub[i].IsFinished() {
					continue
				}
				if sub[i].IsStarted() {
					sub[i].SetStatus(data.StepStatusTimeout)
				} else {
					sub[i].SetStatus(data.StepStatusSkipped)
				}
			}
			fmt.Println("Timed out.")
		}
		_ = orchestration.Executions.Kill()

		return
	}

	// Handle immediate timeout
	finalizeTimeout()

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

	// Register timeouts
	stopTimeoutWatcher := orchestration.WatchTimeout(finalizeTimeout, leaf...)
	defer stopTimeoutWatcher()

	// Ensure there won't be any hanging processes after the command is executed
	defer func() {
		_ = orchestration.Executions.Kill()
	}()

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

	// Abandon saving execution data if the step has been finished before
	if step.IsFinished() {
		return
	}

	// Notify about the status
	step.SetStatus(status).SetExitCode(result.ExitCode)
	orchestration.FinishExecution(step, constants.ExecutionResult{ExitCode: result.ExitCode, Iteration: int(step.Iteration)})
}
