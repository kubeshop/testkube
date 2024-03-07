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
	EventStartTest              = EventTypePtr(START_TEST_EventType)
	EventEndTestSuccess         = EventTypePtr(END_TEST_SUCCESS_EventType)
	EventEndTestFailed          = EventTypePtr(END_TEST_FAILED_EventType)
	EventEndTestAborted         = EventTypePtr(END_TEST_ABORTED_EventType)
	EventEndTestTimeout         = EventTypePtr(END_TEST_TIMEOUT_EventType)
	EventStartTestSuite         = EventTypePtr(START_TESTSUITE_EventType)
	EventEndTestSuiteSuccess    = EventTypePtr(END_TESTSUITE_SUCCESS_EventType)
	EventEndTestSuiteFailed     = EventTypePtr(END_TESTSUITE_FAILED_EventType)
	EventEndTestSuiteAborted    = EventTypePtr(END_TESTSUITE_ABORTED_EventType)
	EventEndTestSuiteTimeout    = EventTypePtr(END_TESTSUITE_TIMEOUT_EventType)
	EventQueueTestWorkflow      = EventTypePtr(QUEUE_TESTWORKFLOW_EventType)
	EventStartTestWorkflow      = EventTypePtr(START_TESTWORKFLOW_EventType)
	EventEndTestWorkflowSuccess = EventTypePtr(END_TESTWORKFLOW_SUCCESS_EventType)
	EventEndTestWorkflowFailed  = EventTypePtr(END_TESTWORKFLOW_FAILED_EventType)
	EventEndTestWorkflowAborted = EventTypePtr(END_TESTWORKFLOW_ABORTED_EventType)
	EventCreated                = EventTypePtr(CREATED_EventType)
	EventDeleted                = EventTypePtr(DELETED_EventType)
	EventUpdated                = EventTypePtr(UPDATED_EventType)
)

func EventTypesFromSlice(types []string) []EventType {
	var t []EventType
	for _, v := range types {
		t = append(t, EventType(v))
	}
	return t
}
