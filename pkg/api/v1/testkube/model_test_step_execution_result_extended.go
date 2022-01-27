package testkube

func NewTestStepQueuedResult(step *TestStep) (result TestStepExecutionResult) {
	result.Step = step
	result.Execution = NewQueuedExecution()

	return
}

func (r *TestStepExecutionResult) Err(err error) TestStepExecutionResult {
	if r.Execution == nil {
		execution := NewFailedExecution(err)
		r.Execution = &execution
	}
	e := r.Execution.Err(err)
	r.Execution = &e
	return *r
}

func (r *TestStepExecutionResult) IsFailed() bool {
	if r.Execution != nil {
		return r.Execution.IsFailed()
	}

	return true
}
