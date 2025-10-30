package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmitter_IsValidEvent_ForTest(t *testing.T) {
	t.Parallel()

	t.Run("should pass only events with given selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedExecution()
		execution.Labels = map[string]string{"test": "1"}
		e := Event{Type_: EventStartTest, TestExecution: execution}

		// when
		types, valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TEST_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with not matching selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedExecution()
		execution.Labels = map[string]string{"test": "2"}
		e := Event{Type_: EventStartTest, TestExecution: execution}

		// when
		types, valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TEST_EventType}, types)
		assert.False(t, valid)
	})

	t.Run("should pass events without selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedExecution()
		e := Event{Type_: EventStartTest, TestExecution: execution}

		// when
		types, valid := e.Valid("", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TEST_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedExecution()
		e := Event{Type_: EventEndTestFailed, TestExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TEST_DOWN_EventType, BECOME_TEST_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TEST_DOWN_EventType, BECOME_TEST_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become and regular events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedExecution()
		e := Event{Type_: EventEndTestFailed, TestExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TEST_DOWN_EventType, END_TEST_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TEST_DOWN_EventType, END_TEST_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with wrong become events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedExecution()
		e := Event{Type_: EventEndTestFailed, TestExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TEST_UP_EventType})

		// then
		assert.Nil(t, types)
		assert.False(t, valid)
	})
}

func TestEmitter_IsValidEvent_ForTestSuite(t *testing.T) {
	t.Parallel()

	t.Run("should pass only events with given selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedTestSuiteExecution("", "")
		execution.Labels = map[string]string{"test": "1"}
		e := Event{Type_: EventStartTestSuite, TestSuiteExecution: execution}

		// when
		types, valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTSUITE_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with not matching selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedTestSuiteExecution("", "")
		execution.Labels = map[string]string{"test": "2"}
		e := Event{Type_: EventStartTestSuite, TestSuiteExecution: execution}

		// when
		types, valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTSUITE_EventType}, types)
		assert.False(t, valid)
	})

	t.Run("should pass events without selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedTestSuiteExecution("", "")
		e := Event{Type_: EventStartTestSuite, TestSuiteExecution: execution}

		// when
		types, valid := e.Valid("", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTSUITE_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedTestSuiteExecution("", "")
		e := Event{Type_: EventEndTestSuiteFailed, TestSuiteExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TESTSUITE_DOWN_EventType, BECOME_TESTSUITE_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TESTSUITE_DOWN_EventType, BECOME_TESTSUITE_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become and regular events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedTestSuiteExecution("", "")
		e := Event{Type_: EventEndTestSuiteFailed, TestSuiteExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TESTSUITE_DOWN_EventType, END_TESTSUITE_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TESTSUITE_DOWN_EventType, END_TESTSUITE_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with wrong become events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := NewQueuedTestSuiteExecution("", "")
		e := Event{Type_: EventEndTestSuiteFailed, TestSuiteExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TESTSUITE_UP_EventType})

		// then
		assert.Nil(t, types)
		assert.False(t, valid)
	})
}

func TestEmitter_IsValidEvent_ForTestWorkflow(t *testing.T) {
	t.Parallel()

	t.Run("should pass only events with given selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		execution.Workflow.Labels = map[string]string{"test": "1"}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with not matching selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		execution.Workflow.Labels = map[string]string{"test": "2"}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.False(t, valid)
	})

	t.Run("should pass events without selector", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{Workflow: &TestWorkflow{}}
		e := Event{Type_: EventStartTestWorkflow, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", AllEventTypes)

		// then
		assert.Equal(t, []EventType{START_TESTWORKFLOW_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{}
		e := Event{Type_: EventEndTestWorkflowFailed, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, BECOME_TESTWORKFLOW_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, BECOME_TESTWORKFLOW_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should pass events with become and regular events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{}
		e := Event{Type_: EventEndTestWorkflowFailed, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, END_TESTWORKFLOW_FAILED_EventType})

		// then
		assert.Equal(t, []EventType{BECOME_TESTWORKFLOW_DOWN_EventType, END_TESTWORKFLOW_FAILED_EventType}, types)
		assert.True(t, valid)
	})

	t.Run("should not pass events with wrong become events", func(t *testing.T) {
		t.Parallel()

		// given
		execution := &TestWorkflowExecution{}
		e := Event{Type_: EventEndTestWorkflowFailed, TestWorkflowExecution: execution}

		// when
		types, valid := e.Valid("", []EventType{BECOME_TESTWORKFLOW_UP_EventType})

		// then
		assert.Nil(t, types)
		assert.False(t, valid)
	})
}

func TestEvent_IsSuccess(t *testing.T) {
	t.Parallel()

	t.Run("should return true for success events", func(t *testing.T) {
		t.Parallel()

		events := map[EventType]bool{
			START_TEST_EventType:               false,
			START_TESTSUITE_EventType:          false,
			END_TEST_FAILED_EventType:          false,
			END_TEST_SUCCESS_EventType:         true,
			END_TESTSUITE_FAILED_EventType:     false,
			END_TESTSUITE_SUCCESS_EventType:    true,
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
