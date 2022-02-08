package testkube

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewExecutionWithID(id, scriptType, scriptName string) Execution {
	return Execution{
		Id:              id,
		ExecutionResult: &ExecutionResult{},
		ScriptName:      scriptName,
		ScriptType:      scriptType,
	}
}

func NewExecution(scriptName, executionName, scriptType string, content *ScriptContent, result ExecutionResult, params map[string]string, tags []string) Execution {
	return Execution{
		Id:              primitive.NewObjectID().Hex(),
		ScriptName:      scriptName,
		Name:            executionName,
		ScriptType:      scriptType,
		ExecutionResult: &result,
		Params:          params,
		Content:         content,
		Tags:            tags,
	}
}

func NewFailedExecution(err error) Execution {
	return Execution{
		Id: primitive.NewObjectID().Hex(),
		ExecutionResult: &ExecutionResult{
			ErrorMessage: err.Error(),
			Status:       ExecutionStatusError,
		},
	}
}

// NewQueued execution for executions status used in test executions
func NewQueuedExecution() *Execution {
	return &Execution{
		ExecutionResult: &ExecutionResult{
			Status: ExecutionStatusQueued,
		},
	}
}

type Executions []Execution

func (executions Executions) Table() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Name", "ID", "Status"}

	for _, e := range executions {
		status := "unknown"
		if e.ExecutionResult != nil && e.ExecutionResult.Status != nil {
			status = string(*e.ExecutionResult.Status)
		}

		output = append(output, []string{
			e.ScriptName,
			e.ScriptType,
			e.Name,
			e.Id,
			status,
		})
	}

	return
}

func (e *Execution) WithContent(content *ScriptContent) *Execution {
	e.Content = content
	return e
}

func (e *Execution) WithParams(params map[string]string) *Execution {
	e.Params = params
	return e
}

func (e Execution) Err(err error) Execution {
	e.ExecutionResult.Err(err)
	return e
}
func (e Execution) Errw(msg string, err error) Execution {
	e.ExecutionResult.Err(fmt.Errorf(msg, err))
	return e
}

func (e *Execution) Start() {
	e.StartTime = time.Now()
	if e.ExecutionResult != nil {
		e.ExecutionResult.Status = ExecutionStatusPending
	}
}

func (e *Execution) Stop() {
	e.EndTime = time.Now()
	e.Duration = e.CalculateDuration().String()
}
func (e *Execution) CalculateDuration() time.Duration {

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
func (e Execution) IsFailed() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == ERROR__ExecutionStatus
}
