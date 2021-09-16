package executions

import (
	"testing"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/stretchr/testify/assert"
)

func TestMapToSummary(t *testing.T) {
	// given
	executions := getScriptExecutions()

	// when
	result := MapToSummary(executions)

	// then - test mappings
	for i := 0; i < len(executions); i++ {
		assert.Equal(t, result[i].Id, executions[i].Id)
		assert.Equal(t, result[i].Name, executions[i].Name)
		assert.Equal(t, result[i].ScriptName, executions[i].ScriptName)
		assert.Equal(t, result[i].ScriptType, executions[i].ScriptType)
		assert.Equal(t, result[i].Status, executions[i].Execution.Status)
		assert.Equal(t, result[i].StartTime, executions[i].Execution.StartTime)
		assert.Equal(t, result[i].EndTime, executions[i].Execution.EndTime)
	}
}

func getScriptExecutions() kubtest.ScriptExecutions {
	ex1 := new(kubtest.Execution).
		WithContent("content1").
		WithParams(map[string]string{"p": "v1"})
	execution1 := kubtest.NewScriptExecution(
		"script1",
		"execution1",
		"test/test",
		*ex1,
		map[string]string{"p": "v1"},
	)
	ex2 := new(kubtest.Execution).
		WithContent("content1").
		WithParams(map[string]string{"p": "v1"})
	execution2 := kubtest.NewScriptExecution(
		"script1",
		"execution2",
		"test/test",
		*ex2,
		map[string]string{"p": "v2"},
	)

	return kubtest.ScriptExecutions{
		execution1,
		execution2,
	}

}
