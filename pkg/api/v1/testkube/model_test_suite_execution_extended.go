package testkube

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/utils"
)

type WatchTestSuiteExecutionResponse struct {
	Execution TestSuiteExecution
	Error     error
}

func NewQueuedTestSuiteExecution(name, namespace string) *TestSuiteExecution {
	return &TestSuiteExecution{
		TestSuite: &ObjectRef{
			Name:      name,
			Namespace: namespace,
		},
		Status: TestSuiteExecutionStatusQueued,
	}
}

func NewStartedTestSuiteExecution(testSuite TestSuite, request TestSuiteExecutionRequest) TestSuiteExecution {

	testExecution := TestSuiteExecution{
		Id:                     primitive.NewObjectID().Hex(),
		StartTime:              time.Now(),
		Name:                   request.Name,
		Status:                 TestSuiteExecutionStatusRunning,
		SecretUUID:             request.SecretUUID,
		TestSuite:              testSuite.GetObjectRef(),
		Labels:                 common.MergeMaps(testSuite.Labels, request.ExecutionLabels),
		Variables:              map[string]Variable{},
		RunningContext:         request.RunningContext,
		TestSuiteExecutionName: request.TestSuiteExecutionName,
	}

	if testSuite.ExecutionRequest != nil {
		testExecution.Variables = testSuite.ExecutionRequest.Variables
	}

	// override variables from request
	for k, v := range request.Variables {
		testExecution.Variables[k] = v
	}

	// add queued execution steps
	batches := append(testSuite.Before, testSuite.Steps...)
	batches = append(batches, testSuite.After...)

	for i := range batches {
		var stepResults []TestSuiteStepExecutionResult
		for j := range batches[i].Execute {
			stepResults = append(stepResults, NewTestStepQueuedResult(&batches[i].Execute[j]))
		}

		testExecution.ExecuteStepResults = append(testExecution.ExecuteStepResults, TestSuiteBatchStepExecutionResult{
			Step:    &batches[i],
			Execute: stepResults,
		})
	}

	return testExecution
}

func (e TestSuiteExecution) FailedStepsCount() (count int) {
	for _, stepResult := range e.StepResults {
		if stepResult.Execution != nil && stepResult.Execution.IsFailed() {
			count++
		}
	}

	for _, batchStepResult := range e.ExecuteStepResults {
		for _, stepResult := range batchStepResult.Execute {
			if stepResult.Execution != nil && stepResult.Execution.IsFailed() {
				count++
				break
			}
		}
	}

	return
}

func (e TestSuiteExecution) IsCompleted() bool {
	if e.Status == nil {
		return false
	}

	return *e.Status == *TestSuiteExecutionStatusFailed ||
		*e.Status == *TestSuiteExecutionStatusPassed ||
		*e.Status == *TestSuiteExecutionStatusAborted ||
		*e.Status == *TestSuiteExecutionStatusTimeout
}

func (e *TestSuiteExecution) Stop() {
	duration := e.CalculateDuration()
	e.EndTime = time.Now()
	e.Duration = utils.RoundDuration(duration).String()
	e.DurationMs = int32(duration.Milliseconds())
}

func (e *TestSuiteExecution) CalculateDuration() time.Duration {
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

func (e TestSuiteExecution) Table() (header []string, output [][]string) {
	if len(e.StepResults) != 0 {
		header = []string{"Status", "Step", "ID", "Error"}
		output = make([][]string, 0)

		for _, sr := range e.StepResults {
			status := "no-execution-result"
			if sr.Execution != nil && sr.Execution.ExecutionResult != nil && sr.Execution.ExecutionResult.Status != nil {
				status = string(*sr.Execution.ExecutionResult.Status)
			}

			if sr.Step == nil {
				continue
			}

			switch sr.Step.Type() {
			case TestSuiteStepTypeExecuteTest:
				var id, errorMessage string
				if sr.Execution != nil && sr.Execution.ExecutionResult != nil {
					errorMessage = sr.Execution.ExecutionResult.ErrorMessage
					id = sr.Execution.Id
				}
				row := []string{status, sr.Step.FullName(), id, errorMessage}
				output = append(output, row)
			case TestSuiteStepTypeDelay:
				row := []string{status, sr.Step.FullName(), "", ""}
				output = append(output, row)
			}
		}
	}

	if len(e.ExecuteStepResults) != 0 {
		header = []string{"Statuses", "Step", "IDs", "Errors"}
		output = make([][]string, 0)

		for _, bs := range e.ExecuteStepResults {
			var statuses, names, ids, errorMessages []string

			for _, sr := range bs.Execute {
				status := "no-execution-result"
				if sr.Execution != nil && sr.Execution.ExecutionResult != nil && sr.Execution.ExecutionResult.Status != nil {
					status = string(*sr.Execution.ExecutionResult.Status)
				}

				statuses = append(statuses, status)
				if sr.Step == nil {
					continue
				}

				switch sr.Step.Type() {
				case TestSuiteStepTypeExecuteTest:
					var id, errorMessage string
					if sr.Execution != nil && sr.Execution.ExecutionResult != nil {
						errorMessage = sr.Execution.ExecutionResult.ErrorMessage
						id = sr.Execution.Id
					}

					names = append(names, sr.Step.FullName())
					ids = append(ids, id)
					errorMessages = append(errorMessages, fmt.Sprintf("%q", errorMessage))
				case TestSuiteStepTypeDelay:
					names = append(names, sr.Step.FullName())
					ids = append(ids, "\"\"")
					errorMessages = append(errorMessages, "\"\"")
				}
			}

			row := []string{strings.Join(statuses, ", "), strings.Join(names, ", "), strings.Join(ids, ", "), strings.Join(errorMessages, ", ")}
			output = append(output, row)
		}
	}

	return
}

func (e *TestSuiteExecution) IsRunning() bool {
	return e.Status != nil && *e.Status == RUNNING_TestSuiteExecutionStatus
}

func (e *TestSuiteExecution) IsQueued() bool {
	return e.Status != nil && *e.Status == QUEUED_TestSuiteExecutionStatus
}

func (e *TestSuiteExecution) IsPassed() bool {
	return e.Status != nil && *e.Status == PASSED_TestSuiteExecutionStatus
}

func (e *TestSuiteExecution) IsFailed() bool {
	return e.Status != nil && *e.Status == FAILED_TestSuiteExecutionStatus
}

func (e *TestSuiteExecution) IsAborted() bool {
	return *e.Status == ABORTED_TestSuiteExecutionStatus
}

func (e *TestSuiteExecution) IsTimeout() bool {
	return *e.Status == TIMEOUT_TestSuiteExecutionStatus
}

func (e *TestSuiteExecution) convertDots(fn func(string) string) *TestSuiteExecution {
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

func (e *TestSuiteExecution) EscapeDots() *TestSuiteExecution {
	return e.convertDots(utils.EscapeDots)
}

func (e *TestSuiteExecution) UnscapeDots() *TestSuiteExecution {
	return e.convertDots(utils.UnescapeDots)
}

func (e *TestSuiteExecution) CleanStepsOutput() *TestSuiteExecution {
	for i := range e.StepResults {
		if e.StepResults[i].Execution != nil && e.StepResults[i].Execution.ExecutionResult != nil {
			e.StepResults[i].Execution.ExecutionResult.Output = ""
		}
	}

	for i := range e.ExecuteStepResults {
		for j := range e.ExecuteStepResults[i].Execute {
			if e.ExecuteStepResults[i].Execute[j].Execution != nil && e.ExecuteStepResults[i].Execute[j].Execution.ExecutionResult != nil {
				e.ExecuteStepResults[i].Execute[j].Execution.ExecutionResult.Output = ""
			}
		}
	}
	return e
}

func (e *TestSuiteExecution) TruncateErrorMessages(length int) *TestSuiteExecution {
	for _, bs := range e.ExecuteStepResults {
		for _, sr := range bs.Execute {
			if sr.Execution != nil && sr.Execution.ExecutionResult != nil && len(sr.Execution.ExecutionResult.ErrorMessage) > length {
				sr.Execution.ExecutionResult.ErrorMessage = sr.Execution.ExecutionResult.ErrorMessage[0:length]
			}
		}
	}
	return e
}
