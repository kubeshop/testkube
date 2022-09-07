package testkube

var AllEventTypes = []EventType{
	START_TEST_EventType,
	START_TESTSUITE_EventType,
	END_TEST_EventType,
	END_TESTSUITE_EventType,
	TEST_FAILED_EventType,
	TESTSUITE_FAILED_EventType,
}

func (t *EventType) String() string {
	return string(*t)
}

func EventTypePtr(t EventType) *EventType {
	return &t
}

var (
	EventStartTest = EventTypePtr(START_TEST_EventType)
	EventEndTest   = EventTypePtr(END_TEST_EventType)
)

func EventTypesFromSlice(types []string) []EventType {
	var t []EventType
	for _, v := range types {
		t = append(t, EventType(v))
	}
	return t
}
