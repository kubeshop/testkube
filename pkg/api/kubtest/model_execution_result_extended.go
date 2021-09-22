package kubtest

func (r ExecutionResult) Err(err error) ExecutionResult {
	r.ErrorMessage = err.Error()
	return r
}

func NewExecutionResult() *ExecutionResult {
	return &ExecutionResult{Status: "queued"}
}
