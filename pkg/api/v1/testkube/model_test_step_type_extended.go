package testkube

func TestStepTypePtr(stepType TestStepType) *TestStepType {
	return &stepType
}

var (
	TestStepTypeExecuteScript = TestStepTypePtr(EXECUTE_SCRIPT_TestStepType)
	TestStepTypeDelay         = TestStepTypePtr(DELAY_TestStepType)
)
