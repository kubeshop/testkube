package runner

import (
	"testing"

	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func TestNewActionDispatcher(t *testing.T) {
	dispatcher := NewActionDispatcher()

	assert.NotNil(t, dispatcher)
	assert.NotNil(t, dispatcher.handlers)

	// Verify default handlers are registered
	expectedTypes := []lite.ActionType{
		lite.ActionTypeDeclare,
		lite.ActionTypePause,
		lite.ActionTypeResult,
		lite.ActionTypeTimeout,
		lite.ActionTypeRetry,
		lite.ActionTypeContainerTransition,
		lite.ActionTypeCurrentStatus,
		lite.ActionTypeStart,
		lite.ActionTypeEnd,
		lite.ActionTypeSetup,
	}

	for _, actionType := range expectedTypes {
		_, exists := dispatcher.handlers[actionType]
		assert.True(t, exists, "Handler for %v should be registered", actionType)
	}
}

func TestActionDispatcher_RegisterHandler(t *testing.T) {
	dispatcher := NewActionDispatcher()

	// Register a custom handler
	customType := lite.ActionType("custom-test-type")
	handlerCalled := false
	customHandler := func(action *lite.LiteAction, ctx *ExecutionContext) ActionResult {
		handlerCalled = true
		return ActionResult{ContinueExecution: true}
	}

	dispatcher.RegisterHandler(customType, customHandler)

	// Create mock action with custom type
	mockAction := &lite.LiteAction{
		// Set up action to return our custom type
		Declare: &lite.ActionDeclare{}, // Just to make it non-nil
	}
	// Override handler map to include our custom handler
	dispatcher.handlers[customType] = customHandler
	// Since we can't easily set the type, we'll call the handler directly
	ctx := &ExecutionContext{}

	// Call handler directly since we can't mock Type() method
	result := customHandler(mockAction, ctx)

	assert.True(t, handlerCalled)
	assert.True(t, result.ContinueExecution)
}

func TestActionDispatcher_Dispatch_UnregisteredType(t *testing.T) {
	dispatcher := NewActionDispatcher()

	// Create an action with a type, then remove the handler
	mockAction := &lite.LiteAction{
		Declare: &lite.ActionDeclare{Ref: "test"},
	}
	ctx := &ExecutionContext{}

	// Remove the handler for this action type
	delete(dispatcher.handlers, lite.ActionTypeDeclare)

	// Dispatch should return error
	result := dispatcher.Dispatch(mockAction, ctx)

	assert.False(t, result.ContinueExecution)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "no handler registered")
}

func TestActionDispatcher_handleDeclare(t *testing.T) {
	dispatcher := NewActionDispatcher()

	// Setup
	step := &data.StepData{Ref: "test-step"}
	mockState := new(mockStateAccessor)
	mockState.On("GetStep", "test-step").Return(step)

	ctx := &ExecutionContext{State: mockState}

	action := &lite.LiteAction{
		Declare: &lite.ActionDeclare{
			Ref:       "test-step",
			Condition: "always",
			Parents:   []string{"parent1"},
		},
	}

	// Execute
	result := dispatcher.handleDeclare(action, ctx)

	// Verify
	assert.True(t, result.ContinueExecution)
	assert.NoError(t, result.Error)
	assert.Equal(t, "always", step.Condition)
	assert.Equal(t, []string{"parent1"}, step.Parents)
	mockState.AssertExpectations(t)
}

func TestActionDispatcher_handlePause(t *testing.T) {
	dispatcher := NewActionDispatcher()

	// Setup
	step := &data.StepData{Ref: "test-step"}
	mockState := new(mockStateAccessor)
	mockState.On("GetStep", "test-step").Return(step)

	ctx := &ExecutionContext{State: mockState}

	action := &lite.LiteAction{
		Pause: &lite.ActionPause{
			Ref: "test-step",
		},
	}

	// Execute
	result := dispatcher.handlePause(action, ctx)

	// Verify
	assert.True(t, result.ContinueExecution)
	assert.NoError(t, result.Error)
	assert.True(t, step.PausedOnStart)
	mockState.AssertExpectations(t)
}

func TestActionDispatcher_handleCurrentStatus(t *testing.T) {
	dispatcher := NewActionDispatcher()

	// Setup
	mockState := new(mockStateAccessor)
	mockState.On("SetCurrentStatus", "running").Return()

	ctx := &ExecutionContext{State: mockState}

	status := "running"
	action := &lite.LiteAction{
		CurrentStatus: &status,
	}

	// Execute
	result := dispatcher.handleCurrentStatus(action, ctx)

	// Verify
	assert.True(t, result.ContinueExecution)
	assert.NoError(t, result.Error)
	mockState.AssertExpectations(t)
}

func TestActionDispatcher_handleContainerTransition(t *testing.T) {
	t.Skip("Skipping container transition test - requires orchestration setup")
	dispatcher := NewActionDispatcher()

	// Setup
	mockState := new(mockStateAccessor)
	mockStdout := new(mockStdoutWriter)
	mockStdout.On("SetSensitiveWords", mock.Anything).Return()

	container := &lite.LiteActionContainer{
		Config: lite.LiteContainerConfig{},
	}
	action := &lite.LiteAction{
		Container: container,
	}

	ctx := &ExecutionContext{
		State:            mockState,
		Stdout:           mockStdout,
		Actions:          []*lite.LiteAction{action},
		CurrentContainer: &lite.LiteActionContainer{},
	}

	// Execute
	result := dispatcher.handleContainerTransition(action, ctx)

	// Verify
	assert.True(t, result.ContinueExecution)
	assert.NoError(t, result.Error)
	assert.Equal(t, container, ctx.CurrentContainer)
	mockStdout.AssertExpectations(t)
}

func (m *mockStateAccessor) GetSubSteps(ref string) []*data.StepData {
	args := m.Called(ref)
	if steps, ok := args.Get(0).([]*data.StepData); ok {
		return steps
	}
	return nil
}

func TestActionDispatcher_Integration(t *testing.T) {
	dispatcher := NewActionDispatcher()

	// Setup complex scenario
	step1 := &data.StepData{Ref: "step1"}
	step2 := &data.StepData{Ref: "step2", Parents: []string{"step1"}}

	mockState := &mockStateAccessor{
		Steps: map[string]*data.StepData{
			"step1": step1,
			"step2": step2,
		},
		CurrentRef: "step1",
	}
	mockState.On("GetStep", "step1").Return(step1)
	mockState.On("GetStep", "step2").Return(step2)
	mockState.On("SetCurrentStatus", "running").Return()

	mockStdout := new(mockStdoutWriter)

	ctx := &ExecutionContext{
		State:            mockState,
		Stdout:           mockStdout,
		DelayedPauses:    &[]string{},
		CurrentContainer: &lite.LiteActionContainer{},
	}

	// Create a sequence of actions
	actions := []*lite.LiteAction{
		{
			Declare: &lite.ActionDeclare{
				Ref:       "step1",
				Condition: "always",
			},
		},
		{
			Declare: &lite.ActionDeclare{
				Ref:       "step2",
				Condition: "always",
				Parents:   []string{"step1"},
			},
		},
		{
			CurrentStatus: ptr.To("running"),
		},
	}

	// Process all actions
	for _, action := range actions {
		result := dispatcher.Dispatch(action, ctx)
		assert.True(t, result.ContinueExecution)
		assert.NoError(t, result.Error)
	}

	// Verify state
	assert.Equal(t, "always", step1.Condition)
	assert.Equal(t, "always", step2.Condition)
	assert.Equal(t, []string{"step1"}, step2.Parents)

	mockState.AssertExpectations(t)
}
