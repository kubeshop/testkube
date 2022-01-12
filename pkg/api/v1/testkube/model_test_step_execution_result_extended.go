package testkube

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

func (r *TestStepExecutionResult) Sto() bool {
	if r.Execution != nil {
		return r.Execution.IsFailed()
	}

	return true
}
