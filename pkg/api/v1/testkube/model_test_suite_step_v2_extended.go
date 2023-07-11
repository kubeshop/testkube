package testkube

func (s TestSuiteStepV2) Type() *TestSuiteStepType {
	if s.Execute != nil {
		return TestSuiteStepTypeExecuteTest
	}
	if s.Delay != nil {
		return TestSuiteStepTypeDelay
	}
	return nil
}

func (s TestSuiteStepV2) FullName() string {
	switch s.Type() {
	case TestSuiteStepTypeDelay:
		return s.Delay.FullName()
	case TestSuiteStepTypeExecuteTest:
		return s.Execute.FullName()
	default:
		return "unknown"
	}
}
