package executiondata

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

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

func TestExecutionFunction(t *testing.T) {
	registry := NewRegistry()
	registry.Add(Execution{
		Id:       "exec-1",
		Name:     "producer-1",
		Workflow: "producer",
		Alias:    "p",
		Status:   "passed",
		Outputs:  map[string]string{"token": "abc123"},
	})
	machine := NewMachine(MachineOptions{Registry: registry})

	t.Run("reads an output by alias", func(t *testing.T) {
		value, err := resolve(t, `execution("p").outputs.token`, machine)
		require.NoError(t, err)
		assert.Equal(t, "abc123", value)
	})

	t.Run("reads an output by workflow name", func(t *testing.T) {
		value, err := resolve(t, `execution("producer").outputs.token`, machine)
		require.NoError(t, err)
		assert.Equal(t, "abc123", value)
	})

	t.Run("reads an output by execution id", func(t *testing.T) {
		value, err := resolve(t, `execution("exec-1").outputs.token`, machine)
		require.NoError(t, err)
		assert.Equal(t, "abc123", value)
	})

	t.Run("exposes execution metadata", func(t *testing.T) {
		value, err := resolve(t, `execution("p").status`, machine)
		require.NoError(t, err)
		assert.Equal(t, "passed", value)

		value, err = resolve(t, `execution("p").id`, machine)
		require.NoError(t, err)
		assert.Equal(t, "exec-1", value)
	})

	t.Run("missing output resolves to null instead of failing", func(t *testing.T) {
		compiled, err := expressions.CompileAndResolve(`execution("p").outputs.missing`, machine)
		require.NoError(t, err)
		assert.Equal(t, "null", compiled.String())
	})

	t.Run("unknown reference lists what is available", func(t *testing.T) {
		_, err := resolve(t, `execution("nope").outputs.token`, machine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown execution "nope"`)
		assert.Contains(t, err.Error(), "available executions are p")
	})

	t.Run("reports an empty registry as nothing executed yet", func(t *testing.T) {
		_, err := resolve(t, `execution("nope").outputs.token`, NewMachine(MachineOptions{Registry: NewRegistry()}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "run it in an earlier step")
	})

	t.Run("rejects malformed arguments", func(t *testing.T) {
		_, err := resolve(t, `execution()`, machine)
		assert.ErrorContains(t, err, "expects 1-2 arguments")

		_, err = resolve(t, `execution("")`, machine)
		assert.ErrorContains(t, err, "expects a non-empty reference")

		_, err = resolve(t, `execution("p", "second")`, machine)
		assert.ErrorContains(t, err, "expects the index to be a number")
	})
}

func TestExecutionFunctionFanOut(t *testing.T) {
	registry := NewRegistry()
	registry.Add(Execution{Id: "exec-1", Workflow: "shard", Alias: "s", Index: 0, Outputs: map[string]string{"n": "first"}})
	registry.Add(Execution{Id: "exec-2", Workflow: "shard", Alias: "s", Index: 1, Outputs: map[string]string{"n": "second"}})
	machine := NewMachine(MachineOptions{Registry: registry})

	value, err := resolve(t, `execution("s").outputs.n`, machine)
	require.NoError(t, err)
	assert.Equal(t, "first", value, "no index should address the first instance")

	value, err = resolve(t, `execution("s", 1).outputs.n`, machine)
	require.NoError(t, err)
	assert.Equal(t, "second", value)

	_, err = resolve(t, `execution("s", 2).outputs.n`, machine)
	assert.ErrorContains(t, err, `unknown execution "s" at index 2`)
}

func TestExecutionFunctionParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)

	t.Run("resolves the closest parent", func(t *testing.T) {
		repository.EXPECT().Get(gomock.Any(), "parent-2").
			Return(Execution{Id: "parent-2", Outputs: map[string]string{"seed": "42"}}, nil)

		machine := NewMachine(MachineOptions{
			Registry:   NewRegistry(),
			Repository: repository,
			ParentIds:  []string{"parent-1", "parent-2"},
		})
		value, err := resolve(t, `execution("parent").outputs.seed`, machine)
		require.NoError(t, err)
		assert.Equal(t, "42", value)
	})

	t.Run("explains that there is no parent", func(t *testing.T) {
		machine := NewMachine(MachineOptions{Registry: NewRegistry(), Repository: repository})
		_, err := resolve(t, `execution("parent").outputs.seed`, machine)
		assert.ErrorContains(t, err, "this execution has no parent")
	})

	t.Run("wraps control plane failures", func(t *testing.T) {
		repository.EXPECT().Get(gomock.Any(), "parent-1").Return(Execution{}, errors.New("connection refused"))

		machine := NewMachine(MachineOptions{Repository: repository, ParentIds: []string{"parent-1"}})
		_, err := resolve(t, `execution("parent").outputs.seed`, machine)
		assert.ErrorContains(t, err, "reading parent execution parent-1: connection refused")
	})
}

func TestExecutionFunctionUnregistered(t *testing.T) {
	// Workflow specs are resolved several times before the execution data exists.
	// An unregistered execution() must survive those passes untouched, the same way
	// credential() does, instead of failing the whole workflow.
	compiled, err := expressions.CompileAndResolve(`execution("p").outputs.token`, expressions.NewMachine())
	require.NoError(t, err)
	assert.Nil(t, compiled.Static())
	assert.Contains(t, compiled.String(), `execution("p")`)
}

// A workflow only registers the test workflows it ran itself. Everything else -
// notably a sibling of the same suite, whose id the parent passed down - is
// addressed by execution id and resolved through the control plane.
func TestExecutionRepositoryFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)
	repository.EXPECT().Get(gomock.Any(), "other-execution").
		Return(Execution{Id: "other-execution", Outputs: map[string]string{"key": "value"}}, nil)

	machine := NewMachine(MachineOptions{Registry: NewRegistry(), Repository: repository})
	value, err := resolve(t, `execution("other-execution").outputs.key`, machine)
	require.NoError(t, err)
	assert.Equal(t, "value", value)
}

// The same reference works for artifacts, so a sibling reads both the values and
// the files of another test workflow of its suite.
func TestSiblingByExecutionId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("produced by the sibling"))
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)
	repository.EXPECT().Get(gomock.Any(), "exec-producer").
		Return(Execution{Id: "exec-producer", Outputs: map[string]string{"token": "abc"}}, nil).Times(2)
	repository.EXPECT().ListArtifacts(gomock.Any(), "exec-producer", []string{"results/summary.json"}).
		Return([]Artifact{{Path: "results/summary.json", Url: server.URL, Size: 23}}, nil)

	// The registry is empty: a child never scheduled anything itself.
	machine := NewMachine(MachineOptions{Registry: NewRegistry(), Repository: repository})

	value, err := resolve(t, `execution("exec-producer").outputs.token`, machine)
	require.NoError(t, err)
	assert.Equal(t, "abc", value)

	value, err = resolve(t, `read_artifact("exec-producer", "results/summary.json")`, machine)
	require.NoError(t, err)
	assert.Equal(t, "produced by the sibling", value)
}

// The id arrives as a config value rather than a literal, so the reference has to
// survive being an accessor.
func TestSiblingReferenceFromConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	repository := NewMockExecutionRepository(ctrl)
	repository.EXPECT().Get(gomock.Any(), "exec-producer").
		Return(Execution{Id: "exec-producer", Outputs: map[string]string{"token": "abc"}}, nil)

	machine := expressions.CombinedMachines(
		expressions.NewMachine().RegisterStringMap("config", map[string]string{"producerId": "exec-producer"}),
		NewMachine(MachineOptions{Registry: NewRegistry(), Repository: repository}),
	)

	value, err := resolve(t, `execution(config.producerId).outputs.token`, machine)
	require.NoError(t, err)
	assert.Equal(t, "abc", value)
}

func TestExecutionRepositoryNotConfigured(t *testing.T) {
	machine := NewMachine(MachineOptions{Registry: NewRegistry()})
	_, err := resolve(t, `execution("some-id").outputs.key`, machine)
	assert.ErrorContains(t, err, `unknown execution "some-id"`)
}
