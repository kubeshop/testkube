package testworkflowconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func TestCreateExecutionMachine_RunningContextVariables(t *testing.T) {
	actorType := testkube.USER_TestWorkflowRunningContextActorType
	interfaceType := testkube.CLI_TestWorkflowRunningContextInterfaceType

	cfg := &ExecutionConfig{
		Id: "test-id",
		RunningContext: &testkube.TestWorkflowRunningContext{
			Actor: &testkube.TestWorkflowRunningContextActor{
				Name:  "test-user",
				Type_: &actorType,
			},
			Interface_: &testkube.TestWorkflowRunningContextInterface{
				Type_: &interfaceType,
			},
		},
	}

	machine := CreateExecutionMachine(cfg)

	actorExpr, err := expressions.Compile("execution.runningContext.actor.name")
	require.NoError(t, err)
	actorResult, err := actorExpr.Resolve(machine)
	require.NoError(t, err)
	actorName, err := actorResult.Static().StringValue()
	require.NoError(t, err)
	require.Equal(t, "test-user", actorName)

	actorTypeExpr, err := expressions.Compile("execution.runningContext.actor.type")
	require.NoError(t, err)
	actorTypeResult, err := actorTypeExpr.Resolve(machine)
	require.NoError(t, err)
	actorTypeValue, err := actorTypeResult.Static().StringValue()
	require.NoError(t, err)
	require.Equal(t, string(actorType), actorTypeValue)

	interfaceExpr, err := expressions.Compile("execution.runningContext.interface.type")
	require.NoError(t, err)
	interfaceResult, err := interfaceExpr.Resolve(machine)
	require.NoError(t, err)
	interfaceTypeValue, err := interfaceResult.Static().StringValue()
	require.NoError(t, err)
	require.Equal(t, string(interfaceType), interfaceTypeValue)
}

func TestCreateExecutionMachine_RunningContextMissing(t *testing.T) {
	cfg := &ExecutionConfig{}

	machine := CreateExecutionMachine(cfg)

	expr, err := expressions.Compile("execution.runningContext.actor.name")
	require.NoError(t, err)

	result, err := expr.Resolve(machine)
	require.NoError(t, err)

	require.Equal(t, "", result.Template())
}

// resolveString compiles and resolves one expression against a machine.
func resolveString(t *testing.T, machine expressions.Machine, expr string) string {
	t.Helper()
	compiled, err := expressions.Compile(expr)
	require.NoError(t, err)
	resolved, err := compiled.Resolve(machine)
	require.NoError(t, err)
	value, err := resolved.Static().StringValue()
	require.NoError(t, err)
	return value
}

func TestCreateExecutionMachine_LineageVariables(t *testing.T) {
	machine := CreateExecutionMachine(&ExecutionConfig{
		Id:      "exec-3",
		Lineage: &LineageConfig{BaseId: "exec-2", RootId: "exec-1", Attempt: 3},
	})

	require.Equal(t, "exec-2", resolveString(t, machine, "execution.lineage.baseId"))
	require.Equal(t, "exec-1", resolveString(t, machine, "execution.lineage.rootId"))
	require.Equal(t, "3", resolveString(t, machine, "execution.lineage.attempt"))
}

// Most executions are not reruns, and the expression has to stay usable in a
// workflow that is run both ways. Resolving to an error, or to an empty string
// that reads as attempt zero, would force every author to guard the accessor.
func TestCreateExecutionMachine_LineageDefaultsToAnOriginalRun(t *testing.T) {
	machine := CreateExecutionMachine(&ExecutionConfig{Id: "exec-1"})

	require.Equal(t, "", resolveString(t, machine, "execution.lineage.baseId"))
	require.Equal(t, "1", resolveString(t, machine, "execution.lineage.attempt"))
	// An original run is its own root, matching EffectiveLineage(). Reporting an
	// empty rootId here would contradict what the API says about the same
	// execution, and break any workflow grouping a chain by it.
	require.Equal(t, "exec-1", resolveString(t, machine, "execution.lineage.rootId"))
}

// A record that carries a base but no root - a partially filled row - still has
// to come back coherent, field by field, the way EffectiveLineage() does.
func TestCreateExecutionMachine_LineagePartialRecordFallsBack(t *testing.T) {
	machine := CreateExecutionMachine(&ExecutionConfig{
		Id:      "exec-2",
		Lineage: &LineageConfig{BaseId: "exec-1"},
	})

	require.Equal(t, "exec-1", resolveString(t, machine, "execution.lineage.baseId"))
	require.Equal(t, "exec-2", resolveString(t, machine, "execution.lineage.rootId"))
	require.Equal(t, "1", resolveString(t, machine, "execution.lineage.attempt"))
}

// The scheduling machine must leave execution.lineage.* as an expression, not
// resolve it to the original run's values and not quietly answer empty. Empty
// would be the worse failure of the two: a workflow branching on
// execution.lineage.baseId would take the original branch on every rerun with
// nothing to show why.
func TestCreateSchedulingExecutionMachine_LeavesLineageUnresolved(t *testing.T) {
	machine := CreateSchedulingExecutionMachine(&ExecutionConfig{
		Id:      "exec-2",
		Lineage: &LineageConfig{BaseId: "exec-1", RootId: "exec-1", Attempt: 2},
	})

	for _, src := range []string{
		"execution.lineage.baseId",
		"execution.lineage.rootId",
		"execution.lineage.attempt",
		`execution.lineage.baseId != ""`,
	} {
		expr, err := expressions.Compile(src)
		require.NoError(t, err)

		resolved, err := expr.Resolve(machine)
		require.NoError(t, err, "%s must not error", src)
		assert.Nil(t, resolved.Static(), "%s must stay dynamic for the pod to resolve, got %q", src, resolved.String())
	}
}

// Everything else about the execution still resolves while scheduling: only
// lineage is deferred, because only lineage differs between a snapshot and a
// rerun replaying it.
func TestCreateSchedulingExecutionMachine_StillResolvesTheRest(t *testing.T) {
	machine := CreateSchedulingExecutionMachine(&ExecutionConfig{Id: "exec-2", Name: "wf-2", Number: 7})

	for src, want := range map[string]string{
		"execution.id":     "exec-2",
		"execution.name":   "wf-2",
		"execution.number": "7",
	} {
		expr, err := expressions.Compile(src)
		require.NoError(t, err)

		resolved, err := expr.Resolve(machine)
		require.NoError(t, err)
		require.NotNil(t, resolved.Static(), "%s should resolve while scheduling", src)

		value, err := resolved.Static().StringValue()
		require.NoError(t, err)
		assert.Equal(t, want, value, src)
	}
}
