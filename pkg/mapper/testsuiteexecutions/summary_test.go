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
			assert.Equal(t, result[i].Execution[j].Batch[0].Id, executions[i].StepResults[j].Execution.Id)
			assert.Equal(t, result[i].Execution[j].Batch[0].Name, executions[i].StepResults[j].Execution.Name)
			assert.Equal(t, result[i].Execution[j].Batch[0].TestName, executions[i].StepResults[j].Execution.TestName)
			assert.Equal(t, result[i].Execution[j].Batch[0].Status, executions[i].StepResults[j].Execution.ExecutionResult.Status)
			var tp *testkube.TestSuiteStepType
			if executions[i].StepResults[j].Step.Execute != nil {
				tp = testkube.TestSuiteStepTypeExecuteTest
			}

			if executions[i].StepResults[j].Step.Delay != nil {
				tp = testkube.TestSuiteStepTypeDelay
			}

			assert.Equal(t, result[i].Execution[j].Batch[0].Type_, tp)
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
			assert.Equal(t, result[i].Execution[j].Batch[0].Id, executions[i].BatchStepResults[j].Batch[0].Execution.Id)
			assert.Equal(t, result[i].Execution[j].Batch[0].Name, executions[i].BatchStepResults[j].Batch[0].Execution.Name)
			assert.Equal(t, result[i].Execution[j].Batch[0].TestName, executions[i].BatchStepResults[j].Batch[0].Execution.TestName)
			assert.Equal(t, result[i].Execution[j].Batch[0].Status, executions[i].BatchStepResults[j].Batch[0].Execution.ExecutionResult.Status)
			var tp *testkube.TestSuiteStepType
			if executions[i].BatchStepResults[j].Batch[0].Step.Execute != nil {
				tp = testkube.TestSuiteStepTypeExecuteTest
			}

			if executions[i].BatchStepResults[j].Batch[0].Step.Delay != nil {
				tp = testkube.TestSuiteStepTypeDelay
			}

			assert.Equal(t, result[i].Execution[j].Batch[0].Type_, tp)
		}
	}
}

func getExecutions() []testkube.TestSuiteExecution {
	stepResults1 := []testkube.TestSuiteStepExecutionResult{
		{
			Step: &testkube.TestSuiteStep{Execute: &testkube.TestSuiteStepExecuteTest{}},
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
		"tid1",
		"script1",
		&testkube.ObjectRef{
			"testkube",
			"testsuite1",
		},
		testkube.TestSuiteExecutionStatusFailed,
		map[string]string{"var": "key"},
		map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v1")},
		"secret-uuid",
		time.Now(),
		time.Now(),
		"",
		0,
		stepResults1,
		nil,
		map[string]string{"label": "value"},
	}

	execution1.Stop()
	stepResults2 := []testkube.TestSuiteStepExecutionResult{
		{
			Step: &testkube.TestSuiteStep{Execute: &testkube.TestSuiteStepExecuteTest{}},
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
		"tid2",
		"script2",
		&testkube.ObjectRef{
			"testkube",
			"testsuite2",
		},
		testkube.TestSuiteExecutionStatusPassed,
		map[string]string{"var": "key"},
		map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v2")},
		"secret-uuid",
		time.Now(),
		time.Now(),
		"",
		0,
		stepResults2,
		nil,
		map[string]string{"label": "value"},
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
			Batch: []testkube.TestSuiteStepExecutionResult{
				{
					Step: &testkube.TestSuiteStep{Execute: &testkube.TestSuiteStepExecuteTest{}},
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
		"tid1",
		"script1",
		&testkube.ObjectRef{
			"testkube",
			"testsuite1",
		},
		testkube.TestSuiteExecutionStatusFailed,
		map[string]string{"var": "key"},
		map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v1")},
		"secret-uuid",
		time.Now(),
		time.Now(),
		"",
		0,
		nil,
		stepResults1,
		map[string]string{"label": "value"},
	}

	execution1.Stop()
	stepResults2 := []testkube.TestSuiteBatchStepExecutionResult{
		{
			Batch: []testkube.TestSuiteStepExecutionResult{
				{
					Step: &testkube.TestSuiteStep{Execute: &testkube.TestSuiteStepExecuteTest{}},
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
		"tid2",
		"script2",
		&testkube.ObjectRef{
			"testkube",
			"testsuite2",
		},
		testkube.TestSuiteExecutionStatusPassed,
		map[string]string{"var": "key"},
		map[string]testkube.Variable{"p": testkube.NewBasicVariable("p", "v2")},
		"secret-uuid",
		time.Now(),
		time.Now(),
		"",
		0,
		nil,
		stepResults2,
		map[string]string{"label": "value"},
	}

	execution2.Stop()

	return []testkube.TestSuiteExecution{
		execution1,
		execution2,
	}

}
