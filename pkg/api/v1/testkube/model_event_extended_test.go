package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmitter_IsValidEvent_ForTestWorkflow(t *testing.T) {
	t.Run("should pass only events with given selector", func(t *testing.T) {
		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		execution.Workflow.Labels = map[string]string{"test": "1"}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", "test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with not matching selector", func(t *testing.T) {
		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		execution.Workflow.Labels = map[string]string{"test": "2"}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", "test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.False(t, valid)
	})

	t.Run("should pass events without selector", func(t *testing.T) {
		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", "", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should treat empty group as wildcard", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution, GroupId: "env-1"}

		// when
		types, valid := e.Valid("", "", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become events", func(t *testing.T) {
		// given
		execution := &TestWorkflowExecution{}
		e := Event{Type_: EventEndTestWorkflowFailed, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", "", []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, BECOME_TESTWORKFLOW_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, BECOME_TESTWORKFLOW_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become and regular events", func(t *testing.T) {
		// given
		execution := &TestWorkflowExecution{}
		e := Event{Type_: EventEndTestWorkflowFailed, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", "", []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, END_TESTWORKFLOW_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, END_TESTWORKFLOW_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with wrong become events", func(t *testing.T) {
		// given
		execution := &TestWorkflowExecution{}
		e := Event{Type_: EventEndTestWorkflowFailed, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", "", []EventType{BECOME_TESTWORKFLOW_UP_EventType})

		// then
		assert.Nil(t, types)
		assert.False(t, valid)
	})
}

func TestEvent_IsSuccess(t *testing.T) {
	t.Run("should return true for success events", func(t *testing.T) {
		events := map[EventType]bool{
			END_TESTWORKFLOW_FAILED_EventType:  false,
			END_TESTWORKFLOW_SUCCESS_EventType: true,
		}

		for eventType, expected := range events {
			// given
			e := Event{Type_: &eventType}

			// when
			success := e.IsSuccess()

			// then
			assert.Equal(t, expected, success)
		}
	})
}

func TestNewWorkflowEvents_UseExplicitRoutingGroup(t *testing.T) {
	t.Parallel()

	execution := &TestWorkflowExecution{Id: "execution-1", GroupId: "execution-group-1"}

	queueEvent := NewEventQueueTestWorkflow(execution, "env-123")
	startEvent := NewEventStartTestWorkflow(execution, "env-123")

	assert.Equal(t, "env-123", queueEvent.GroupId)
	assert.Equal(t, "env-123", startEvent.GroupId)
}

func TestEvent_Valid_CauseEvents(t *testing.T) {
	withDetails := func(detailsType StatusDetailsType) *TestWorkflowExecution {
		return &TestWorkflowExecution{
			Workflow: &TestWorkflow{Labels: map[string]string{"team": "platform"}},
			Result:   &TestWorkflowResult{StatusDetails: &TestWorkflowStatusDetails{Type_: string(detailsType)}},
		}
	}

	tests := []struct {
		name      string
		eventType *EventType
		execution *TestWorkflowExecution
		selector  string
		types     []EventType
		wantTypes []EventType
		wantValid bool
	}{
		{
			name:      "the infrastructure event matches the not-passed event of an execution failure",
			eventType: EventEndTestWorkflowNotPassed,
			execution: withDetails(StatusDetailsTypeExecutionFailure),
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
			wantTypes: []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
			wantValid: true,
		},
		{
			name:      "the test failure event matches the not-passed event of a step failure",
			eventType: EventEndTestWorkflowNotPassed,
			execution: withDetails(StatusDetailsTypeStepFailure),
			types:     []EventType{END_TESTWORKFLOW_TEST_FAILURE_EventType},
			wantTypes: []EventType{END_TESTWORKFLOW_TEST_FAILURE_EventType},
			wantValid: true,
		},
		{
			name:      "the configuration error event matches the not-passed event of an init failure",
			eventType: EventEndTestWorkflowNotPassed,
			execution: withDetails(StatusDetailsTypeInitFailure),
			types:     []EventType{END_TESTWORKFLOW_CONFIGURATION_ERROR_EventType},
			wantTypes: []EventType{END_TESTWORKFLOW_CONFIGURATION_ERROR_EventType},
			wantValid: true,
		},
		{
			name:      "the failed event of the same execution does not match, so a listener gets one call",
			eventType: EventEndTestWorkflowFailed,
			execution: withDetails(StatusDetailsTypeExecutionFailure),
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
		},
		{
			name:      "the aborted event of the same execution does not match",
			eventType: EventEndTestWorkflowAborted,
			execution: withDetails(StatusDetailsTypeExecutionFailure),
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
		},
		{
			name:      "another type does not match",
			eventType: EventEndTestWorkflowNotPassed,
			execution: withDetails(StatusDetailsTypeStepFailure),
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
		},
		{
			name:      "an execution without status details does not match",
			eventType: EventEndTestWorkflowNotPassed,
			execution: &TestWorkflowExecution{Result: &TestWorkflowResult{}},
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
		},
		{
			name:      "an execution without a result does not match",
			eventType: EventEndTestWorkflowNotPassed,
			execution: &TestWorkflowExecution{},
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
		},
		{
			name:      "a cause event next to the not-passed event matches both on one event",
			eventType: EventEndTestWorkflowNotPassed,
			execution: withDetails(StatusDetailsTypeExecutionFailure),
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType, END_TESTWORKFLOW_NOT_PASSED_EventType},
			wantTypes: []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType, END_TESTWORKFLOW_NOT_PASSED_EventType},
			wantValid: true,
		},
		{
			name:      "the selector still applies to a cause event",
			eventType: EventEndTestWorkflowNotPassed,
			execution: withDetails(StatusDetailsTypeExecutionFailure),
			selector:  "team=qa",
			types:     []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
			wantTypes: []EventType{END_TESTWORKFLOW_INFRASTRUCTURE_FAILURE_EventType},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{Type_: tt.eventType, TestWorkflowExecution: tt.execution}

			types, valid := e.Valid("", tt.selector, tt.types)

			assert.Equal(t, tt.wantTypes, types)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}
