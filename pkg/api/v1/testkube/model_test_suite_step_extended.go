package testkube

func (s TestSuiteStep) Type() *TestSuiteStepType {
	if s.Test != nil {
		return TestSuiteStepTypeExecuteTest
	}
	if s.Delay != nil {
		return TestSuiteStepTypeDelay
	}
	return nil
}

func (s TestSuiteStep) FullName() string {
	switch s.Type() {
	case TestSuiteStepTypeDelay:
		return s.Delay.FullName()
	case TestSuiteStepTypeExecuteTest:
		return s.Test.FullName()
	default:
		return "unknown"
	}
}
