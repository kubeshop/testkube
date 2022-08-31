package testkube

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
