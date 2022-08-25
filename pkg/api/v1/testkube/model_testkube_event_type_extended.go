package testkube

func (t *TestkubeEventType) String() string {
	return string(*t)
}

func TestkubeEventTypePtr(t TestkubeEventType) *TestkubeEventType {
	return &t
}

var (
	TestkubeEventStartTest = TestkubeEventTypePtr(START_TEST_TestkubeEventType)
	TestkubeEventEndTest   = TestkubeEventTypePtr(END_TEST_TestkubeEventType)
)

func TestkubeEventTypesFromSlice(types []string) []TestkubeEventType {
	var t []TestkubeEventType
	for _, v := range types {
		t = append(t, TestkubeEventType(v))
	}
	return t
}
