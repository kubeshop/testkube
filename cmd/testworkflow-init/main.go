package main

import (
	"errors"
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
	"github.com/kubeshop/testkube/cmd/testworkflow-init/obfuscator"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func main() {
	// Force colors
	color.ForceColor()

	// Configure standard output
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Configure sensitive data obfuscation
	stdout.SetSensitiveReplacer(obfuscator.ShowLastCharacters("****", 2))
	orchestration.Setup.SetSensitiveWordMinimumLength(4)

	// Prepare empty state file if it doesn't exist
	_, err := os.Stat(data.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		stdout.Hint(data.InitStepName, constants.InstructionStart)
		stdoutUnsafe.Print("Creating state...")
		err := os.WriteFile(data.StatePath, nil, 0777)
		if err != nil {
			stdoutUnsafe.Error(" error\n")
			output.ExitErrorf(data.CodeInternal, "failed to create state file: %s", err.Error())
		}
		os.Chmod(data.StatePath, 0777)
		stdoutUnsafe.Print(" done\n")
	} else if err != nil {
		stdout.Hint(data.InitStepName, constants.InstructionStart)
		stdoutUnsafe.Print("Accessing state...")
		stdoutUnsafe.Error(" error\n")
		output.ExitErrorf(data.CodeInternal, "cannot access state file: %s", err.Error())
	}

	// Store the instructions in the state if they are provided
	orchestration.Setup.UseBaseEnv()
	stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
	actionGroups := orchestration.Setup.GetActionGroups()
	if actionGroups != nil {
		stdoutUnsafe.Print("Initializing state...")
		data.GetState().Actions = actionGroups
		stdoutUnsafe.Print(" done\n")

		// Release the memory
		actionGroups = nil
	}

	// Distribute the details
	currentContainer := lite.LiteActionContainer{}

	// Ensure there is a group index provided
	if len(os.Args) != 2 {
		output.ExitErrorf(data.CodeInternal, "invalid arguments provided - expected only one")
	}

	// Determine group index to run
	groupIndex, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		output.ExitErrorf(data.CodeInputError, "invalid run group passed: %s", err.Error())
	}

	// Handle aborting
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignal
		stdoutUnsafe.Print("The task was aborted.\n")
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
			stdoutUnsafe.Warnf("warn: pause: %s\n", err.Error())
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
			stdoutUnsafe.Warnf("warn: resume: %s\n", err.Error())
		}
		orchestration.Resume(step, ts)
		for _, parentRef := range step.Parents {
			parent := state.GetStep(parentRef)
			orchestration.Resume(parent, ts)
		}
		return err
	}
	controlSrv := control.NewServer(constants.ControlServerPort, control.ServerOptions{
		HandlePause: func(ts time.Time) error {
			return handlePause(ts, state.GetStep(state.CurrentRef))
		},
		HandleResume: func(ts time.Time) error {
			return handleResume(ts, state.GetStep(state.CurrentRef))
		},
	})
	_, err = controlSrv.Listen()
	if err != nil {
		output.ExitErrorf(data.CodeInternal, "Failed to start control server at port %d: %s\n", constants.ControlServerPort, err.Error())
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
			stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
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
				output.ExitErrorf(data.CodeInternal, "failed to determine condition of '%s' step: %s: %v", *action.Start, step.Condition, err.Error())
			}
			if !executable {
				step.SetStatus(data.StepStatusSkipped)

				// Skip all the children
				for _, v := range state.Steps {
					if slices.Contains(v.Parents, step.Ref) {
						v.SetStatus(data.StepStatusSkipped)
					}
				}
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
					output.ExitErrorf(data.CodeInternal, "failed to determine result of '%s' step: %s: %v", *action.End, step.Result, err.Error())
				}
				step.SetStatus(status)
			}
			orchestration.End(step)

		case lite.ActionTypeSetup:
			// TODO: Handle error
			orchestration.Setup.UseEnv("00")
			stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
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

			// Configure the environment
			orchestration.Setup.UseCurrentEnv()
			if !action.Execute.Toolkit {
				_ = os.Unsetenv("TK_REF")
			}

			// List all the parents
			leaf := []*data.StepData{step}
			for i := range step.Parents {
				leaf = append(leaf, state.GetStep(step.Parents[i]))
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

			// Configure timeout finalizer
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
					stdoutUnsafe.Println("Timed out.")
				}
				_ = orchestration.Executions.Kill()

				return
			}

			// Handle immediate timeout
			finalizeTimeout()

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

				// Register timeouts
				stopTimeoutWatcher := orchestration.WatchTimeout(finalizeTimeout, leaf...)

				// Run the command
				commands.Run(*action.Execute, currentContainer)

				// Stop timer listener
				stopTimeoutWatcher()

				// Ensure there won't be any hanging processes after the command is executed
				_ = orchestration.Executions.Kill()

				// TODO: Handle retry policy in tree independently
				// Verify if there may be any other iteration
				if step.Iteration >= step.Retry.Count {
					break
				}

				// Verify if the retry condition is matching
				until := step.Retry.Until
				if until == "" {
					until = "passed"
				}
				expr, err := expressions.CompileAndResolve(until, data.LocalMachine, data.GetInternalTestWorkflowMachine(), expressions.FinalizerFail)
				if err != nil {
					stdout.Printf("failed to execute retry condition: %s: %s\n", until, err.Error())
					break
				}
				shouldStop, _ := expr.Static().BoolValue()
				if shouldStop {
					break
				}

				// Continue with the next iteration
				step.Iteration++
				stdout.HintDetails(step.Ref, constants.InstructionIteration, step.Iteration)
				stdoutUnsafe.Printf("\nExit code: %d • Retrying: attempt #%d (of %d):\n", step.ExitCode, step.Iteration, step.Retry.Count)
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
