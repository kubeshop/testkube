package testkube

var AllEventTypes = []EventType{
	START_TEST_EventType,
	END_TEST_SUCCESS_EventType,
	END_TEST_FAILED_EventType,
	END_TEST_ABORTED_EventType,
	END_TEST_TIMEOUT_EventType,
	START_TESTSUITE_EventType,
	END_TESTSUITE_SUCCESS_EventType,
	END_TESTSUITE_FAILED_EventType,
	END_TESTSUITE_ABORTED_EventType,
	END_TESTSUITE_TIMEOUT_EventType,
	QUEUE_TESTWORKFLOW_EventType,
	START_TESTWORKFLOW_EventType,
	END_TESTWORKFLOW_SUCCESS_EventType,
	END_TESTWORKFLOW_FAILED_EventType,
	END_TESTWORKFLOW_ABORTED_EventType,
	END_TESTWORKFLOW_CANCELED_EventType,
	CREATED_EventType,
	DELETED_EventType,
	UPDATED_EventType,
}

func (t EventType) String() string {
	return string(t)
}

func EventTypePtr(t EventType) *EventType {
	return &t
}

var (
	EventStartTest               = EventTypePtr(START_TEST_EventType)
	EventEndTestSuccess          = EventTypePtr(END_TEST_SUCCESS_EventType)
	EventEndTestFailed           = EventTypePtr(END_TEST_FAILED_EventType)
	EventEndTestAborted          = EventTypePtr(END_TEST_ABORTED_EventType)
	EventEndTestTimeout          = EventTypePtr(END_TEST_TIMEOUT_EventType)
	EventStartTestSuite          = EventTypePtr(START_TESTSUITE_EventType)
	EventEndTestSuiteSuccess     = EventTypePtr(END_TESTSUITE_SUCCESS_EventType)
	EventEndTestSuiteFailed      = EventTypePtr(END_TESTSUITE_FAILED_EventType)
	EventEndTestSuiteAborted     = EventTypePtr(END_TESTSUITE_ABORTED_EventType)
	EventEndTestSuiteTimeout     = EventTypePtr(END_TESTSUITE_TIMEOUT_EventType)
	EventQueueTestWorkflow       = EventTypePtr(QUEUE_TESTWORKFLOW_EventType)
	EventStartTestWorkflow       = EventTypePtr(START_TESTWORKFLOW_EventType)
	EventEndTestWorkflowSuccess  = EventTypePtr(END_TESTWORKFLOW_SUCCESS_EventType)
	EventEndTestWorkflowFailed   = EventTypePtr(END_TESTWORKFLOW_FAILED_EventType)
	EventEndTestWorkflowAborted  = EventTypePtr(END_TESTWORKFLOW_ABORTED_EventType)
	EventEndTestWorkflowCanceled = EventTypePtr(END_TESTWORKFLOW_CANCELED_EventType)
	EventCreated                 = EventTypePtr(CREATED_EventType)
	EventDeleted                 = EventTypePtr(DELETED_EventType)
	EventUpdated                 = EventTypePtr(UPDATED_EventType)
)

func EventTypesFromSlice(types []string) []EventType {
	var t []EventType
	for _, v := range types {
		t = append(t, EventType(v))
	}
	return t
}

func (t EventType) IsBecome() bool {
	types := []EventType{
		BECOME_TEST_UP_EventType,
		BECOME_TEST_DOWN_EventType,
		BECOME_TEST_FAILED_EventType,
		BECOME_TEST_ABORTED_EventType,
		BECOME_TEST_TIMEOUT_EventType,

		BECOME_TESTSUITE_UP_EventType,
		BECOME_TESTSUITE_DOWN_EventType,
		BECOME_TESTSUITE_FAILED_EventType,
		BECOME_TESTSUITE_ABORTED_EventType,
		BECOME_TESTSUITE_TIMEOUT_EventType,

		BECOME_TESTWORKFLOW_UP_EventType,
		BECOME_TESTWORKFLOW_DOWN_EventType,
		BECOME_TESTWORKFLOW_FAILED_EventType,
		BECOME_TESTWORKFLOW_ABORTED_EventType,
		BECOME_TESTWORKFLOW_CANCELED_EventType,
	}

	for _, tp := range types {
		if tp == t {
			return true
		}
	}

	return false
}

func (t EventType) MapBecomeToRegular() []EventType {
	eventMap := map[EventType][]EventType{
		BECOME_TEST_UP_EventType:      {END_TEST_SUCCESS_EventType},
		BECOME_TEST_DOWN_EventType:    {END_TEST_FAILED_EventType, END_TEST_ABORTED_EventType, END_TEST_TIMEOUT_EventType},
		BECOME_TEST_FAILED_EventType:  {END_TEST_FAILED_EventType},
		BECOME_TEST_ABORTED_EventType: {END_TEST_ABORTED_EventType},
		BECOME_TEST_TIMEOUT_EventType: {END_TEST_TIMEOUT_EventType},

		BECOME_TESTSUITE_UP_EventType:      {END_TESTSUITE_SUCCESS_EventType},
		BECOME_TESTSUITE_DOWN_EventType:    {END_TESTSUITE_FAILED_EventType, END_TESTSUITE_ABORTED_EventType, END_TESTSUITE_TIMEOUT_EventType},
		BECOME_TESTSUITE_FAILED_EventType:  {END_TESTSUITE_FAILED_EventType},
		BECOME_TESTSUITE_ABORTED_EventType: {END_TESTSUITE_ABORTED_EventType},
		BECOME_TESTSUITE_TIMEOUT_EventType: {END_TESTSUITE_TIMEOUT_EventType},

		BECOME_TESTWORKFLOW_UP_EventType: {END_TESTWORKFLOW_SUCCESS_EventType},
		// TODO: is cancelled down?
		BECOME_TESTWORKFLOW_DOWN_EventType:     {END_TESTWORKFLOW_FAILED_EventType, END_TESTWORKFLOW_ABORTED_EventType},
		BECOME_TESTWORKFLOW_FAILED_EventType:   {END_TESTWORKFLOW_FAILED_EventType},
		BECOME_TESTWORKFLOW_ABORTED_EventType:  {END_TESTWORKFLOW_ABORTED_EventType},
		BECOME_TESTWORKFLOW_CANCELED_EventType: {END_TESTWORKFLOW_CANCELED_EventType},
	}

	return eventMap[t]
}

func (t EventType) IsBecomeExecutionStatus(previousStatus ExecutionStatus) bool {
	eventMap := map[EventType]map[ExecutionStatus]struct{}{
		BECOME_TEST_UP_EventType: {
			FAILED_ExecutionStatus:  {},
			ABORTED_ExecutionStatus: {},
			TIMEOUT_ExecutionStatus: {},
		},

		BECOME_TEST_DOWN_EventType: {
			PASSED_ExecutionStatus: {},
		},

		BECOME_TEST_FAILED_EventType: {
			PASSED_ExecutionStatus:  {},
			ABORTED_ExecutionStatus: {},
			TIMEOUT_ExecutionStatus: {},
		},

		BECOME_TEST_ABORTED_EventType: {
			PASSED_ExecutionStatus:  {},
			FAILED_ExecutionStatus:  {},
			TIMEOUT_ExecutionStatus: {},
		},

		BECOME_TEST_TIMEOUT_EventType: {
			PASSED_ExecutionStatus:  {},
			FAILED_ExecutionStatus:  {},
			ABORTED_ExecutionStatus: {},
		},
	}

	if statusMap, ok := eventMap[t]; ok {
		if _, ok := statusMap[previousStatus]; ok {
			return true
		}
	}

	return false
}

func (t EventType) IsBecomeTestSuiteExecutionStatus(previousStatus TestSuiteExecutionStatus) bool {
	eventMap := map[EventType]map[TestSuiteExecutionStatus]struct{}{
		BECOME_TESTSUITE_UP_EventType: {
			FAILED_TestSuiteExecutionStatus:  {},
			ABORTED_TestSuiteExecutionStatus: {},
			TIMEOUT_TestSuiteExecutionStatus: {},
		},

		BECOME_TESTSUITE_DOWN_EventType: {
			PASSED_TestSuiteExecutionStatus: {},
		},

		BECOME_TESTSUITE_FAILED_EventType: {
			PASSED_TestSuiteExecutionStatus:  {},
			ABORTED_TestSuiteExecutionStatus: {},
			TIMEOUT_TestSuiteExecutionStatus: {},
		},

		BECOME_TESTSUITE_ABORTED_EventType: {
			PASSED_TestSuiteExecutionStatus:  {},
			FAILED_TestSuiteExecutionStatus:  {},
			TIMEOUT_TestSuiteExecutionStatus: {},
		},

		BECOME_TESTSUITE_TIMEOUT_EventType: {
			PASSED_TestSuiteExecutionStatus:  {},
			FAILED_TestSuiteExecutionStatus:  {},
			ABORTED_TestSuiteExecutionStatus: {},
		},
	}

	if statusMap, ok := eventMap[t]; ok {
		if _, ok := statusMap[previousStatus]; ok {
			return true
		}
	}

	return false
}

func (t EventType) IsBecomeTestWorkflowExecutionStatus(previousStatus TestWorkflowStatus) bool {
	eventMap := map[EventType]map[TestWorkflowStatus]struct{}{
		BECOME_TESTWORKFLOW_UP_EventType: {
			FAILED_TestWorkflowStatus:  {},
			ABORTED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_DOWN_EventType: {
			PASSED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_FAILED_EventType: {
			PASSED_TestWorkflowStatus:  {},
			ABORTED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_ABORTED_EventType: {
			PASSED_TestWorkflowStatus: {},
			FAILED_TestWorkflowStatus: {},
		},

		BECOME_TESTWORKFLOW_CANCELED_EventType: {
			PASSED_TestWorkflowStatus: {},
			FAILED_TestWorkflowStatus: {},
		},
	}

	if statusMap, ok := eventMap[t]; ok {
		if _, ok := statusMap[previousStatus]; ok {
			return true
		}
	}

	return false
}
