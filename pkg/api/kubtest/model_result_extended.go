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

func NewResult() Result {
	return Result{
		Status: ResultQueued,
	}
}

func NewQueuedResult() Result {
	return Result{
		Status: ResultQueued,
	}
}

func (e *Result) Start() {
	e.StartTime = time.Now()
}

func (e *Result) Stop() {
	e.EndTime = time.Now()
}

func (e *Result) Success() {
	e.Status = ResultSuceess
}

func (e *Result) Error() {
	e.Status = ResultError
}

func (e *Result) IsCompleted() bool {
	return e.IsSuccesful() || e.IsFailed()
}

func (e *Result) IsPending() bool {
	return e.Status == ResultPending
}

func (e *Result) IsQueued() bool {
	return e.Status == ResultQueued
}

func (e *Result) IsSuccesful() bool {
	return e.Status == ResultSuceess
}

func (e *Result) IsFailed() bool {
	return e.Status == ResultError
}

func (e *Result) Duration() time.Duration {

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
func (r Result) Err(err error) Result {
	r.ErrorMessage = err.Error()
	return r
}
