package executiondata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestRegistryGroups(t *testing.T) {
	registry := NewRegistry()
	registry.Add(Execution{Id: "a-0", Workflow: "shard", Alias: "s", Index: 0})
	registry.Add(Execution{Id: "a-1", Workflow: "shard", Alias: "s", Index: 1})
	registry.Add(Execution{Id: "b-0", Workflow: "other"})

	assert.Equal(t, []string{"s", "other"}, registry.Refs())
	assert.Len(t, registry.Group("s"), 2)
	assert.Len(t, registry.Group("other"), 1)

	t.Run("re-registering the same position replaces it", func(t *testing.T) {
		registry.Add(Execution{Id: "a-1", Workflow: "shard", Alias: "s", Index: 1, Status: "passed"})
		require.Len(t, registry.Group("s"), 2)

		execution, ok := registry.Lookup("s", 1)
		require.True(t, ok)
		assert.Equal(t, "passed", execution.Status)
	})
}

func TestRegistryReset(t *testing.T) {
	// A later step running the same workflow takes the reference over, so it must be
	// able to drop what an earlier step left behind before indexing into the group.
	registry := NewRegistry()
	registry.Add(Execution{Id: "old", Workflow: "producer"})
	registry.Add(Execution{Id: "other", Workflow: "keep-me"})

	registry.Reset("producer")
	assert.Empty(t, registry.Group("producer"))

	_, ok := registry.Lookup("producer", 0)
	assert.False(t, ok)

	assert.Len(t, registry.Group("keep-me"), 1, "resetting one reference must not touch the others")

	registry.Add(Execution{Id: "new", Workflow: "producer"})
	execution, ok := registry.Lookup("producer", 0)
	require.True(t, ok)
	assert.Equal(t, "new", execution.Id)
}

func TestRegistryKey(t *testing.T) {
	assert.Equal(t, "alias", Execution{Alias: "alias", Workflow: "workflow"}.Key())
	assert.Equal(t, "workflow", Execution{Workflow: "workflow"}.Key())
}

func TestExecutionInstructionName(t *testing.T) {
	assert.Equal(t, "testworkflow-execution.my-workflow", ExecutionInstructionName("my-workflow"))
	assert.Equal(t, "testworkflow-execution.odd_name", ExecutionInstructionName("odd.name"),
		"characters the instruction grammar rejects must not leak into the name")
}

func TestOutputsOf(t *testing.T) {
	execution := &testkube.TestWorkflowExecution{
		Output: []testkube.TestWorkflowOutput{
			{Ref: "step-1", Name: "testworkflow-start", Value: map[string]interface{}{"id": "ignored"}},
			{Ref: "step-1", Name: OutputsInstructionName, Value: map[string]interface{}{"token": "abc", "count": "1"}},
			{Ref: "step-2", Name: OutputsInstructionName, Value: map[string]interface{}{"token": "overridden"}},
		},
	}

	assert.Equal(t, map[string]string{"token": "overridden", "count": "1"}, OutputsOf(execution),
		"only outputs instructions count, and a later step wins on a repeated key")
	assert.Nil(t, OutputsOf(nil))
}

// The execution() function shares its name with the execution.* accessor. The
// expression machine keeps functions and accessors apart, and combining both
// machines must keep each working.
func TestExecutionFunctionCoexistsWithExecutionAccessor(t *testing.T) {
	registry := NewRegistry()
	registry.Add(Execution{Id: "child-1", Workflow: "producer", Outputs: map[string]string{"token": "abc"}})

	machine := expressions.CombinedMachines(
		testworkflowconfig.CreateExecutionMachine(&testworkflowconfig.ExecutionConfig{Id: "self", Name: "self-1"}),
		NewMachine(MachineOptions{Registry: registry}),
	)

	value, err := resolve(t, `execution.id`, machine)
	require.NoError(t, err)
	assert.Equal(t, "self", value)

	value, err = resolve(t, `execution("producer").outputs.token`, machine)
	require.NoError(t, err)
	assert.Equal(t, "abc", value)
}
