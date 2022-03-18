package executions

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestMapToSummary(t *testing.T) {
	// given
	executions := getExecutions()

	// when
	result := MapToSummary(executions)

	// then - test mappings
	for i := 0; i < len(executions); i++ {
		assert.Equal(t, result[i].Id, executions[i].Id)
		assert.Equal(t, result[i].Name, executions[i].Name)
		assert.Equal(t, result[i].TestName, executions[i].TestName)
		assert.Equal(t, result[i].TestType, executions[i].TestType)
		assert.Equal(t, result[i].Status, executions[i].ExecutionResult.Status)
		assert.Equal(t, result[i].StartTime, executions[i].StartTime)
		assert.Equal(t, result[i].EndTime, executions[i].EndTime)
	}
}

func getExecutions() testkube.Executions {
	ex1 := new(testkube.ExecutionResult)

	execution1 := testkube.NewExecution(
		"testkube",
		"script1",
		"execution1",
		"test/test",
		testkube.NewStringTestContent(""),
		*ex1,
		map[string]string{"p": "v1"},
		nil,
	)
	execution1.Start()
	execution1.Stop()
	ex2 := new(testkube.ExecutionResult)

	execution2 := testkube.NewExecution(
		"testkube",
		"script1",
		"execution2",
		"test/test",
		testkube.NewStringTestContent(""),
		*ex2,
		map[string]string{"p": "v2"},
		nil,
	)
	execution2.Start()
	execution2.Stop()

	return testkube.Executions{
		execution1,
		execution2,
	}

}
