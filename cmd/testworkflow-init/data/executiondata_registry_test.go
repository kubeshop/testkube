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

func TestExecutionRegistryCarriesAmbiguity(t *testing.T) {
	ClearState()
	t.Cleanup(ClearState)

	// Only one of the two unaliased "producer" entries was scheduled, so the group holds
	// a single run. The name must stay refused in the following steps all the same.
	GetState().SetOutput("step", executiondata.ExecutionInstructionName("producer"), executiondata.ExecutionGroup{
		Executions: []executiondata.Execution{{Id: "only", Workflow: "producer"}},
		Ambiguous:  true,
	})

	registry := ExecutionRegistry()

	_, ok, err := registry.Lookup("producer", 0)
	assert.False(t, ok)
	assert.ErrorContains(t, err, `ambiguous execution "producer"`)

	byId, ok, err := registry.Lookup("only", 0)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "only", byId.Id)
}
