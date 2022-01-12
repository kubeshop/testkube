package testkube

func (s TestStep) Type() *TestStepType {
	if s.Execute != nil {
		return TestStepTypeExecuteScript
	}
	if s.Delay != nil {
		return TestStepTypeDelay
	}
	return nil
}

func (s TestStep) FullName() string {
	switch s.Type() {
	case TestStepTypeDelay:
		return s.Delay.Name
	case TestStepTypeExecuteScript:
		return s.Execute.Name
	default:
		return "unknown"
	}
}
