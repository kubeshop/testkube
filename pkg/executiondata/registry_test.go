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

		execution, ok, err := registry.Lookup("s", 1)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "passed", execution.Status)
	})
}

// An id names one execution; an index only tells apart the members of a group. Filtering
// an id by index hid every member of a fan-out but the first, so an id was addressable
// only if its position happened to match the index the caller asked for - which, for the
// common execution("<id>") with no index, meant only index 0.
func TestRegistryLookupById(t *testing.T) {
	registry := NewRegistry()
	registry.Add(Execution{Id: "shard-0", Workflow: "shard", Alias: "s", Index: 0})
	registry.Add(Execution{Id: "shard-1", Workflow: "shard", Alias: "s", Index: 1})
	registry.Add(Execution{Id: "shard-2", Workflow: "shard", Alias: "s", Index: 2})

	t.Run("an id past the first position resolves without an index", func(t *testing.T) {
		for _, id := range []string{"shard-0", "shard-1", "shard-2"} {
			execution, ok, err := registry.Lookup(id, 0)
			require.NoError(t, err, id)
			require.True(t, ok, id)
			assert.Equal(t, id, execution.Id)
		}
	})

	t.Run("the entry keeps its position and alias", func(t *testing.T) {
		// Resolving through the control plane instead would answer with a record that
		// carries neither, so the local entry has to be the one that is found.
		execution, ok, err := registry.Lookup("shard-2", 0)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, int64(2), execution.Index)
		assert.Equal(t, "s", execution.Alias)
	})

	t.Run("an index alongside an id is not consulted", func(t *testing.T) {
		// An unspecified index and an explicit 0 are the same argument here, so an index
		// cannot contradict an id - the id names the execution on its own.
		execution, ok, err := registry.Lookup("shard-1", 7)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "shard-1", execution.Id)
	})

	t.Run("a group key still addresses a position", func(t *testing.T) {
		execution, ok, err := registry.Lookup("s", 2)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "shard-2", execution.Id)

		_, ok, err = registry.Lookup("s", 3)
		require.NoError(t, err)
		assert.False(t, ok, "the group has three members")
	})

	t.Run("an unknown id resolves to nothing rather than erroring", func(t *testing.T) {
		// The caller falls back to the control plane, which is what serves an id this
		// workflow did not schedule itself.
		_, ok, err := registry.Lookup("shard-9", 0)
		require.NoError(t, err)
		assert.False(t, ok)
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

	_, ok, err := registry.Lookup("producer", 0)
	require.NoError(t, err)
	assert.False(t, ok)

	assert.Len(t, registry.Group("keep-me"), 1, "resetting one reference must not touch the others")

	registry.Add(Execution{Id: "new", Workflow: "producer"})
	execution, ok, err := registry.Lookup("producer", 0)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "new", execution.Id)
}

// An aliased entry stays addressable by the workflow it ran, so a workflow name can end
// up addressing two different executions. Resolving that by insertion order would feed
// later configuration the outputs of an arbitrary child.
func TestRegistryLookupAmbiguity(t *testing.T) {
	t.Run("a selector alias and an unaliased entry on the same workflow", func(t *testing.T) {
		registry := NewRegistry()
		// `as: all` over a selector matching a and b, then a separate entry running a.
		registry.Add(Execution{Id: "all-0", Workflow: "a", Alias: "all", Index: 0})
		registry.Add(Execution{Id: "all-1", Workflow: "b", Alias: "all", Index: 1})
		registry.Add(Execution{Id: "own-a", Workflow: "a", Index: 0})

		_, ok, err := registry.Lookup("a", 0)
		assert.False(t, ok)
		require.Error(t, err)
		assert.ErrorContains(t, err, `ambiguous execution "a"`)
		assert.ErrorContains(t, err, `the "a" run aliased "all"`)
		assert.ErrorContains(t, err, `the unaliased "a" run`)

		t.Run("the unambiguous references still resolve", func(t *testing.T) {
			execution, ok, err := registry.Lookup("all", 0)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "all-0", execution.Id)

			execution, ok, err = registry.Lookup("own-a", 0)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "own-a", execution.Id)
		})
	})

	t.Run("two aliased entries running the same workflow", func(t *testing.T) {
		// Legitimate: told apart by their aliases. Only the workflow name is ambiguous,
		// and only when it is the reference actually used.
		registry := NewRegistry()
		registry.Add(Execution{Id: "x-0", Workflow: "a", Alias: "x", Index: 0})
		registry.Add(Execution{Id: "y-0", Workflow: "a", Alias: "y", Index: 0})

		execution, ok, err := registry.Lookup("x", 0)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "x-0", execution.Id)

		_, _, err = registry.Lookup("a", 0)
		assert.ErrorContains(t, err, `ambiguous execution "a"`)
	})

	t.Run("a fan-out group is not ambiguous with itself", func(t *testing.T) {
		// Same workflow at several indexes: each position answers on its own.
		registry := NewRegistry()
		registry.Add(Execution{Id: "s-0", Workflow: "shard", Alias: "s", Index: 0})
		registry.Add(Execution{Id: "s-1", Workflow: "shard", Alias: "s", Index: 1})

		execution, ok, err := registry.Lookup("shard", 1)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, "s-1", execution.Id)
	})
}

func TestRegistryKey(t *testing.T) {
	assert.Equal(t, "alias", Execution{Alias: "alias", Workflow: "workflow"}.Key())
	assert.Equal(t, "workflow", Execution{Workflow: "workflow"}.Key())
}

func TestExecutionInstructionName(t *testing.T) {
	assert.Equal(t, "executiondata.my-workflow", ExecutionInstructionName("my-workflow"))
	assert.Equal(t, "executiondata.odd_name", ExecutionInstructionName("odd.name"),
		"characters the instruction grammar rejects must not leak into the name")

	// The dashboard reads every instruction named ^testworkflow(-.*)?$ as the status of
	// one child execution. This one carries a list, so it has to stay out of that family.
	assert.NotRegexp(t, `^testworkflow(-.*)?$`, ExecutionInstructionName("my-workflow"))
	assert.NotRegexp(t, `^test(-.*)?$`, ExecutionInstructionName("my-workflow"))
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
