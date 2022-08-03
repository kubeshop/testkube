package testkube

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewExecutionWithID(id, testType, testName string) Execution {
	return Execution{
		Id:              id,
		ExecutionResult: &ExecutionResult{},
		TestName:        testName,
		TestType:        testType,
	}
}

func NewExecution(testNamespace, testName, testSuiteName, executionName, testType string,
	executionNumber int32, content *TestContent, result ExecutionResult,
	variables map[string]Variable, testSecretUUID, testSuiteSecretUUID string,
	labels map[string]string) Execution {
	return Execution{
		Id:                  primitive.NewObjectID().Hex(),
		TestName:            testName,
		TestSuiteName:       testSuiteName,
		TestNamespace:       testNamespace,
		Name:                executionName,
		Number:              executionNumber,
		TestType:            testType,
		ExecutionResult:     &result,
		Variables:           variables,
		TestSecretUUID:      testSecretUUID,
		TestSuiteSecretUUID: testSuiteSecretUUID,
		Content:             content,
		Labels:              labels,
	}
}

func NewFailedExecution(err error) Execution {
	return Execution{
		Id: primitive.NewObjectID().Hex(),
		ExecutionResult: &ExecutionResult{
			ErrorMessage: err.Error(),
			Status:       ExecutionStatusFailed,
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
	header = []string{"Id", "Name", "Test Name", "Type", "Status", "Labels"}

	for _, e := range executions {
		status := "unknown"
		if e.ExecutionResult != nil && e.ExecutionResult.Status != nil {
			status = string(*e.ExecutionResult.Status)
		}

		output = append(output, []string{
			e.Id,
			e.Name,
			e.TestName,
			e.TestType,
			status,
			MapToString(e.Labels),
		})
	}

	return
}

func (e *Execution) WithContent(content *TestContent) *Execution {
	e.Content = content
	return e
}

func (e *Execution) WithVariables(variables map[string]Variable) *Execution {
	e.Variables = variables
	return e
}

func (e *Execution) Err(err error) Execution {
	if e.ExecutionResult == nil {
		e.ExecutionResult = &ExecutionResult{}
	}

	e.ExecutionResult.Err(err)
	return *e
}
func (e *Execution) Errw(msg string, err error) Execution {
	if e.ExecutionResult == nil {
		e.ExecutionResult = &ExecutionResult{}
	}

	e.ExecutionResult.Err(fmt.Errorf(msg, err))
	return *e
}

func (e *Execution) Start() {
	e.StartTime = time.Now()
	if e.ExecutionResult != nil {
		e.ExecutionResult.Status = ExecutionStatusRunning
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

	return *e.ExecutionResult.Status == FAILED_ExecutionStatus
}
