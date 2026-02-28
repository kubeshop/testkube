package runner

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/obfuscator"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/output"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// Security constants for sensitive data obfuscation.
const (
	SensitiveMask             = "****" // Replaces sensitive content
	SensitiveVisibleLastChars = 2      // Shows last N chars for identification
	SensitiveMinimumLength    = 4      // Min length to consider sensitive
)

// RunInit executes the test workflow step for the specified group index.
// Group 0 runs setup actions, groups 1+ run test steps.
// Returns 0 on success, 137 on abort, or other exit codes on failure.
func RunInit(groupIndex int) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	return RunInitWithContext(ctx, groupIndex)
}

// RunInitWithContext executes a test workflow step with explicit context control.
// Exported for testing to allow context injection.
func RunInitWithContext(ctx context.Context, groupIndex int) (int, error) {
	stdout := output.Std
	stdoutUnsafe := stdout.Direct()

	stdout.SetSensitiveReplacer(obfuscator.ShowLastCharacters(SensitiveMask, SensitiveVisibleLastChars))
	orchestration.Setup.SetSensitiveWordMinimumLength(SensitiveMinimumLength)

	stateManager := NewStateManager(stdout, stdoutUnsafe)

	if err := stateManager.EnsureStateFile(); err != nil {
		return int(constants.CodeInternal), err
	}

	if err := stateManager.LoadInitialState(); err != nil {
		return int(constants.CodeInternal), err
	}

	currentContainer := lite.LiteActionContainer{}

	setupAbortHandler(ctx)

	state := data.GetState()

	getCurrentRef := func() string { return data.GetState().CurrentRef }
	stopControlServer, err := setupControlServer(state.GetStep, getCurrentRef, stdoutUnsafe)
	if err != nil {
		return int(constants.CodeInternal), err
	}
	defer func() {
		if stopControlServer != nil {
			stopControlServer()
		}
	}()

	containerResources := state.ContainerResources
	stateWrapper := &simpleStateAccessor{getState: func() interface{} { return state }}
	execCtx := NewExecutionContext(ctx, stateWrapper, stdout, stdoutUnsafe, containerResources)
	execCtx.UpdateContainer(&currentContainer)

	dispatcher := NewActionDispatcher()

	actions, err := state.GetActions(int(groupIndex))
	if err != nil {
		return int(constants.CodeInputError), fmt.Errorf("failed to get actions: %w", err)
	}
	actionPtrs := make([]*lite.LiteAction, len(actions))
	for i := range actions {
		actionPtrs[i] = &actions[i]
	}
	execCtx.SetActions(actionPtrs)

	for i := range actions {
		select {
		case <-ctx.Done():
			return int(constants.CodeAborted), nil
		default:
		}

		result := dispatcher.Dispatch(&actions[i], execCtx)

		if !result.ContinueExecution {
			if result.Error != nil {
				code := constants.CodeInternal
				if result.ErrorCode != 0 {
					code = result.ErrorCode
				}
				return int(code), result.Error
			}
			if result.ExitCode != 0 {
				return result.ExitCode, nil
			}
		}

		data.SaveTerminationLog()
	}

	data.SaveState()

	_ = orchestration.Executions.Kill()
	if orchestration.Executions.IsAborted() {
		return int(constants.CodeAborted), nil
	} else {
		return 0, nil
	}
}

// setupAbortHandler sets up context-based abort handling
func setupAbortHandler(ctx context.Context) {
	go func() {
		<-ctx.Done()
		orchestration.Executions.Abort()
	}()
}

// setupControlServer sets up the control server for pause/resume functionality
// Returns the stop function and any error that occurred during setup
func setupControlServer(getStep func(string) *data.StepData, getCurrentRef func() string, stdoutUnsafe interface {
	Warnf(format string, args ...interface{})
}) (func(), error) {
	handlePause := func(ts time.Time, step *data.StepData) error {
		if step.PausedStart != nil {
			return nil
		}
		err := orchestration.Executions.Pause()
		if err != nil {
			stdoutUnsafe.Warnf("warn: pause: %s\n", err.Error())
		}
		orchestration.Pause(step, *step.StartedAt)
		for _, parentRef := range step.Parents {
			parent := getStep(parentRef)
			// TODO: What about parents of the parents?
			orchestration.Pause(parent, *step.StartedAt)
		}
		return err
	}

	handleResume := func(ts time.Time, step *data.StepData) error {
		if step.PausedStart == nil {
			return nil
		}
		err := orchestration.Executions.Resume()
		if err != nil {
			stdoutUnsafe.Warnf("warn: resume: %s\n", err.Error())
		}
		orchestration.Resume(step, ts)
		for _, parentRef := range step.Parents {
			parent := getStep(parentRef)
			// TODO: What about parents of the parents?
			orchestration.Resume(parent, ts)
		}
		return err
	}

	controlSrv := control.NewServer(constants.ControlServerPort, control.ServerOptions{
		HandlePause: func(ts time.Time) error {
			return handlePause(ts, getStep(getCurrentRef()))
		},
		HandleResume: func(ts time.Time) error {
			return handleResume(ts, getStep(getCurrentRef()))
		},
	})

	stopControlServer, err := controlSrv.Listen()
	if err != nil {
		return nil, fmt.Errorf("failed to start control server at port %d: %s", constants.ControlServerPort, err.Error())
	}

	return stopControlServer, nil
}
