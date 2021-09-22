package kubtest

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// ExecutionStatusCreated status for execution which is requested to queue
	ExecutionStatusCreated = "created"
	// ExecutionStatusQueued status for execution which is added for queue but not get yet by worker
	ExecutionStatusQueued = "queued"
	// ExecutionStatusPending status for execution which is taken by worker
	ExecutionStatusPending = "pending"
	// ExecutionStatusSuceess execution complete with success
	ExecutionStatusSuceess = "success"
	// ExecutionStatusSuceess execution failed
	ExecutionStatusError = "error"
)

func NewExecution() Result {
	return Result{
		Id:     primitive.NewObjectID().Hex(),
		Status: ExecutionStatusQueued,
		Result: &ExecutionResult{Status: ExecutionStatusQueued},
	}
}

func NewQueuedExecution() Result {
	return Result{
		Id:     primitive.NewObjectID().Hex(),
		Status: ExecutionStatusQueued,
		Result: &ExecutionResult{Status: ExecutionStatusQueued},
	}
}

func (e *Result) WithContent(content string) *Result {
	e.ScriptContent = content
	return e
}

func (e *Result) WithRepository(repository *Repository) *Result {
	e.Repository = repository
	return e
}

func (e *Result) WithParams(params map[string]string) *Result {
	e.Params = params
	return e
}

func (e *Result) WithRepositoryData(uri, branch, path string) *Result {
	e.Repository = &Repository{
		Uri:    uri,
		Branch: branch,
		Path:   path,
	}
	return e
}

func (e *Result) Start() {
	e.StartTime = time.Now()
}

func (e *Result) Stop() {
	e.EndTime = time.Now()
}

func (e *Result) Success() {
	e.Status = ExecutionStatusSuceess
}

func (e *Result) Error() {
	e.Status = ExecutionStatusError
}

func (e *Result) IsCompleted() bool {
	return e.IsSuccesful() || e.IsFailed()
}

func (e *Result) IsPending() bool {
	return e.Status == ExecutionStatusPending
}

func (e *Result) IsQueued() bool {
	return e.Status == ExecutionStatusQueued
}

func (e *Result) IsSuccesful() bool {
	return e.Status == ExecutionStatusSuceess
}

func (e *Result) IsFailed() bool {
	return e.Status == ExecutionStatusError
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
