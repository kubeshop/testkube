package testkube

func NewTestStepQueuedResult(step *TestStep) (result TestStepExecutionResult) {
	result.Step = step
	result.Execution = &Execution{ExecutionResult: &ExecutionResult{Status: ExecutionStatusQueued}}

	return
}

func NewTestStepExecutionResult(execution Execution, executeStep *TestStepExecuteScript) (result TestStepExecutionResult) {
	result.Execution = &execution
	result.Script = &ObjectRef{Name: executeStep.Name, Namespace: executeStep.Namespace}

	return
}

func NewTestStepDelayResult() (result TestStepExecutionResult) {
	result.Execution = &Execution{ExecutionResult: &ExecutionResult{Status: ExecutionStatusSuccess}}

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
