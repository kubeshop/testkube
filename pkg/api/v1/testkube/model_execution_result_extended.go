package testkube

func NewPendingExecutionResult() ExecutionResult {
	return ExecutionResult{
		Status: StatusPtr(RUNNING_ExecutionStatus),
	}
}

func NewErrorExecutionResult(err error) ExecutionResult {
	return ExecutionResult{
		Status:       StatusPtr(FAILED_ExecutionStatus),
		ErrorMessage: err.Error(),
	}
}

func (e *ExecutionResult) InProgress() {
	e.Status = StatusPtr(RUNNING_ExecutionStatus)
}

func (e *ExecutionResult) Success() {
	e.Status = StatusPtr(PASSED_ExecutionStatus)
}

func (e *ExecutionResult) Error() {
	e.Status = StatusPtr(FAILED_ExecutionStatus)
}

func (e *ExecutionResult) IsCompleted() bool {
	return e.IsPassed() || e.IsFailed()
}

func (e *ExecutionResult) IsRunning() bool {
	return *e.Status == RUNNING_ExecutionStatus
}

func (e *ExecutionResult) IsQueued() bool {
	return *e.Status == QUEUED_ExecutionStatus
}

func (e *ExecutionResult) IsPassed() bool {
	return *e.Status == PASSED_ExecutionStatus
}

func (e *ExecutionResult) IsFailed() bool {
	return *e.Status == FAILED_ExecutionStatus
}

func (r *ExecutionResult) Err(err error) ExecutionResult {
	r.Status = ExecutionStatusFailed
	r.ErrorMessage = err.Error()
	return *r
}

// Errs return error result if any of passed errors is not nil
func (r *ExecutionResult) WithErrors(errors ...error) ExecutionResult {
	for _, err := range errors {
		if err != nil {
			return r.Err(err)
		}
	}
	return *r
}
