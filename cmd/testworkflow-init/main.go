package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/expressions/libs"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func main() {
	// Prepare empty state file if it doesn't exist
	_, err := os.Stat(data.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		data.PrintHint(data.InitStepName, constants.InstructionStart)
		fmt.Print("Creating state...")
		err := os.WriteFile(data.StatePath, nil, 0777)
		if err != nil {
			fmt.Println(color.FgRed.Render(" error"))
			data.Failf(data.CodeInternal, "failed to create state file: %s", err.Error())
		}
		os.Chmod(data.StatePath, 0777)
		fmt.Println(" done")
	} else if err != nil {
		data.PrintHint(data.InitStepName, constants.InstructionStart)
		fmt.Print("Accessing state...")
		fmt.Println(color.FgRed.Render(" error"))
		data.Failf(data.CodeInternal, "cannot access state file: %s", err.Error())
	}

	// Store the instructions in the state if they are provided
	orchestration.Setup.UseEnv("01")
	instructions := os.Getenv(constants.EnvInstructions)
	orchestration.Setup.UseBaseEnv()
	if instructions != "" {
		fmt.Print("Initializing state...")
		err = json.Unmarshal([]byte(instructions), &data.GetState().Actions)
		if err != nil {
			fmt.Println(color.FgRed.Render(" error"))
			data.Failf(data.CodeInternal, "failed to read the actions from Pod: %s", err.Error())
		}
		fmt.Println(" done")

		// Release the memory
		instructions = ""
	}

	// Distribute the details
	currentContainer := lite.LiteActionContainer{}

	// Ensure there is a group index provided
	if len(os.Args) != 2 {
		data.Failf(data.CodeInternal, "invalid arguments provided - expected only one")
	}

	// Determine group index to run
	groupIndex, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		data.Failf(data.CodeInputError, "invalid run group passed: %s", err.Error())
	}

	// Handle aborting
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignal
		fmt.Println("The task was aborted.")
		orchestration.Executions.Abort()
	}()

	// Read the current state
	state := data.GetState()

	// Run the control server
	handlePause := func(ts time.Time, step *data.StepData) error {
		if step.PausedStart != nil {
			return nil
		}
		err = orchestration.Executions.Pause()
		if err != nil {
			fmt.Printf("warning: pause: %s\n", err.Error())
		}
		orchestration.Pause(step, *step.StartedAt)
		for _, parentRef := range step.Parents {
			parent := state.GetStep(parentRef)
			orchestration.Pause(parent, *step.StartedAt)
		}
		return err
	}
	handleResume := func(ts time.Time, step *data.StepData) error {
		if step.PausedStart == nil {
			return nil
		}
		err = orchestration.Executions.Resume()
		if err != nil {
			fmt.Printf("warning: resume: %s\n", err.Error())
		}
		orchestration.Resume(step, ts)
		for _, parentRef := range step.Parents {
			parent := state.GetStep(parentRef)
			orchestration.Resume(parent, ts)
		}
		return err
	}
	controlSrv := control.NewServer(constants.ControlServerPort, control.ControlServerOptions{
		HandlePause: func(ts time.Time) error {
			return handlePause(ts, state.GetStep(state.CurrentRef))
		},
		HandleResume: func(ts time.Time) error {
			return handleResume(ts, state.GetStep(state.CurrentRef))
		},
	})
	_, err = controlSrv.Listen()
	if err != nil {
		data.Failf(data.CodeInternal, "Failed to start control server at port %d: %s\n", constants.ControlServerPort, err.Error())
	}

	// Keep a list of paused steps for execution
	delayedPauses := make([]string, 0)

	// Interpret the operations
	for _, action := range state.GetActions(int(groupIndex)) {
		switch action.Type() {
		case lite.ActionTypeDeclare:
			state.GetStep(action.Declare.Ref).
				SetCondition(action.Declare.Condition).
				SetParents(action.Declare.Parents)

		case lite.ActionTypePause:
			state.GetStep(action.Pause.Ref).
				SetPausedOnStart(true)

		case lite.ActionTypeResult:
			state.GetStep(action.Result.Ref).
				SetResult(action.Result.Value)

		case lite.ActionTypeTimeout:
			state.GetStep(action.Timeout.Ref).
				SetTimeout(action.Timeout.Timeout)

		case lite.ActionTypeRetry:
			state.GetStep(action.Retry.Ref).
				SetRetryPolicy(data.RetryPolicy{Count: action.Retry.Count, Until: action.Retry.Until})

		case lite.ActionTypeContainerTransition:
			orchestration.Setup.SetConfig(action.Container.Config)
			orchestration.Setup.AdvanceEnv()
			currentContainer = *action.Container

		case lite.ActionTypeCurrentStatus:
			state.SetCurrentStatus(*action.CurrentStatus)

		case lite.ActionTypeStart:
			if *action.Start == "" {
				continue
			}
			step := state.GetStep(*action.Start)
			orchestration.Start(step)

			// Determine if the step should be skipped
			executable, err := step.ResolveCondition()
			if err != nil {
				panic(fmt.Sprintf("failed to determine condition of '%s' step: %s: %s", *action.Start, step.Condition, err.Error()))
			}
			if !executable {
				step.SetStatus(data.StepStatusSkipped)
			}

			// Delay the pause until next children execution
			if !step.IsFinished() && step.PausedOnStart {
				delayedPauses = append(delayedPauses, state.CurrentRef)
			}

		case lite.ActionTypeEnd:
			if *action.End == "" {
				continue
			}
			step := state.GetStep(*action.End)
			if step.Status == nil {
				status, err := step.ResolveResult()
				if err != nil {
					panic(fmt.Sprintf("failed to determine result of '%s' step: %s: %s", *action.End, step.Result, err.Error()))
				}
				step.SetStatus(status)
			}
			orchestration.End(step)

		case lite.ActionTypeSetup:
			// TODO: Handle error
			orchestration.Setup.UseEnv("00")
			step := state.GetStep(data.InitStepName)
			commands.Setup(*action.Setup)
			step.SetStatus(data.StepStatusPassed)
			orchestration.End(step)

		case lite.ActionTypeExecute:
			// Ignore running when the step is already resolved (= skipped)
			step := state.GetStep(action.Execute.Ref)
			if step.IsFinished() {
				continue
			}

			// Ignore when it is aborted
			if orchestration.Executions.IsAborted() {
				step.SetStatus(data.StepStatusAborted)
				continue
			}

			// Compute the pause
			paused := make([]string, 0)
			if slices.Contains(delayedPauses, action.Execute.Ref) {
				paused = append(paused, action.Execute.Ref)
			}
			for _, parentRef := range step.Parents {
				if slices.Contains(delayedPauses, parentRef) {
					paused = append(paused, parentRef)
				}
			}

			// Pause
			if len(paused) > 0 {
				delayedPauses = nil
				_ = handlePause(*step.StartedAt, step)
			}

			// Avoid execution if it's finished
			if step.IsFinished() {
				continue
			}

			// Iterate retries
			for {
				// Reset the status
				step.Status = nil

				// Ignore when it is aborted
				if orchestration.Executions.IsAborted() {
					step.SetStatus(data.StepStatusAborted)
					break
				}

				// Run the command
				commands.Run(*action.Execute, currentContainer)

				// TODO: Handle retry policy in tree independently
				// Verify if there may be any other iteration
				if step.Iteration >= step.Retry.Count {
					break
				}

				// Verify if the retry condition is matching
				wd, _ := os.Getwd()
				if wd == "" {
					wd = "/"
				}
				until := step.Retry.Until
				if until == "" {
					until = "passed"
				}
				machine := expressions.CombinedMachines(data.LocalMachine, data.RefSuccessMachine, data.AliasMachine, data.StateMachine, libs.NewFsMachine(os.DirFS("/"), wd))
				expr, err := expressions.CompileAndResolve(until, machine, expressions.FinalizerFail)
				if err != nil {
					fmt.Printf("failed to execute retry condition: %s: %s\n", until, err.Error())
					break
				}
				shouldStop, _ := expr.Static().BoolValue()
				if shouldStop {
					break
				}

				// Continue with the next iteration
				step.Iteration++
				data.PrintHintDetails(step.Ref, constants.InstructionIteration, step.Iteration)
				fmt.Printf("\nExit code: %d • Retrying: attempt #%d (of %d):\n", step.ExitCode, step.Iteration, step.Retry.Count)
			}
		}

		// Save the status after each action
		data.SaveState()
	}

	// Ensure the latest state is saved
	data.SaveState()

	// Stop the container after all the instructions are interpret
	_ = orchestration.Executions.Kill()
	if orchestration.Executions.IsAborted() {
		os.Exit(int(data.CodeAborted))
	} else {
		os.Exit(0)
	}
}
