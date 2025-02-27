package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/utilization/core"

	"github.com/kubeshop/testkube/pkg/utilization"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/obfuscator"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runtime"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

const (
	SensitiveMask             = "****"
	SensitiveVisibleLastChars = 2
	SensitiveMinimumLength    = 4
)

func main() {
	// Force colors
	color.ForceColor()

	// Configure standard output
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	// Configure sensitive data obfuscation
	stdout.SetSensitiveReplacer(obfuscator.ShowLastCharacters(SensitiveMask, SensitiveVisibleLastChars))
	orchestration.Setup.SetSensitiveWordMinimumLength(SensitiveMinimumLength)

	// Prepare empty state file if it doesn't exist
	_, err := os.Stat(constants.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		stdout.Hint(constants.InitStepName, constants.InstructionStart)
		stdoutUnsafe.Print("Creating state...")
		err := os.WriteFile(constants.StatePath, nil, 0777)
		if err != nil {
			stdoutUnsafe.Error(" error\n")
			output.ExitErrorf(constants.CodeInternal, "failed to create state file: %s", err.Error())
		}
		os.Chmod(constants.StatePath, 0777)
		stdoutUnsafe.Print(" done\n")
	} else if err != nil {
		stdout.Hint(constants.InitStepName, constants.InstructionStart)
		stdoutUnsafe.Print("Accessing state...")
		stdoutUnsafe.Error(" error\n")
		output.ExitErrorf(constants.CodeInternal, "cannot access state file: %s", err.Error())
	}

	// Store the instructions in the state if they are provided
	orchestration.Setup.UseBaseEnv()
	stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
	actionGroups := orchestration.Setup.GetActionGroups()
	internalConfig := orchestration.Setup.GetInternalConfig()
	signature := orchestration.Setup.GetSignature()
	containerResources := orchestration.Setup.GetContainerResources()
	if actionGroups != nil {
		stdoutUnsafe.Print("Initializing state...")
		data.GetState().Actions = actionGroups
		data.GetState().InternalConfig = internalConfig
		data.GetState().Signature = signature
		data.GetState().ContainerResources = containerResources
		stdoutUnsafe.Print(" done\n")

		// Release the memory
		actionGroups = nil
	}

	// Distribute the details
	currentContainer := lite.LiteActionContainer{}

	// Ensure there is a group index provided
	if len(os.Args) != 2 {
		output.ExitErrorf(constants.CodeInternal, "invalid arguments provided - expected only one")
	}

	// Determine group index to run
	groupIndex, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		output.ExitErrorf(constants.CodeInputError, "invalid run group passed: %s", err.Error())
	}

	// Handle aborting
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopSignal
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
			// TODO: What about parents of the parents?
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
			// TODO: What about parents of the parents?
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
		output.ExitErrorf(constants.CodeInternal, "Failed to start control server at port %d: %s\n", constants.ControlServerPort, err.Error())
	}

	// Keep a list of paused steps for execution
	delayedPauses := make([]string, 0)

	// Interpret the operations
	actions := state.GetActions(int(groupIndex))
	for i, action := range actions {
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
			err := orchestration.Setup.AdvanceEnv()
			// Attach the error to the next consecutive step
			if err != nil {
				for _, next := range actions[i:] {
					if next.Type() != lite.ActionTypeStart || *next.Start == "" {
						continue
					}
					step := state.GetStep(*next.Start)
					orchestration.Start(step)
					break
				}
				output.ExitErrorf(constants.CodeInputError, err.Error())
			}
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
				output.ExitErrorf(constants.CodeInternal, "failed to determine condition of '%s' step: %s: %v", *action.Start, step.Condition, err.Error())
			}
			if !executable {
				step.SetStatus(constants.StepStatusSkipped)

				// Skip all the children
				for _, v := range state.Steps {
					if slices.Contains(v.Parents, step.Ref) {
						v.SetStatus(constants.StepStatusSkipped)
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
					output.ExitErrorf(constants.CodeInternal, "failed to determine result of '%s' step: %s: %v", *action.End, step.Result, err.Error())
				}
				step.SetStatus(status)
			}
			orchestration.End(step)

		case lite.ActionTypeSetup:
			err := orchestration.Setup.UseEnv(constants.EnvGroupDebug)
			if err != nil {
				output.ExitErrorf(constants.CodeInputError, err.Error())
			}
			stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
			step := state.GetStep(constants.InitStepName)
			err = commands.Setup(*action.Setup)
			if err == nil {
				step.SetStatus(constants.StepStatusPassed)
			} else {
				step.SetStatus(constants.StepStatusFailed)
			}
			orchestration.End(step)
			if err != nil {
				os.Exit(1)
			}

		case lite.ActionTypeExecute:
			// Ensure the latest state before each execute,
			// as it may refer to the state file (Toolkit).
			data.SaveState()

			// Ignore running when the step is already resolved (= skipped)
			step := state.GetStep(action.Execute.Ref)
			if step.IsFinished() {
				continue
			}

			// Ignore when it is aborted
			if orchestration.Executions.IsAborted() {
				step.SetStatus(constants.StepStatusAborted)
				continue
			}

			// Configure the environment
			err := orchestration.Setup.UseCurrentEnv()
			if err != nil {
				output.ExitErrorf(constants.CodeInputError, err.Error())
			}
			if action.Execute.Toolkit {
				serialized, _ := json.Marshal(state.InternalConfig)
				_ = os.Setenv("TK_CFG", string(serialized))
			} else {
				_ = os.Unsetenv("TK_REF")
				_ = os.Unsetenv("TK_CFG")
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
			var hasTimeout, hasOwnTimeout atomic.Bool
			finalizeTimeout := func() {
				// Check timed out steps in leaf
				timedOut := orchestration.GetTimedOut(leaf...)
				if timedOut == nil {
					return
				}

				// Iterate over timed out step
				for _, r := range timedOut {
					r.SetStatus(constants.StepStatusTimeout)
					sub := state.GetSubSteps(r.Ref)
					hasTimeout.Store(true)
					if step.Ref == r.Ref {
						hasOwnTimeout.Store(true)
					}
					for i := range sub {
						if sub[i].IsFinished() {
							continue
						}
						if sub[i].IsStarted() {
							sub[i].SetStatus(constants.StepStatusTimeout)
						} else {
							sub[i].SetStatus(constants.StepStatusSkipped)
						}
					}
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
					step.SetStatus(constants.StepStatusAborted)
					break
				}

				// Register timeouts
				hasTimeout.Store(false)
				hasOwnTimeout.Store(false)
				stopTimeoutWatcher := orchestration.WatchTimeout(finalizeTimeout, leaf...)

				// Run the command
				// WithMetricsRecorder will run a goroutine which will identify the process of the underlying binary which gets executed,
				// it will then scrape the metrics of the process and store them as artifacts in the internal folder.
				config := newMetricsRecorderConfig(step.Ref, action.Execute.Toolkit, containerResources)
				utilization.WithMetricsRecorder(
					config,
					func() {
						commands.Run(*action.Execute, currentContainer)
					},
					scrapeMetricsPostProcessor(config.Dir, step.Ref, data.GetState().InternalConfig),
				)

				// Stop timer listener
				stopTimeoutWatcher()

				// Handle timeout gracefully
				if hasOwnTimeout.Load() {
					orchestration.Executions.ClearAbortedStatus()
				}

				// Ensure there won't be any hanging processes after the command is executed
				_ = orchestration.Executions.Kill()

				// TODO: Handle retry policy in tree independently
				// Verify if there may be any other iteration
				if step.Iteration >= step.Retry.Count || (!hasOwnTimeout.Load() && hasTimeout.Load()) {
					break
				}

				// Verify if the retry condition is matching
				until := step.Retry.Until
				if until == "" {
					until = "passed"
				}
				expr, err := expressions.CompileAndResolve(until, data.LocalMachine, runtime.GetInternalTestWorkflowMachine(), expressions.FinalizerFail)
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
				message := fmt.Sprintf("Exit code: %d", step.ExitCode)
				if hasOwnTimeout.Load() {
					message = "Timed out"
				}
				stdoutUnsafe.Printf("\n%s • Retrying: attempt #%d (of %d):\n", message, step.Iteration, step.Retry.Count)

				// Restart start time for the next iteration to allow retries
				now := time.Now()
				step.StartedAt = &now
			}
		}

		// Save the status after each action
		data.SaveTerminationLog()
	}

	// Ensure the latest state is saved
	data.SaveState()

	// Stop the container after all the instructions are interpret
	_ = orchestration.Executions.Kill()
	if orchestration.Executions.IsAborted() {
		os.Exit(int(constants.CodeAborted))
	} else {
		os.Exit(0)
	}
}

func newMetricsRecorderConfig(stepRef string, skip bool, containerResources testworkflowconfig.ContainerResourceConfig) utilization.Config {
	s := data.GetState()
	metricsDir := filepath.Join(constants.InternalPath, "metrics", stepRef)
	return utilization.Config{
		Dir:  metricsDir,
		Skip: skip,
		ExecutionConfig: utilization.ExecutionConfig{
			Workflow:  s.InternalConfig.Workflow.Name,
			Step:      stepRef,
			Execution: s.InternalConfig.Execution.Id,
		},
		Format: core.FormatInflux,
		ContainerResources: core.ContainerResources{
			Requests: core.ResourceList{
				CPU:    appendSuffixIfNeeded(containerResources.Requests.CPU, "m"),
				Memory: containerResources.Requests.Memory,
			},
			Limits: core.ResourceList{
				CPU:    appendSuffixIfNeeded(containerResources.Limits.CPU, "m"),
				Memory: containerResources.Limits.Memory,
			},
		},
	}
}

func appendSuffixIfNeeded(s, suffix string) string {
	if !strings.HasSuffix(s, suffix) {
		return s + suffix
	}
	return s
}

func scrapeMetricsPostProcessor(path, step string, config testworkflowconfig.InternalConfig) func() error {
	return func() error {
		// Configure the environment
		err := orchestration.Setup.UseCurrentEnv()
		if err != nil {
			return errors.Wrapf(err, "failed to configure environment for scraping metrics: %v", err)
		}
		serialized, _ := json.Marshal(config)
		_ = os.Setenv("TK_CFG", string(serialized))
		_ = os.Setenv("TK_REF", step)
		defer func() {
			_ = os.Unsetenv("TK_CFG")
			_ = os.Unsetenv("TK_REF")
		}()

		// Scrape the metrics to internal storage
		storage := artifacts.InternalStorage()
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return errors.Wrapf(err, "failed to open metrics file %q: %v", path, err)
			}
			defer f.Close()
			path = filepath.Join("metrics", filepath.Base(path))
			if err := storage.SaveFile(path, f, info); err != nil {
				return errors.Wrapf(err, "failed to save stream to %q: %v", path, err)
			}
			return nil
		})
		return err
	}
}
