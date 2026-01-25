package runner

import (
	"fmt"
	"slices"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// ActionResult represents the result of an action execution
type ActionResult struct {
	ContinueExecution bool
	ExitCode          int
	Error             error
	ErrorCode         uint8 // If non-zero, use this code when Error is set
}

// ActionHandler is a function that handles a specific action type
type ActionHandler func(action *lite.LiteAction, ctx *ExecutionContext) ActionResult

// ActionDispatcher handles dispatching actions to their appropriate handlers
type ActionDispatcher struct {
	handlers map[lite.ActionType]ActionHandler
}

// NewActionDispatcher creates a new action dispatcher with default handlers
func NewActionDispatcher() *ActionDispatcher {
	d := &ActionDispatcher{
		handlers: make(map[lite.ActionType]ActionHandler),
	}
	d.registerDefaultHandlers()
	return d
}

// RegisterHandler registers a handler for a specific action type
func (d *ActionDispatcher) RegisterHandler(actionType lite.ActionType, handler ActionHandler) {
	d.handlers[actionType] = handler
}

// Dispatch processes an action using the appropriate handler
func (d *ActionDispatcher) Dispatch(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	handler, exists := d.handlers[action.Type()]
	if !exists {
		return ActionResult{
			ContinueExecution: false,
			Error:             fmt.Errorf("no handler registered for action type: %v", action.Type()),
		}
	}
	return handler(action, ctx)
}

// registerDefaultHandlers registers all the default action handlers
func (d *ActionDispatcher) registerDefaultHandlers() {
	d.RegisterHandler(lite.ActionTypeDeclare, d.handleDeclare)
	d.RegisterHandler(lite.ActionTypePause, d.handlePause)
	d.RegisterHandler(lite.ActionTypeResult, d.handleResult)
	d.RegisterHandler(lite.ActionTypeTimeout, d.handleTimeout)
	d.RegisterHandler(lite.ActionTypeRetry, d.handleRetry)
	d.RegisterHandler(lite.ActionTypeContainerTransition, d.handleContainerTransition)
	d.RegisterHandler(lite.ActionTypeCurrentStatus, d.handleCurrentStatus)
	d.RegisterHandler(lite.ActionTypeStart, d.handleStart)
	d.RegisterHandler(lite.ActionTypeEnd, d.handleEnd)
	d.RegisterHandler(lite.ActionTypeSetup, d.handleSetup)
	d.RegisterHandler(lite.ActionTypeExecute, d.handleExecute)
}

func (d *ActionDispatcher) handleDeclare(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	handleDeclareAction(ctx.State.GetStep(action.Declare.Ref), action.Declare)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handlePause(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	handlePauseAction(ctx.State.GetStep(action.Pause.Ref), action.Pause)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleResult(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	handleResultAction(ctx.State.GetStep(action.Result.Ref), action.Result)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleTimeout(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	handleTimeoutAction(ctx.State.GetStep(action.Timeout.Ref), action.Timeout)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleRetry(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	handleRetryAction(ctx.State.GetStep(action.Retry.Ref), action.Retry)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleContainerTransition(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	currentIndex := -1
	for i := range ctx.Actions {
		if ctx.Actions[i] == action {
			currentIndex = i
			break
		}
	}

	actions := make([]lite.LiteAction, len(ctx.Actions))
	for i, a := range ctx.Actions {
		if a != nil {
			actions[i] = *a
		}
	}

	container, err := handleContainerTransition(action.Container, actions, currentIndex, ctx.State, ctx.Stdout)
	if err != nil {
		return ActionResult{
			ContinueExecution: false,
			Error:             err,
			ErrorCode:         constants.CodeInputError,
		}
	}
	ctx.UpdateContainer(container)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleCurrentStatus(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	ctx.State.SetCurrentStatus(*action.CurrentStatus)
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleStart(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	if *action.Start == "" {
		return ActionResult{ContinueExecution: true}
	}

	step := ctx.State.GetStep(*action.Start)
	orchestration.Start(step)

	executable, err := step.ResolveCondition()
	if err != nil {
		return ActionResult{
			ContinueExecution: false,
			Error:             fmt.Errorf("failed to determine condition of '%s' step: %s: %v", *action.Start, step.Condition, err),
		}
	}
	if !executable {
		step.SetStatus(constants.StepStatusSkipped)

		for _, v := range ctx.State.GetSteps() {
			if slices.Contains(v.Parents, step.Ref) {
				v.SetStatus(constants.StepStatusSkipped)
			}
		}
	}

	if !step.IsFinished() && step.PausedOnStart {
		ctx.AddDelayedPause(ctx.State.GetCurrentRef())
	}

	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleEnd(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	if *action.End == "" {
		return ActionResult{ContinueExecution: true}
	}

	step := ctx.State.GetStep(*action.End)
	if step.Status == nil {
		status, err := step.ResolveResult()
		if err != nil {
			return ActionResult{
				ContinueExecution: false,
				Error:             fmt.Errorf("failed to determine result of '%s' step: %s: %v", *action.End, step.Result, err),
			}
		}
		step.SetStatus(status)
	}
	orchestration.End(step)

	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleSetup(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	success, err := handleSetupAction(action.Setup, ctx)
	if err != nil && !success {
		return ActionResult{
			ContinueExecution: false,
			Error:             err,
			ErrorCode:         constants.CodeInputError,
		}
	}
	if !success {
		return ActionResult{
			ContinueExecution: false,
			ExitCode:          1,
		}
	}
	return ActionResult{ContinueExecution: true}
}

func (d *ActionDispatcher) handleExecute(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
	return handleExecuteAction(action.Execute, ctx)
}
