package testkube

func NewTestStepQueuedResult(step *TestSuiteStep) (result TestSuiteStepExecutionResult) {
	result.Step = step
	result.Execution = NewQueuedExecution().WithID()

	return
}

func (r *TestSuiteStepExecutionResult) Err(err error) TestSuiteStepExecutionResult {
	if r.Execution == nil {
		execution := NewFailedExecution(err)
		r.Execution = &execution
	}
	e := r.Execution.Err(err)
	r.Execution = &e
	return *r
}

func (r *TestSuiteStepExecutionResult) IsFailed() bool {
	if r.Execution != nil {
		return r.Execution.IsFailed()
	}

	return true
}

func (r *TestSuiteStepExecutionResult) IsAborted() bool {
	if r.Execution != nil {
		return r.Execution.IsAborted()
	}

	return false
}
