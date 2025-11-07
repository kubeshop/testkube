package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventType_IsBecome(t *testing.T) {
	t.Parallel()

	t.Run("should return true for become events", func(t *testing.T) {
		t.Parallel()

		events := map[EventType]bool{
			START_TEST_EventType:             false,
			BECOME_TEST_UP_EventType:         true,
			START_TESTSUITE_EventType:        false,
			BECOME_TESTSUITE_UP_EventType:    true,
			START_TESTWORKFLOW_EventType:     false,
			BECOME_TESTWORKFLOW_UP_EventType: true,
		}

		for eventType, expected := range events {
			// given
			e := Event{Type_: &eventType}

			// when
			become := e.Type_.IsBecome()

			// then
			assert.Equal(t, expected, become)
		}
	})

}

func TestEventType_MapBecomeToRegular(t *testing.T) {
	t.Parallel()

	t.Run("should return event types for become events", func(t *testing.T) {
		t.Parallel()

		events := map[EventType][]EventType{
			START_TEST_EventType:             nil,
			BECOME_TEST_UP_EventType:         {END_TEST_SUCCESS_EventType},
			START_TESTSUITE_EventType:        nil,
			BECOME_TESTSUITE_UP_EventType:    {END_TESTSUITE_SUCCESS_EventType},
			START_TESTWORKFLOW_EventType:     nil,
			BECOME_TESTWORKFLOW_UP_EventType: {END_TESTWORKFLOW_SUCCESS_EventType},
		}

		for eventType, expected := range events {
			// given
			e := Event{Type_: &eventType}

			// when
			types := e.Type_.MapBecomeToRegular()

			// then
			assert.Equal(t, expected, types)
		}
	})

}

func TestEventType_IsBecomeTestWorkflowExecutionStatus(t *testing.T) {
	t.Parallel()

	t.Run("should return true for become test workflow execution status", func(t *testing.T) {
		t.Parallel()

		events := []struct {
			eventType EventType
			status    TestWorkflowStatus
			result    bool
		}{
			{
				eventType: BECOME_TESTWORKFLOW_UP_EventType,
				status:    FAILED_TestWorkflowStatus,
				result:    true,
			},
			{
				eventType: END_TESTWORKFLOW_SUCCESS_EventType,
				status:    FAILED_TestWorkflowStatus,
				result:    false,
			},
			{
				eventType: BECOME_TESTWORKFLOW_UP_EventType,
				status:    PASSED_TestWorkflowStatus,
				result:    false,
			},
		}

		for _, expected := range events {
			// given
			e := Event{Type_: &expected.eventType}

			// when
			become := e.Type_.IsBecomeTestWorkflowExecutionStatus(expected.status)

			// then
			assert.Equal(t, expected.result, become)
		}
	})

}
