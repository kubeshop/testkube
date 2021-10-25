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
		assert.Equal(t, result[i].ScriptName, executions[i].ScriptName)
		assert.Equal(t, result[i].ScriptType, executions[i].ScriptType)
		assert.Equal(t, result[i].Status, executions[i].ExecutionResult.Status)
		assert.Equal(t, result[i].StartTime, executions[i].ExecutionResult.StartTime)
		assert.Equal(t, result[i].EndTime, executions[i].ExecutionResult.EndTime)
	}
}

func getExecutions() testkube.Executions {
	ex1 := new(testkube.ExecutionResult)
	ex1.Start()
	ex1.Stop()

	execution1 := testkube.NewExecution(
		"script1",
		"execution1",
		"test/test",
		"",
		*ex1,
		map[string]string{"p": "v1"},
	)
	ex2 := new(testkube.ExecutionResult)
	ex2.Start()
	ex2.Stop()

	execution2 := testkube.NewExecution(
		"script1",
		"execution2",
		"test/test",
		"",
		*ex2,
		map[string]string{"p": "v2"},
	)

	return testkube.Executions{
		execution1,
		execution2,
	}

}
