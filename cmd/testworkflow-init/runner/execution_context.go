package runner

import (
	"context"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// ExecutionContext carries shared state through action execution.
type ExecutionContext struct {
	Context            context.Context
	State              StateAccessor
	Stdout             StdoutWriter
	StdoutUnsafe       StdoutUnsafeWriter
	CurrentContainer   *lite.LiteActionContainer
	DelayedPauses      *[]string
	Actions            []*lite.LiteAction
	ContainerResources testworkflowconfig.ContainerResourceConfig
	InternalConfig     testworkflowconfig.InternalConfig
	Steps              map[string]*data.StepData
}

type StateAccessor interface {
	GetStep(ref string) *data.StepData
	GetSubSteps(ref string) []*data.StepData
	GetSteps() map[string]*data.StepData
	GetCurrentRef() string
	GetInternalConfig() testworkflowconfig.InternalConfig
	SetCurrentStatus(status string)
}

type simpleStateAccessor struct {
	getState func() interface{}
}

func (s *simpleStateAccessor) GetStep(ref string) *data.StepData {
	return data.GetState().GetStep(ref)
}

func (s *simpleStateAccessor) GetSubSteps(ref string) []*data.StepData {
	return data.GetState().GetSubSteps(ref)
}

func (s *simpleStateAccessor) GetSteps() map[string]*data.StepData {
	return data.GetState().Steps
}

func (s *simpleStateAccessor) GetCurrentRef() string {
	return data.GetState().CurrentRef
}

func (s *simpleStateAccessor) GetInternalConfig() testworkflowconfig.InternalConfig {
	return data.GetState().InternalConfig
}

func (s *simpleStateAccessor) SetCurrentStatus(status string) {
	data.GetState().CurrentStatus = status
}

type StdoutWriter interface {
	SetSensitiveWords(words []string)
	HintDetails(ref, name string, value interface{})
	Printf(format string, args ...interface{})
}

type StdoutUnsafeWriter interface {
	Printf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

func NewExecutionContext(
	ctx context.Context,
	state StateAccessor,
	stdout StdoutWriter,
	stdoutUnsafe StdoutUnsafeWriter,
	containerResources testworkflowconfig.ContainerResourceConfig,
) *ExecutionContext {
	return &ExecutionContext{
		Context:            ctx,
		State:              state,
		Stdout:             stdout,
		StdoutUnsafe:       stdoutUnsafe,
		CurrentContainer:   &lite.LiteActionContainer{},
		DelayedPauses:      &[]string{},
		Actions:            []*lite.LiteAction{},
		ContainerResources: containerResources,
		InternalConfig:     state.GetInternalConfig(),
		Steps:              state.GetSteps(),
	}
}

func (ctx *ExecutionContext) UpdateContainer(container *lite.LiteActionContainer) {
	if container != nil {
		ctx.CurrentContainer = container
	}
}

func (ctx *ExecutionContext) AddDelayedPause(ref string) {
	if ctx.DelayedPauses != nil {
		*ctx.DelayedPauses = append(*ctx.DelayedPauses, ref)
	}
}

func (ctx *ExecutionContext) ClearDelayedPauses() {
	if ctx.DelayedPauses != nil {
		*ctx.DelayedPauses = nil
	}
}

func (ctx *ExecutionContext) GetDelayedPauses() []string {
	if ctx.DelayedPauses != nil {
		return *ctx.DelayedPauses
	}
	return []string{}
}

func (ctx *ExecutionContext) SetActions(actions []*lite.LiteAction) {
	ctx.Actions = actions
}
