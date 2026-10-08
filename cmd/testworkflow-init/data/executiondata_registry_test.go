package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/executiondata"
)

func TestExecutionRegistryReadsObjectAndArray(t *testing.T) {
	ClearState()
	t.Cleanup(ClearState)

	state := GetState()
	state.SetOutput("step", executiondata.ExecutionInstructionName("producer"), executiondata.ExecutionGroup{
		Executions: []executiondata.Execution{{Id: "new", Workflow: "producer"}},
	})
	state.SetOutput("step", executiondata.ExecutionInstructionName("legacy"), []executiondata.Execution{
		{Id: "old", Workflow: "legacy"},
	})

	registry := ExecutionRegistry()

	current, ok, err := registry.Lookup("producer", 0)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "new", current.Id)

	legacy, ok, err := registry.Lookup("legacy", 0)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "old", legacy.Id)
}
