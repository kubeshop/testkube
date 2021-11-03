package testkube

func NewPendingExecutionResult() ExecutionResult {
	return ExecutionResult{
		Status: StatusPtr(PENDING_ExecutionStatus),
	}
}

func NewQueuedExecutionResult() ExecutionResult {
	return ExecutionResult{
		Status: StatusPtr(QUEUED_ExecutionStatus),
	}
}

func (e *ExecutionResult) Success() {
	e.Status = StatusPtr(SUCCESS_ExecutionStatus)
}

func (e *ExecutionResult) Error() {
	e.Status = StatusPtr(ERROR__ExecutionStatus)
}

func (e *ExecutionResult) IsCompleted() bool {
	return e.IsSuccesful() || e.IsFailed()
}

func (e *ExecutionResult) IsPending() bool {
	return *e.Status == PENDING_ExecutionStatus
}

func (e *ExecutionResult) IsQueued() bool {
	return *e.Status == QUEUED_ExecutionStatus
}

func (e *ExecutionResult) IsSuccesful() bool {
	return *e.Status == SUCCESS_ExecutionStatus
}

func (e *ExecutionResult) IsFailed() bool {
	return *e.Status == ERROR__ExecutionStatus
}

func (r *ExecutionResult) Err(err error) ExecutionResult {
	r.ErrorMessage = err.Error()
	return *r
}
