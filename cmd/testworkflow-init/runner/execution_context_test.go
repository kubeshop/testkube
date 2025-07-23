package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// Mock implementations
type mockStateAccessor struct {
	mock.Mock
	Steps          map[string]*data.StepData
	CurrentRef     string
	InternalConfig interface{}
}

func (m *mockStateAccessor) GetStep(ref string) *data.StepData {
	args := m.Called(ref)
	if step, ok := args.Get(0).(*data.StepData); ok {
		return step
	}
	return nil
}

func (m *mockStateAccessor) SetCurrentStatus(status string) {
	m.Called(status)
}

func (m *mockStateAccessor) GetSteps() map[string]*data.StepData {
	return m.Steps
}

func (m *mockStateAccessor) GetCurrentRef() string {
	return m.CurrentRef
}

func (m *mockStateAccessor) GetInternalConfig() testworkflowconfig.InternalConfig {
	if cfg, ok := m.InternalConfig.(testworkflowconfig.InternalConfig); ok {
		return cfg
	}
	return testworkflowconfig.InternalConfig{}
}

type mockStdoutWriter struct {
	mock.Mock
}

func (m *mockStdoutWriter) SetSensitiveWords(words []string) {
	m.Called(words)
}

func (m *mockStdoutWriter) HintDetails(ref, name string, value interface{}) {
	m.Called(ref, name, value)
}

func (m *mockStdoutWriter) Printf(format string, args ...interface{}) {
	m.Called(format, args)
}

type mockStdoutUnsafeWriter struct {
	mock.Mock
}

func (m *mockStdoutUnsafeWriter) Printf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *mockStdoutUnsafeWriter) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func TestNewExecutionContext(t *testing.T) {
	// Setup mocks
	mockState := &mockStateAccessor{
		Steps:          make(map[string]*data.StepData),
		InternalConfig: testworkflowconfig.InternalConfig{},
	}
	mockStdout := new(mockStdoutWriter)
	mockStdoutUnsafe := new(mockStdoutUnsafeWriter)
	containerResources := testworkflowconfig.ContainerResourceConfig{
		Requests: testworkflowconfig.ContainerResources{
			CPU: "100m",
		},
	}

	// Create context
	ctx := NewExecutionContext(context.Background(), mockState, mockStdout, mockStdoutUnsafe, containerResources)

	// Verify initialization
	assert.NotNil(t, ctx)
	assert.Equal(t, mockState, ctx.State)
	assert.Equal(t, mockStdout, ctx.Stdout)
	assert.Equal(t, mockStdoutUnsafe, ctx.StdoutUnsafe)
	assert.NotNil(t, ctx.CurrentContainer)
	assert.NotNil(t, ctx.DelayedPauses)
	assert.Empty(t, *ctx.DelayedPauses)
	assert.Empty(t, ctx.Actions)
	assert.Equal(t, containerResources, ctx.ContainerResources)
}

func TestExecutionContext_UpdateContainer(t *testing.T) {
	ctx := &ExecutionContext{
		CurrentContainer: &lite.LiteActionContainer{},
	}

	// Test update with valid container
	newContainer := &lite.LiteActionContainer{
		Config: lite.LiteContainerConfig{},
	}
	ctx.UpdateContainer(newContainer)
	assert.Equal(t, newContainer, ctx.CurrentContainer)

	// Test update with nil (should not change)
	ctx.UpdateContainer(nil)
	assert.Equal(t, newContainer, ctx.CurrentContainer)
}

func TestExecutionContext_DelayedPauses(t *testing.T) {
	t.Run("add delayed pause", func(t *testing.T) {
		pauses := []string{}
		ctx := &ExecutionContext{
			DelayedPauses: &pauses,
		}

		ctx.AddDelayedPause("ref1")
		ctx.AddDelayedPause("ref2")

		assert.Equal(t, []string{"ref1", "ref2"}, *ctx.DelayedPauses)
	})

	t.Run("clear delayed pauses", func(t *testing.T) {
		pauses := []string{"ref1", "ref2"}
		ctx := &ExecutionContext{
			DelayedPauses: &pauses,
		}

		ctx.ClearDelayedPauses()
		assert.Nil(t, *ctx.DelayedPauses)
	})

	t.Run("get delayed pauses", func(t *testing.T) {
		pauses := []string{"ref1", "ref2"}
		ctx := &ExecutionContext{
			DelayedPauses: &pauses,
		}

		result := ctx.GetDelayedPauses()
		assert.Equal(t, []string{"ref1", "ref2"}, result)
	})

	t.Run("get delayed pauses with nil pointer", func(t *testing.T) {
		ctx := &ExecutionContext{
			DelayedPauses: nil,
		}

		result := ctx.GetDelayedPauses()
		assert.Empty(t, result)
	})
}

func TestExecutionContext_Actions(t *testing.T) {
	ctx := &ExecutionContext{}

	// Create test actions
	action1 := &lite.ActionDeclare{Ref: "step1"}
	action2 := &lite.ActionPause{Ref: "step2"}
	actions := []*lite.LiteAction{
		{Declare: action1},
		{Pause: action2},
	}

	// Set actions
	ctx.SetActions(actions)
	assert.Equal(t, actions, ctx.Actions)

	// Verify actions are accessible through the slice
	assert.Len(t, ctx.Actions, 2)
	assert.Equal(t, actions[0], ctx.Actions[0])
	assert.Equal(t, actions[1], ctx.Actions[1])
}

func TestExecutionContext_Integration(t *testing.T) {
	// Create a more realistic context
	step1 := &data.StepData{Ref: "step1"}
	step2 := &data.StepData{Ref: "step2"}

	mockState := &mockStateAccessor{
		Steps: map[string]*data.StepData{
			"step1": step1,
			"step2": step2,
		},
		CurrentRef: "step1",
	}
	mockState.On("GetStep", "step1").Return(step1)
	mockState.On("SetCurrentStatus", "running").Return()

	mockStdout := new(mockStdoutWriter)
	mockStdout.On("SetSensitiveWords", []string{"secret"}).Return()

	mockStdoutUnsafe := new(mockStdoutUnsafeWriter)

	ctx := NewExecutionContext(context.Background(), mockState, mockStdout, mockStdoutUnsafe, testworkflowconfig.ContainerResourceConfig{})

	// Test state access
	step := ctx.State.GetStep("step1")
	assert.Equal(t, step1, step)

	// Test status update
	ctx.State.SetCurrentStatus("running")

	// Test stdout operations
	ctx.Stdout.SetSensitiveWords([]string{"secret"})

	// Verify all expectations
	mockState.AssertExpectations(t)
	mockStdout.AssertExpectations(t)
}
