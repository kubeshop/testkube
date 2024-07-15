package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	// Prepare empty state file if it doesn't exist
	_, err := os.Stat(data.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		data.PrintHint(data.InitStepName, constants.InstructionStart)
		fmt.Print("Creating state...")
		err := os.WriteFile(data.StatePath, nil, 0777)
		if err != nil {
			fmt.Println(ui.Red(" error"))
			data.Failf(data.CodeInternal, "failed to create state file: %s", err.Error())
		}
		fmt.Println(" done")
	} else if err != nil {
		data.PrintHint(data.InitStepName, constants.InstructionStart)
		fmt.Print("Accessing state...")
		fmt.Println(ui.Red(" error"))
		data.Failf(data.CodeInternal, "cannot access state file: %s", err.Error())
	}

	// Store the instructions in the state if they are provided
	instructions := os.Getenv(fmt.Sprintf("_01_%s", constants.EnvInstructions))
	if instructions != "" {
		fmt.Print("Initializing state...")
		err = json.Unmarshal([]byte(instructions), &data.GetState().Actions)
		if err != nil {
			fmt.Println(ui.Red(" error"))
			data.Failf(data.CodeInternal, "failed to read the actions from Pod: %s", err.Error())
		}
		fmt.Println(" done")

		// Release the memory
		instructions = ""
		_ = os.Unsetenv(constants.EnvInstructions)
	}

	// Distribute the details
	currentContainer := testworkflowprocessor.ActionContainer{}

	// Ensure there is a group index provided
	if len(os.Args) != 2 {
		data.Failf(data.CodeInternal, "invalid arguments provided - expected only one")
	}

	// Determine group index to run
	groupIndex, err := strconv.ParseInt(os.Args[1], 10, 32)
	if err != nil {
		data.Failf(data.CodeInputError, "invalid run group passed: %s", err.Error())
	}

	// Keep a list of paused steps for execution
	delayedPauses := make([]string, 0)

	// Get the list of operations
	state := data.GetState()
	for _, action := range state.GetActions(int(groupIndex)) {
		if action.Declare != nil {
			state.SetCondition(action.Declare.Ref, action.Declare.Condition)
			state.SetParents(action.Declare.Ref, action.Declare.Parents)
		} else if action.Pause != nil {
			state.SetPause(action.Pause.Ref, true)
		} else if action.Result != nil {
			state.SetResult(action.Result.Ref, action.Result.Value)
		} else if action.Timeout != nil {
			state.SetTimeout(action.Timeout.Ref, action.Timeout.Timeout)
		} else if action.CurrentStatus != nil {
			state.SetCurrentStatus(*action.CurrentStatus)
		} else if action.Setup != nil {
			// TODO: Handle error
			orchestration.Setup.UseEnv("00")
			commands.Setup(*action.Setup)
			orchestration.End(data.InitStepName, data.StepStatusPassed)
		} else if action.Container != nil {
			orchestration.Setup.SetConfig(action.Container.Config)
			orchestration.Setup.AdvanceEnv()
			currentContainer = *action.Container
		} else if action.Execute != nil {
			// Ignore running when the step is already resolved (= skipped)
			step := state.GetStep(action.Execute.Ref)
			if step.Status != nil {
				return
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
				orchestration.Pause(action.Execute.Ref)
				// // TODO: Wait for resume
				//orchestration.Resume(action.Execute.Ref)
			}

			commands.Run(*action.Execute, currentContainer)
		} else if action.Start != nil {
			if *action.Start == "" {
				continue
			}
			orchestration.Start(*action.Start)

			step := state.GetStep(*action.Start)
			expr, err := data.Expression(step.Condition, data.RefSuccessMachine)
			if err != nil {
				panic(fmt.Sprintf("failed to determine condition of '%s' step: %s: %s", *action.Start, step.Condition, err.Error()))
			}
			result, err := expr.BoolValue()
			if !result {
				step.Status = common.Ptr(data.StepStatusSkipped)
				// TODO: Should it immediately inform outside about skip?
			}

			// Delay the pause until next children execution
			if step.Status == nil && step.Paused {
				delayedPauses = append(delayedPauses, state.CurrentRef)
			}
		} else if action.End != nil {
			if *action.End == "" {
				continue
			}
			step := state.GetStep(*action.End)
			if step.Status == nil {
				if step.Result == "" {
					panic(fmt.Sprintf("missing definition of '%s' step success", *action.End))
				}
				expr, err := data.Expression(step.Result, data.RefSuccessMachine)
				if err != nil {
					panic(fmt.Sprintf("failed to determine result of '%s' step: %s: %s", *action.End, step.Result, err.Error()))
				}
				result, err := expr.BoolValue()
				if result {
					step.Status = common.Ptr(data.StepStatusPassed)
				} else {
					step.Status = common.Ptr(data.StepStatusFailed)
				}
			}
			orchestration.End(*action.End, *step.Status)
		} else {
			serialized, _ := json.Marshal(action)
			data.Failf(data.CodeInternal, "Unsupported instruction: %s", string(serialized))
		}
	}

	// Save the data
	data.SaveState()
	data.SaveTerminationLog()
}

//func main() {
//	if len(os.Args) < 2 {
//		output.Failf(output.CodeInputError, "missing step reference")
//	}
//	data.Step.Ref = os.Args[1]
//
//	now := time.Now()
//
//	// Load shared state
//	data.LoadState()
//
//	// Initialize space for parsing args
//	config := map[string]string{}
//	computed := []string(nil)
//	conditions := []data.Rule(nil)
//	resulting := []data.Rule(nil)
//	timeouts := []data.Timeout(nil)
//	paused := false
//	toolkit := false
//	args := []string(nil)
//
//	// Read arguments into the base data
//	for i := 2; i < len(os.Args); i += 2 {
//		if i+1 == len(os.Args) {
//			break
//		}
//		switch os.Args[i] {
//		case constants.ArgSeparator:
//			args = os.Args[i+1:]
//			i = len(os.Args)
//		case constants.ArgInit, constants.ArgInitLong:
//			data.Step.InitStatus = os.Args[i+1]
//		case constants.ArgCondition, constants.ArgConditionLong:
//			v := strings.SplitN(os.Args[i+1], "=", 2)
//			refs := strings.Split(v[0], ",")
//			if len(v) == 2 {
//				conditions = append(conditions, data.Rule{Expr: v[1], Refs: refs})
//			} else {
//				conditions = append(conditions, data.Rule{Expr: "true", Refs: refs})
//			}
//		case constants.ArgResult, constants.ArgResultLong:
//			v := strings.SplitN(os.Args[i+1], "=", 2)
//			refs := strings.Split(v[0], ",")
//			if len(v) == 2 {
//				resulting = append(resulting, data.Rule{Expr: v[1], Refs: refs})
//			} else {
//				resulting = append(resulting, data.Rule{Expr: "true", Refs: refs})
//			}
//		case constants.ArgTimeout, constants.ArgTimeoutLong:
//			v := strings.SplitN(os.Args[i+1], "=", 2)
//			if len(v) == 2 {
//				timeouts = append(timeouts, data.Timeout{Ref: v[0], Duration: v[1]})
//			} else {
//				timeouts = append(timeouts, data.Timeout{Ref: v[0], Duration: ""})
//			}
//		case constants.ArgComputeEnv, constants.ArgComputeEnvLong:
//			computed = append(computed, strings.Split(os.Args[i+1], ",")...)
//		case constants.ArgPaused, constants.ArgPausedLong:
//			paused = true
//			i--
//		case constants.ArgNegative, constants.ArgNegativeLong:
//			config["negative"] = os.Args[i+1]
//		case constants.ArgWorkingDir, constants.ArgWorkingDirLong:
//			wd, err := filepath.Abs(os.Args[i+1])
//			if err == nil {
//				_ = os.MkdirAll(wd, 0755)
//				err = os.Chdir(wd)
//			} else {
//				_ = os.MkdirAll(wd, 0755)
//				err = os.Chdir(os.Args[i+1])
//			}
//			if err != nil {
//				fmt.Printf("warning: error using %s as working director: %s\n", os.Args[i+1], err.Error())
//			}
//		case constants.ArgRetryCount:
//			config["retryCount"] = os.Args[i+1]
//		case constants.ArgRetryUntil:
//			config["retryUntil"] = os.Args[i+1]
//		case constants.ArgToolkit, constants.ArgToolkitLong:
//			toolkit = true
//			i--
//		case constants.ArgDebug:
//			config["debug"] = os.Args[i+1]
//		default:
//			output.Failf(output.CodeInputError, "unknown parameter: %s", os.Args[i])
//		}
//	}
//
//	// Clean up unnecessary variables for non-toolkit containers
//	if !toolkit {
//		_ = os.Unsetenv("TK_REF")
//	}
//
//	// Configure PWD variable, to make it similar to shell environment variables
//	if os.Getenv("PWD") == "" {
//		cwd, err := os.Getwd()
//		if err == nil {
//			_ = os.Setenv("PWD", cwd)
//		}
//	}
//
//	// Compute environment variables
//	for _, name := range computed {
//		initial := os.Getenv(name)
//		value, err := data.Template(initial)
//		if err != nil {
//			output.Failf(output.CodeInputError, `resolving "%s" environment variable: %s: %s`, name, initial, err.Error())
//		}
//		_ = os.Setenv(name, value)
//	}
//
//	// Compute conditional steps - ignore errors initially, as the may be dependent on themselves
//	data.Iterate(conditions, func(c data.Rule) bool {
//		expr, err := data.Expression(c.Expr)
//		if err != nil {
//			return false
//		}
//		v, _ := expr.BoolValue()
//		if !v {
//			for _, r := range c.Refs {
//				data.State.GetStep(r).Skip(now)
//			}
//		}
//		return true
//	})
//
//	// Fail invalid conditional steps
//	for _, c := range conditions {
//		_, err := data.Expression(c.Expr)
//		if err != nil {
//			output.Failf(output.CodeInputError, "broken condition for refs: %s: %s: %s", strings.Join(c.Refs, ", "), c.Expr, err.Error())
//		}
//	}
//
//	// Start all acknowledged steps
//	for _, f := range resulting {
//		for _, r := range f.Refs {
//			if r != "" {
//				data.State.GetStep(r).Start(now)
//			}
//		}
//	}
//	for _, t := range timeouts {
//		if t.Ref != "" {
//			data.State.GetStep(t.Ref).Start(now)
//		}
//	}
//	data.State.GetStep(data.Step.Ref).Start(now)
//
//	// Register timeouts
//	for _, t := range timeouts {
//		err := data.State.GetStep(t.Ref).SetTimeoutDuration(now, t.Duration)
//		if err != nil {
//			output.Failf(output.CodeInputError, "broken timeout for ref: %s: %s: %s", t.Ref, t.Duration, err.Error())
//		}
//	}
//
//	// Save the resulting conditions
//	data.Config.Resulting = resulting
//
//	// Don't call further if the step is already skipped
//	if data.State.GetStep(data.Step.Ref).Status == data.StepStatusSkipped {
//		if data.Config.Debug {
//			fmt.Printf("Skipped.\n")
//		}
//		data.Finish()
//	}
//
//	// Handle pausing
//	if paused {
//		data.Step.Pause(now)
//	}
//
//	// Load the rest of the configuration
//	var err error
//	for k, v := range config {
//		config[k], err = data.Template(v)
//		if err != nil {
//			output.Failf(output.CodeInputError, `resolving "%s" param: %s: %s`, k, v, err.Error())
//		}
//	}
//	data.LoadConfig(config)
//
//	// Compute templates in the cmd/args
//	original := slices.Clone(args)
//	for i := range args {
//		args[i], err = data.Template(args[i])
//		if err != nil {
//			output.Failf(output.CodeInputError, `resolving command: %s: %s`, shellquote.Join(original...), err.Error())
//		}
//	}
//
//	// Fail when there is nothing to run
//	if len(args) == 0 {
//		output.Failf(output.CodeNoCommand, "missing command to run")
//	}
//
//	// Handle aborting
//	stopSignal := make(chan os.Signal, 1)
//	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
//	go func() {
//		<-stopSignal
//		fmt.Println("The task was aborted.")
//		data.Step.Status = data.StepStatusAborted
//		data.Step.ExitCode = output.CodeAborted
//		data.Finish()
//	}()
//
//	// Handle timeouts.
//	// Ignores time when the step was paused.
//	for _, t := range timeouts {
//		go func(ref string) {
//			start := now
//			timeout := data.State.GetStep(ref).TimeoutAt.Sub(start)
//			for {
//				time.Sleep(timeout)
//				took := data.Step.Took(start)
//				if took < timeout {
//					timeout -= took
//					continue
//				}
//				fmt.Printf("Timed out.\n")
//				data.State.GetStep(ref).SetStatus(data.StepStatusTimeout)
//				data.Step.Status = data.StepStatusTimeout
//				data.Step.ExitCode = output.CodeTimeout
//				data.Finish()
//			}
//		}(t.Ref)
//	}
//
//	// Run the control server
//	controlSrv := control.NewServer(constants.ControlServerPort, data.Step)
//	_, err = controlSrv.Listen()
//	if err != nil {
//		fmt.Printf("Failed to start control server at port %d: %s\n", constants.ControlServerPort, err.Error())
//		os.Exit(int(output.CodeInternal))
//	}
//
//	// Start the task
//	data.Step.Executed = true
//	run.Run(args[0], args[1:])
//
//	os.Exit(0)
//}
