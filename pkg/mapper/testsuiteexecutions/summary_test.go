package testsuiteexecutions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestSuiteMapToSummary(t *testing.T) {
	// given
	executions := getExecutions()

	// when
	var result []*testkube.TestSuiteExecutionSummary
	for i := range executions {
		result = append(result, MapToSummary(&executions[i]))
	}

	// then - test mappings
	for i := 0; i < len(executions); i++ {
		assert.Equal(t, result[i].Id, executions[i].Id)
		assert.Equal(t, result[i].Name, executions[i].Name)
		assert.Equal(t, result[i].TestSuiteName, executions[i].TestSuite.Name)
		assert.Equal(t, result[i].Status, executions[i].Status)
		assert.Equal(t, result[i].StartTime, executions[i].StartTime)
		assert.Equal(t, result[i].EndTime, executions[i].EndTime)
		assert.Equal(t, result[i].Duration, executions[i].Duration)
		assert.Equal(t, result[i].DurationMs, executions[i].DurationMs)
		for j := range result[i].Execution {
			assert.Equal(t, result[i].Execution[j].Execute[0].Id, executions[i].StepResults[j].Execution.Id)
			assert.Equal(t, result[i].Execution[j].Execute[0].Name, executions[i].StepResults[j].Execution.Name)
			assert.Equal(t, result[i].Execution[j].Execute[0].TestName, executions[i].StepResults[j].Execution.TestName)
			assert.Equal(t, result[i].Execution[j].Execute[0].Status, executions[i].StepResults[j].Execution.ExecutionResult.Status)
			var tp *testkube.TestSuiteStepType
			if executions[i].StepResults[j].Step.Execute != nil {
				tp = testkube.TestSuiteStepTypeExecuteTest
			}

			if executions[i].StepResults[j].Step.Delay != nil {
				tp = testkube.TestSuiteStepTypeDelay
			}

			assert.Equal(t, result[i].Execution[j].Execute[0].Type_, tp)
		}
	}
}

func TestSuiteMapBatchToSummary(t *testing.T) {
	// given
	executions := getBatchExecutions()

	// when
	var result []*testkube.TestSuiteExecutionSummary
	for i := range executions {
		result = append(result, MapToSummary(&executions[i]))
	}

	// then - test mappings
	for i := 0; i < len(executions); i++ {
		assert.Equal(t, result[i].Id, executions[i].Id)
		assert.Equal(t, result[i].Name, executions[i].Name)
		assert.Equal(t, result[i].TestSuiteName, executions[i].TestSuite.Name)
		assert.Equal(t, result[i].Status, executions[i].Status)
		assert.Equal(t, result[i].StartTime, executions[i].StartTime)
		assert.Equal(t, result[i].EndTime, executions[i].EndTime)
		assert.Equal(t, result[i].Duration, executions[i].Duration)
		assert.Equal(t, result[i].DurationMs, executions[i].DurationMs)
		for j := range result[i].Execution {
			assert.Equal(t, result[i].Execution[j].Execute[0].Id, executions[i].ExecuteStepResults[j].Execute[0].Execution.Id)
			assert.Equal(t, result[i].Execution[j].Execute[0].Name, executions[i].ExecuteStepResults[j].Execute[0].Execution.Name)
			assert.Equal(t, result[i].Execution[j].Execute[0].TestName, executions[i].ExecuteStepResults[j].Execute[0].Execution.TestName)
			assert.Equal(t, result[i].Execution[j].Execute[0].Status, executions[i].ExecuteStepResults[j].Execute[0].Execution.ExecutionResult.Status)
			var tp *testkube.TestSuiteStepType
			if executions[i].ExecuteStepResults[j].Execute[0].Step.Test != "" {
				tp = testkube.TestSuiteStepTypeExecuteTest
			}

			if executions[i].ExecuteStepResults[j].Execute[0].Step.Delay != "" {
				tp = testkube.TestSuiteStepTypeDelay
			}

			assert.Equal(t, result[i].Execution[j].Execute[0].Type_, tp)
		}
	}
}

func getExecutions() []testkube.TestSuiteExecution {
	stepResults1 := []testkube.TestSuiteStepExecutionResultV2{
		{
			Step: &testkube.TestSuiteStepV2{Execute: &testkube.TestSuiteStepExecuteTestV2{Name: "testname1"}},
			Execution: &testkube.Execution{
				Id:       "id1",
				Name:     "name1",
				TestName: "testname1",
				ExecutionResult: &testkube.ExecutionResult{
					Status: testkube.ExecutionStatusPassed,
				},
			},
		},
	}

	execution1 := testkube.TestSuiteExecution{
		Id:   "tid1",
		Name: "script1",
		TestSuite: &testkube.ObjectRef{
			Namespace: "testkube",
			Name:      "testsuite1",
		},
		Status:      testkube.TestSuiteExecutionStatusFailed,
		Envs:        map[string]string{"var": "key"},
		Variables:   map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v1")},
		SecretUUID:  "secret-uuid",
		StartTime:   time.Now(),
		EndTime:     time.Now(),
		StepResults: stepResults1,
		Labels:      map[string]string{"label": "value"},
	}

	execution1.Stop()
	stepResults2 := []testkube.TestSuiteStepExecutionResultV2{
		{
			Step: &testkube.TestSuiteStepV2{Execute: &testkube.TestSuiteStepExecuteTestV2{Name: "testname2"}},
			Execution: &testkube.Execution{
				Id:       "id2",
				Name:     "name2",
				TestName: "testname2",
				ExecutionResult: &testkube.ExecutionResult{
					Status: testkube.ExecutionStatusFailed,
				},
			},
		},
	}

	execution2 := testkube.TestSuiteExecution{
		Id:   "tid2",
		Name: "script2",
		TestSuite: &testkube.ObjectRef{
			Namespace: "testkube",
			Name:      "testsuite2",
		},
		Status:      testkube.TestSuiteExecutionStatusPassed,
		Envs:        map[string]string{"var": "key"},
		Variables:   map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v2")},
		SecretUUID:  "secret-uuid",
		StartTime:   time.Now(),
		EndTime:     time.Now(),
		StepResults: stepResults2,
		Labels:      map[string]string{"label": "value"},
	}

	execution2.Stop()

	return []testkube.TestSuiteExecution{
		execution1,
		execution2,
	}

}

func getBatchExecutions() []testkube.TestSuiteExecution {
	stepResults1 := []testkube.TestSuiteBatchStepExecutionResult{
		{
			Execute: []testkube.TestSuiteStepExecutionResult{
				{
					Step: &testkube.TestSuiteStep{Test: "testname1"},
					Execution: &testkube.Execution{
						Id:       "id1",
						Name:     "name1",
						TestName: "testname1",
						ExecutionResult: &testkube.ExecutionResult{
							Status: testkube.ExecutionStatusPassed,
						},
					},
				},
			},
		},
	}

	execution1 := testkube.TestSuiteExecution{
		Id:   "tid1",
		Name: "script1",
		TestSuite: &testkube.ObjectRef{
			Namespace: "testkube",
			Name:      "testsuite1",
		},
		Status:             testkube.TestSuiteExecutionStatusFailed,
		Envs:               map[string]string{"var": "key"},
		Variables:          map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v1")},
		SecretUUID:         "secret-uuid",
		StartTime:          time.Now(),
		EndTime:            time.Now(),
		ExecuteStepResults: stepResults1,
		Labels:             map[string]string{"label": "value"},
	}

	execution1.Stop()
	stepResults2 := []testkube.TestSuiteBatchStepExecutionResult{
		{
			Execute: []testkube.TestSuiteStepExecutionResult{
				{
					Step: &testkube.TestSuiteStep{Test: "testname2"},
					Execution: &testkube.Execution{
						Id:       "id2",
						Name:     "name2",
						TestName: "testname2",
						ExecutionResult: &testkube.ExecutionResult{
							Status: testkube.ExecutionStatusFailed,
						},
					},
				},
			},
		},
	}

	execution2 := testkube.TestSuiteExecution{
		Id:   "tid2",
		Name: "script2",
		TestSuite: &testkube.ObjectRef{
			Namespace: "testkube",
			Name:      "testsuite2",
		},
		Status:             testkube.TestSuiteExecutionStatusPassed,
		Envs:               map[string]string{"var": "key"},
		Variables:          map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v2")},
		SecretUUID:         "secret-uuid",
		StartTime:          time.Now(),
		EndTime:            time.Now(),
		ExecuteStepResults: stepResults2,
		Labels:             map[string]string{"label": "value"},
	}

	execution2.Stop()

	return []testkube.TestSuiteExecution{
		execution1,
		execution2,
	}

}
