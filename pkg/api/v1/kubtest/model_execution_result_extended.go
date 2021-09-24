package kubtest

import (
	"time"
)

func NewResult() ExecutionResult {
	return ExecutionResult{
		Status: StatusPtr(QUEUED_ExecutionStatus),
	}
}

func NewQueuedResult() ExecutionResult {
	return ExecutionResult{
		Status: StatusPtr(QUEUED_ExecutionStatus),
	}
}

func (e *ExecutionResult) Start() {
	e.StartTime = time.Now()
}

func (e *ExecutionResult) Stop() {
	e.EndTime = time.Now()
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

func (e *ExecutionResult) Duration() time.Duration {

	end := e.EndTime
	start := e.StartTime

	if start.UnixNano() <= 0 && end.UnixNano() <= 0 {
		return time.Duration(0)
	}

	if end.UnixNano() <= 0 {
		end = time.Now()
	}

	return end.Sub(e.StartTime)
}
func (r ExecutionResult) Err(err error) ExecutionResult {
	r.ErrorMessage = err.Error()
	return r
}
