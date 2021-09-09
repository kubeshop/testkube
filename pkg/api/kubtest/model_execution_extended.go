package kubtest

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewExecution() Execution {
	status := QUEUED_ExecutionStatus
	return Execution{
		Id:     primitive.NewObjectID().Hex(),
		Status: &status,
		Result: &ExecutionResult{},
	}
}

func (e *Execution) WithContent(content string) *Execution {
	e.ScriptContent = content
	return e
}

func (e *Execution) WithRepository(repository *Repository) *Execution {
	e.Repository = repository
	return e
}

func (e *Execution) WithParams(params map[string]string) *Execution {
	e.Params = params
	return e
}

func (e *Execution) WithRepositoryData(uri, branch, path string) *Execution {
	e.Repository = &Repository{
		Uri:    uri,
		Branch: branch,
		Path:   path,
	}
	return e
}

func (e *Execution) Start() {
	e.StartTime = time.Now()
}

func (e *Execution) Stop() {
	e.EndTime = time.Now()
}

func (e *Execution) Success() {
	success := SUCCESS_ExecutionStatus
	e.Status = &success
}

func (e *Execution) Error() {
	failed := FAILED_ExecutionStatus
	e.Status = &failed
}

func (e *Execution) IsCompleted() bool {
	return e.IsSuccesful() || e.IsFailed()
}

func (e *Execution) IsPending() bool {
	return *e.Status == PENDING_ExecutionStatus
}

func (e *Execution) IsQueued() bool {
	return *e.Status == QUEUED_ExecutionStatus
}

func (e *Execution) IsSuccesful() bool {
	return *e.Status == SUCCESS_ExecutionStatus
}

func (e *Execution) IsFailed() bool {
	return *e.Status == FAILED_ExecutionStatus
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
