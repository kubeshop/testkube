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
}

func (t EventType) String() string {
	return string(t)
}

func EventTypePtr(t EventType) *EventType {
	return &t
}

var (
	EventStartTest           = EventTypePtr(START_TEST_EventType)
	EventEndTestSuccess      = EventTypePtr(END_TEST_SUCCESS_EventType)
	EventEndTestFailed       = EventTypePtr(END_TEST_FAILED_EventType)
	EventEndTestAborted      = EventTypePtr(END_TEST_ABORTED_EventType)
	EventEndTestTimeout      = EventTypePtr(END_TEST_TIMEOUT_EventType)
	EventStartTestSuite      = EventTypePtr(START_TESTSUITE_EventType)
	EventEndTestSuiteSuccess = EventTypePtr(END_TESTSUITE_SUCCESS_EventType)
	EventEndTestSuiteFailed  = EventTypePtr(END_TESTSUITE_FAILED_EventType)
	EventEndTestSuiteAborted = EventTypePtr(END_TESTSUITE_ABORTED_EventType)
	EventEndTestSuiteTimeout = EventTypePtr(END_TESTSUITE_TIMEOUT_EventType)
)

func EventTypesFromSlice(types []string) []EventType {
	var t []EventType
	for _, v := range types {
		t = append(t, EventType(v))
	}
	return t
}
