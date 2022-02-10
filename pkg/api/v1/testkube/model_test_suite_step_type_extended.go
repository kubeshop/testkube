package testkube

func TestSuiteStepTypePtr(stepType TestSuiteStepType) *TestSuiteStepType {
	return &stepType
}

var (
	TestSuiteStepTypeExecuteScript = TestSuiteStepTypePtr(EXECUTE_SCRIPT_TestSuiteStepType)
	TestSuiteStepTypeDelay         = TestSuiteStepTypePtr(DELAY_TestSuiteStepType)
)
