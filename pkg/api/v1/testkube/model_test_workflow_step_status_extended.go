package testkube

func (s *TestWorkflowStepStatus) Finished() bool {
	return s != nil && *s != "" && *s != QUEUED_TestWorkflowStepStatus && *s != PAUSED_TestWorkflowStepStatus && *s != RUNNING_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) Aborted() bool {
	return s != nil && *s == ABORTED_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) Skipped() bool {
	return s != nil && *s == SKIPPED_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) Paused() bool {
	return s != nil && *s == PAUSED_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) Running() bool {
	return s != nil && *s == RUNNING_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) AnyProgress() bool {
	return s.Running() || s.Paused()
}

func (s *TestWorkflowStepStatus) TimedOut() bool {
	return s != nil && *s == TIMEOUT_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) Failed() bool {
	return s != nil && *s == FAILED_TestWorkflowStepStatus
}

func (s *TestWorkflowStepStatus) AnyError() bool {
	return s.Failed() || s.TimedOut() || s.Aborted()
}

func (s *TestWorkflowStepStatus) NotStarted() bool {
	return s != nil && (*s == "" || *s == QUEUED_TestWorkflowStepStatus)
}
