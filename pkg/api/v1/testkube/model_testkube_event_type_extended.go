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
