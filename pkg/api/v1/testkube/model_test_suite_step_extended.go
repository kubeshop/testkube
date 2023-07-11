package testkube

func (s TestSuiteStep) Type() *TestSuiteStepType {
	if s.Test != "" {
		return TestSuiteStepTypeExecuteTest
	}
	if s.Delay != "" {
		return TestSuiteStepTypeDelay
	}
	return nil
}

func (s TestSuiteStep) FullName() string {
	switch s.Type() {
	case TestSuiteStepTypeDelay:
		return s.Delay
	case TestSuiteStepTypeExecuteTest:
		return s.Test
	default:
		return "unknown"
	}
}
