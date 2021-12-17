package testkube

func (r *TestStepExecutionResult) Err(err error) TestStepExecutionResult {
	if r.Result == nil {
		execution := NewFailedExecution(err)
		r.Result = &execution

	}
	e := r.Result.Err(err)
	r.Result = &e
	return *r
}

func (r *TestStepExecutionResult) IsFailed() bool {
	if r.Result != nil {
		return r.Result.IsFailed()
	}

	return true
}
