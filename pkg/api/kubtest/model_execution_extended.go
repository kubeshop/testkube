package kubtest

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// ExecutionStatusQueued status for execution which is added for queue but not get yet by worker
	ExecutionStatusQueued = "queued"
	// ExecutionStatusPending status for execution which is taken by worker
	ExecutionStatusPending = "pending"
	// ExecutionStatusSuceess execution complete with success
	ExecutionStatusSuceess = "success"
	// ExecutionStatusSuceess execution failed
	ExecutionStatusError = "error"
)

func NewExecution(content string, params map[string]string) Execution {
	return Execution{
		Id:            primitive.NewObjectID().Hex(),
		ScriptContent: content,
		Status:        ExecutionStatusQueued,
		Params:        params,
		Result:        &ExecutionResult{},
	}
}
func (e *Execution) Start() {
	e.StartTime = time.Now()
}

func (e *Execution) Stop() {
	e.EndTime = time.Now()
}

func (e *Execution) Success() {
	e.Status = ExecutionStatusSuceess
}

func (e *Execution) Error() {
	e.Status = ExecutionStatusError
}

func (e *Execution) IsCompleted() bool {
	return e.IsSuccesful() || e.IsFailed()
}

func (e *Execution) IsPending() bool {
	return e.Status == ExecutionStatusPending
}

func (e *Execution) IsQueued() bool {
	return e.Status == ExecutionStatusQueued
}

func (e *Execution) IsSuccesful() bool {
	return e.Status == ExecutionStatusSuceess
}

func (e *Execution) IsFailed() bool {
	return e.Status == ExecutionStatusError
}

func (e *Execution) Duration() time.Duration {

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
