package kubtest

import (
	"time"
)

const (
	// ResultCreated status for execution which is requested to queue
	ResultCreated = "created"
	// ResultQueued status for execution which is added for queue but not get yet by worker
	ResultQueued = "queued"
	// ResultPending status for execution which is taken by worker
	ResultPending = "pending"
	// ResultSuceess execution complete with success
	ResultSuceess = "success"
	// ResultError execution failed
	ResultError = "error"
)

func NewResult() ExecutionResult {
	return ExecutionResult{
		Status: ResultQueued,
	}
}

func NewQueuedResult() ExecutionResult {
	return ExecutionResult{
		Status: ResultQueued,
	}
}

func (e *ExecutionResult) Start() {
	e.StartTime = time.Now()
}

func (e *ExecutionResult) Stop() {
	e.EndTime = time.Now()
}

func (e *ExecutionResult) Success() {
	e.Status = ResultSuceess
}

func (e *ExecutionResult) Error() {
	e.Status = ResultError
}

func (e *ExecutionResult) IsCompleted() bool {
	return e.IsSuccesful() || e.IsFailed()
}

func (e *ExecutionResult) IsPending() bool {
	return e.Status == ResultPending
}

func (e *ExecutionResult) IsQueued() bool {
	return e.Status == ResultQueued
}

func (e *ExecutionResult) IsSuccesful() bool {
	return e.Status == ResultSuceess
}

func (e *ExecutionResult) IsFailed() bool {
	return e.Status == ResultError
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
