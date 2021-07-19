package executor

import "time"

const (
	ExecutionStatusQueued  = "queued"
	ExecutionStatusPending = "pending"
	ExecutionStatusSuceess = "success"
	ExecutionStatusError   = "error"
)

func NewExecution(ID string, name string, content string) Execution {
	return Execution{
		Id:            ID,
		Name:          name,
		ScriptType:    "postman/collection",
		ScriptContent: content,
		Status:        ExecutionStatusQueued,
	}
}
func (e *Execution) Start() {
	e.StartTime = time.Now()
}

func (e *Execution) Stop() {
	e.EndTime = time.Now()
}
