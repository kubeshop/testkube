package paralleldata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func resolve(t *testing.T, expression string, machine expressions.Machine) (interface{}, error) {
	t.Helper()
	compiled, err := expressions.CompileAndResolve(expression, machine)
	if err != nil {
		return nil, err
	}
	require.NotNil(t, compiled.Static(), "expression was not resolved: %s", compiled.String())
	return compiled.Static().Value(), nil
}

func TestParallelFunction(t *testing.T) {
	machine := NewMachine(map[string]Group{
		"shard": {
			Count: 3,
			Workers: []Worker{
				{Index: 0, Status: testkube.PASSED_TestWorkflowStatus, Outputs: map[string]string{"duration": "12"}},
				{Index: 1, Status: testkube.FAILED_TestWorkflowStatus, Outputs: map[string]string{"duration": "7"}},
			},
		},
		"other": {Count: 1, Workers: []Worker{{Index: 0}}},
	})

	t.Run("reads an output of a worker", func(t *testing.T) {
		value, err := resolve(t, `parallel("shard", 0).outputs.duration`, machine)
		require.NoError(t, err)
		assert.Equal(t, "12", value)
	})

	t.Run("a failed worker keeps what it published", func(t *testing.T) {
		value, err := resolve(t, `parallel("shard", 1).outputs.duration`, machine)
		require.NoError(t, err)
		assert.Equal(t, "7", value)

		value, err = resolve(t, `parallel("shard", 1).status`, machine)
		require.NoError(t, err)
		assert.Equal(t, "failed", value)
	})

	t.Run("an expected worker that did not start has no status", func(t *testing.T) {
		value, err := resolve(t, `parallel("shard", 2).status`, machine)
		require.NoError(t, err)
		assert.Equal(t, "", value)

		compiled, err := expressions.CompileAndResolve(`parallel("shard", 2).outputs.duration`, machine)
		require.NoError(t, err)
		assert.Equal(t, "null", compiled.String())
	})

	t.Run("exposes the index", func(t *testing.T) {
		value, err := resolve(t, `parallel("shard", 1).index`, machine)
		require.NoError(t, err)
		assert.EqualValues(t, 1, value)
	})

	t.Run("an index out of range is an error", func(t *testing.T) {
		_, err := resolve(t, `parallel("shard", 3).outputs.duration`, machine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `parallel step "shard" has 3 workers, there is no worker 3`)

		_, err = resolve(t, `parallel("shard", -1).outputs.duration`, machine)
		require.Error(t, err)
	})

	t.Run("an unknown step lists what is available", func(t *testing.T) {
		_, err := resolve(t, `parallel("nope", 0).outputs.duration`, machine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown parallel step "nope", available parallel steps are other, shard`)
	})

	t.Run("reports when no parallel step has finished", func(t *testing.T) {
		_, err := resolve(t, `parallel("shard", 0).outputs.duration`, NewMachine(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no parallel step has finished yet")
	})

	t.Run("requires the index", func(t *testing.T) {
		_, err := resolve(t, `parallel("shard").outputs.duration`, machine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expects 2 arguments, 1 provided")
	})

	t.Run("rejects a reference that is not a string", func(t *testing.T) {
		_, err := resolve(t, `parallel(1, 0)`, machine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expects the reference to be a string")
	})
}

func TestScopedParallelFunction(t *testing.T) {
	groups := map[string]Group{
		"outer": {Count: 1, Workers: []Worker{{Index: 0, Outputs: map[string]string{"seed": "from-parent"}}}},
		"inner": {Count: 1, Workers: []Worker{{Index: 0, Outputs: map[string]string{"seed": "parent-inner"}}}},
	}
	machine := NewScopedMachine(groups, map[string]bool{"inner": true})

	t.Run("resolves a group that finished here", func(t *testing.T) {
		value, err := resolve(t, `parallel("outer", 0).outputs.seed`, machine)
		require.NoError(t, err)
		assert.Equal(t, "from-parent", value)
	})

	t.Run("leaves a step the specification declares to the specification", func(t *testing.T) {
		compiled, err := expressions.CompileAndResolve(`parallel("inner", 0).outputs.seed`, machine)
		require.NoError(t, err)
		assert.Nil(t, compiled.Static())
		assert.Contains(t, compiled.String(), `parallel("inner",0)`)
	})

	t.Run("leaves an unknown step to the specification", func(t *testing.T) {
		compiled, err := expressions.CompileAndResolve(`parallel("later", 0).outputs.seed`, machine)
		require.NoError(t, err)
		assert.Nil(t, compiled.Static())
	})

	t.Run("still refuses an index outside a group that finished here", func(t *testing.T) {
		_, err := resolve(t, `parallel("outer", 1).outputs.seed`, machine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "there is no worker 1")
	})
}
