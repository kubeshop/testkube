package testkube

func (s TestWorkflowStepStatus) Finished() bool {
	return s != "" && s != QUEUED_TestWorkflowStepStatus && s != PAUSED_TestWorkflowStepStatus && s != RUNNING_TestWorkflowStepStatus
}
