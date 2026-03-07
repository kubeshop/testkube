package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runtime"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
	"github.com/kubeshop/testkube/pkg/utilization"
	"github.com/kubeshop/testkube/pkg/utilization/core"
)

// handleDeclareAction processes declare actions
func handleDeclareAction(step *data.StepData, action *lite.ActionDeclare) {
	step.SetCondition(action.Condition).SetParents(action.Parents)
}

// handlePauseAction processes pause actions
func handlePauseAction(step *data.StepData, action *lite.ActionPause) {
	step.SetPausedOnStart(true)
}

// handleResultAction processes result actions
func handleResultAction(step *data.StepData, action *lite.ActionResult) {
	step.SetResult(action.Value)
}

// handleTimeoutAction processes timeout actions
func handleTimeoutAction(step *data.StepData, action *lite.ActionTimeout) {
	step.SetTimeout(action.Timeout)
}

// handleRetryAction processes retry actions
func handleRetryAction(step *data.StepData, action *lite.ActionRetry) {
	step.SetRetryPolicy(data.RetryPolicy{
		Count: action.Count,
		Until: action.Until,
	})
}

// handleContainerTransition processes container transition actions
func handleContainerTransition(container *lite.LiteActionContainer, actions []lite.LiteAction, currentIndex int, state interface{ GetStep(string) *data.StepData }, stdout interface{ SetSensitiveWords([]string) }) (*lite.LiteActionContainer, error) {
	orchestration.Setup.SetConfig(container.Config)
	err := orchestration.Setup.AdvanceEnv()
	if err != nil {
		for _, next := range actions[currentIndex:] {
			if next.Type() != lite.ActionTypeStart || *next.Start == "" {
				continue
			}
			step := state.GetStep(*next.Start)
			orchestration.Start(step)
			break
		}
		return nil, err
	}
	stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
	return container, nil
}

// handleSetupAction processes ActionTypeSetup (Group 0).
// Copies binaries to shared volume and initializes environment.
func handleSetupAction(action *lite.ActionSetup, ctx *ExecutionContext) (bool, error) {
	select {
	case <-ctx.Context.Done():
		return false, ctx.Context.Err()
	default:
	}

	err := orchestration.Setup.UseEnv(constants.EnvGroupDebug)
	if err != nil {
		return false, err
	}
	ctx.Stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())

	step := ctx.State.GetStep(constants.InitStepName)
	err = commands.Setup(*action)
	if err == nil {
		step.SetStatus(constants.StepStatusPassed)
	} else {
		step.SetStatus(constants.StepStatusFailed)
	}
	orchestration.End(step)

	return err == nil, err
}

// handleExecuteAction executes ActionTypeExecute with retry logic, timeout handling,
// and metrics collection. Handles both toolkit and regular command execution.
func handleExecuteAction(action *lite.ActionExecute, ctx *ExecutionContext) ActionResult {
	data.SaveState()

	step := ctx.State.GetStep(action.Ref)
	if step.IsFinished() {
		return ActionResult{ContinueExecution: true}
	}

	if orchestration.Executions.IsAborted() {
		step.SetStatus(constants.StepStatusAborted)
		return ActionResult{ContinueExecution: true}
	}

	err := orchestration.Setup.UseCurrentEnv()
	if err != nil {
		return ActionResult{
			ContinueExecution: false,
			Error:             err,
			ErrorCode:         constants.CodeInputError,
		}
	}
	if action.Toolkit {
		serialized, _ := json.Marshal(ctx.InternalConfig)
		_ = os.Setenv("TK_CFG", string(serialized))
	} else {
		_ = os.Unsetenv("TK_REF")
		_ = os.Unsetenv("TK_CFG")
	}

	leaf := []*data.StepData{step}
	for i := range step.Parents {
		leaf = append(leaf, ctx.State.GetStep(step.Parents[i]))
	}

	paused := make([]string, 0)
	delayedPauses := ctx.GetDelayedPauses()
	if slices.Contains(delayedPauses, action.Ref) {
		paused = append(paused, action.Ref)
	}
	for _, parentRef := range step.Parents {
		if slices.Contains(delayedPauses, parentRef) {
			paused = append(paused, parentRef)
		}
	}

	if len(paused) > 0 {
		ctx.ClearDelayedPauses()
		_ = handlePause(*step.StartedAt, step, ctx)
	}

	var hasTimeout, hasOwnTimeout atomic.Bool
	finalizeTimeout := func() {
		timedOut := orchestration.GetTimedOut(leaf...)
		if timedOut == nil {
			return
		}

		for _, r := range timedOut {
			r.SetStatus(constants.StepStatusTimeout)
			sub := ctx.State.GetSubSteps(r.Ref)
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
	}

	finalizeTimeout()

	if step.IsFinished() {
		return ActionResult{ContinueExecution: true}
	}

	for {
		select {
		case <-ctx.Context.Done():
			step.SetStatus(constants.StepStatusAborted)
			return ActionResult{ContinueExecution: false, ExitCode: int(constants.CodeAborted)}
		default:
		}

		step.Status = nil

		if orchestration.Executions.IsAborted() {
			step.SetStatus(constants.StepStatusAborted)
			break
		}

		hasTimeout.Store(false)
		hasOwnTimeout.Store(false)
		stopTimeoutWatcher := orchestration.WatchTimeout(finalizeTimeout, leaf...)

		// WithMetricsRecorder will run a goroutine which will identify the process of the underlying binary which gets executed,
		// it will then scrape the metrics of the process and store them as artifacts in the internal folder.
		config := newMetricsRecorderConfig(step.Ref, action.Toolkit, ctx.ContainerResources)
		utilization.WithMetricsRecorder(
			config,
			func() {
				commands.Run(ctx.Context, *action, *ctx.CurrentContainer)
			},
			scrapeMetricsPostProcessor(config.Dir, step.Ref, ctx.InternalConfig),
		)

		stopTimeoutWatcher()

		if hasOwnTimeout.Load() {
			orchestration.Executions.ClearAbortedStatus()
		}

		_ = orchestration.Executions.Kill()

		if !shouldRetry(step, hasTimeout.Load(), hasOwnTimeout.Load(), ctx.Stdout) {
			break
		}

		step.Iteration++
		ctx.Stdout.HintDetails(step.Ref, constants.InstructionIteration, step.Iteration)
		message := fmt.Sprintf("Exit code: %d", step.ExitCode)
		if hasOwnTimeout.Load() {
			message = "Timed out"
		}
		ctx.StdoutUnsafe.Printf("\n%s • Retrying: attempt #%d (of %d):\n", message, step.Iteration, step.Retry.Count)

		now := time.Now()
		step.StartedAt = &now
	}

	return ActionResult{ContinueExecution: true}
}

// handlePause handles pausing of a step and its parents
func handlePause(ts time.Time, step *data.StepData, ctx *ExecutionContext) error {
	if step.PausedStart != nil {
		return nil
	}
	err := orchestration.Executions.Pause()
	if err != nil {
		ctx.StdoutUnsafe.Warnf("warn: pause: %s\n", err.Error())
	}
	orchestration.Pause(step, *step.StartedAt)
	for _, parentRef := range step.Parents {
		parent := ctx.State.GetStep(parentRef)
		orchestration.Pause(parent, *step.StartedAt)
	}
	return err
}

// newMetricsRecorderConfig creates configuration for metrics recording
func newMetricsRecorderConfig(stepRef string, skip bool, containerResources testworkflowconfig.ContainerResourceConfig) utilization.Config {
	s := data.GetState()
	metricsDir := filepath.Join(constants.InternalPath, "metrics", stepRef)
	return utilization.Config{
		Dir:  metricsDir,
		Skip: skip || s.InternalConfig.Worker.DisableResourceMetrics,
		ExecutionConfig: utilization.ExecutionConfig{
			Workflow:  s.InternalConfig.Workflow.Name,
			Step:      stepRef,
			Execution: s.InternalConfig.Execution.Id,
			Resource:  s.InternalConfig.Resource.Id,
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
	if s == "" {
		return s
	}
	if strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
}

// scrapeMetricsPostProcessor returns a function that processes scraped metrics
func scrapeMetricsPostProcessor(path, step string, config testworkflowconfig.InternalConfig) func() error {
	return func() error {
		err := orchestration.Setup.UseCurrentEnv()
		if err != nil {
			return fmt.Errorf("failed to configure environment for scraping metrics: %v", err)
		}
		serialized, _ := json.Marshal(config)
		_ = os.Setenv("TK_CFG", string(serialized))
		_ = os.Setenv("TK_REF", step)
		defer func() {
			_ = os.Unsetenv("TK_CFG")
			_ = os.Unsetenv("TK_REF")
		}()

		storage, err := artifacts.InternalStorage()
		if err != nil {
			return fmt.Errorf("failed to create internal storage for metrics: %v", err)
		}
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open metrics file %q: %v", path, err)
			}
			defer f.Close()
			path = filepath.Join("metrics", filepath.Base(path))
			if err := storage.SaveFile(path, f, info); err != nil {
				return fmt.Errorf("failed to save stream to %q: %v", path, err)
			}
			return nil
		})
		return err
	}
}

// shouldRetry determines if a step should be retried based on conditions
func shouldRetry(step *data.StepData, hasTimeout bool, hasOwnTimeout bool, stdout interface {
	Printf(format string, args ...interface{})
}) bool {
	if step.Iteration >= step.Retry.Count || (!hasOwnTimeout && hasTimeout) {
		return false
	}

	until := step.Retry.Until
	if until == "" {
		until = "passed"
	}
	expr, err := expressions.CompileAndResolve(until, data.LocalMachine, runtime.GetInternalTestWorkflowMachine(), expressions.FinalizerFail)
	if err != nil {
		stdout.Printf("failed to execute retry condition: %s: %s\n", until, err.Error())
		return false
	}
	shouldStop, _ := expr.Static().BoolValue()
	return !shouldStop
}
