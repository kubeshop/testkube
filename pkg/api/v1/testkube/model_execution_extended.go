package testkube

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/kubeshop/testkube/pkg/utils"
)

func NewExecutionWithID(id, testType, testName string) *Execution {
	return &Execution{
		Id: id,
		ExecutionResult: &ExecutionResult{
			Status: ExecutionStatusQueued,
		},
		TestName: testName,
		TestType: testType,
		Labels:   map[string]string{},
	}
}

func NewExecution(id, testNamespace, testName, testSuiteName, executionName, testType string,
	executionNumber int, content *TestContent, result ExecutionResult,
	variables map[string]Variable, testSecretUUID, testSuiteSecretUUID string,
	labels map[string]string) Execution {
	if id == "" {
		id = primitive.NewObjectID().Hex()
	}

	return Execution{
		Id:                  id,
		TestName:            testName,
		TestSuiteName:       testSuiteName,
		TestNamespace:       testNamespace,
		Name:                executionName,
		Number:              int32(executionNumber),
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
func (e *Execution) Errw(id, msg string, err error) Execution {
	if e.ExecutionResult == nil {
		e.ExecutionResult = &ExecutionResult{}
	}

	e.Id = id
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
	duration := e.CalculateDuration()
	e.Duration = utils.RoundDuration(duration).String()
	e.DurationMs = int32(duration.Milliseconds())
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

func (e Execution) IsAborted() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == ABORTED_ExecutionStatus
}

func (e Execution) IsRunning() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == RUNNING_ExecutionStatus
}

func (e Execution) IsQueued() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == QUEUED_ExecutionStatus
}

func (e Execution) IsCanceled() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == ABORTED_ExecutionStatus
}

func (e Execution) IsTimeout() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == TIMEOUT_ExecutionStatus
}

func (e Execution) IsPassed() bool {
	if e.ExecutionResult == nil {
		return true
	}

	return *e.ExecutionResult.Status == PASSED_ExecutionStatus
}

func (e *Execution) WithID() *Execution {
	if e.Id == "" {
		e.Id = primitive.NewObjectID().Hex()
	}

	return e
}

func (e *Execution) convertDots(fn func(string) string) *Execution {
	labels := make(map[string]string, len(e.Labels))
	for key, value := range e.Labels {
		labels[fn(key)] = value
	}
	e.Labels = labels

	envs := make(map[string]string, len(e.Envs))
	for key, value := range e.Envs {
		envs[fn(key)] = value
	}
	e.Envs = envs

	vars := make(map[string]Variable, len(e.Variables))
	for key, value := range e.Variables {
		vars[fn(key)] = value
	}
	e.Variables = vars
	return e
}

func (e *Execution) EscapeDots() *Execution {
	return e.convertDots(utils.EscapeDots)
}

func (e *Execution) UnscapeDots() *Execution {
	return e.convertDots(utils.UnescapeDots)
}
