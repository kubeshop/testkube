package testworkflowexecutor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
)

func lineageTestWorkflow() *testworkflowsv1.TestWorkflow {
	return &testworkflowsv1.TestWorkflow{}
}

// A fan-out derives one lineage for the whole request and hands it to every
// execution, so SetLineage has to copy. Aliasing would let a later write to one
// execution's lineage change every sibling's.
func TestSetLineageCopies(t *testing.T) {
	shared := &testkube.TestWorkflowExecutionLineage{BaseId: "exec-1", RootId: "exec-1", Attempt: 2}

	first := NewIntermediateExecution().SetWorkflow(lineageTestWorkflow()).SetLineage(shared)
	second := NewIntermediateExecution().SetWorkflow(lineageTestWorkflow()).SetLineage(shared)

	shared.Attempt = 99

	require.NotNil(t, first.Execution().Lineage)
	assert.Equal(t, int32(2), first.Execution().Lineage.Attempt)
	assert.Equal(t, int32(2), second.Execution().Lineage.Attempt)
	assert.NotSame(t, first.Execution().Lineage, second.Execution().Lineage)
}

// Nil has to stay nil so the pod falls back to the rerun policy rather than
// reading a zeroed record.
func TestSetLineageNil(t *testing.T) {
	ie := NewIntermediateExecution().SetWorkflow(lineageTestWorkflow()).SetLineage(nil)
	assert.Nil(t, ie.Execution().Lineage)
}

// The default rerun path replays TestWorkflowExecution.ResolvedWorkflow, which
// is whatever Resolve() left behind on the run that produced it. So anything
// Resolve() substitutes is frozen for every rerun of that snapshot, and lineage
// substituted there would pin a rerun to the original's "no base, attempt 1":
// the step that selects failed cases would stay switched off and
// execution("rerun") would never be reached.
//
// This drives that exact path - resolve as the original, take the snapshot it
// stores, resolve it again as a rerun - and asserts the expressions are still
// there to be resolved in the pod. It is the regression for the default path;
// asking for the latest definition was never the broken case.
func TestResolveLeavesLineageForThePod(t *testing.T) {
	workflow := func() *testworkflowsv1.TestWorkflow {
		return &testworkflowsv1.TestWorkflow{
			Spec: testworkflowsv1.TestWorkflowSpec{
				Steps: []testworkflowsv1.Step{
					{
						StepMeta: testworkflowsv1.StepMeta{Name: "narrow", Condition: `execution.lineage.baseId != ""`},
						StepOperations: testworkflowsv1.StepOperations{
							Shell: `echo 'attempt {{ execution.lineage.attempt }} of {{ execution.lineage.rootId }}, id {{ execution.id }}'`,
						},
					},
				},
			},
		}
	}

	original := NewIntermediateExecution().SetWorkflow(workflow())
	original.execution.Id = "exec-1"
	original.execution.GroupId = "group-1"
	original.execution.Name = "wf-1"
	original.execution.Number = 1
	require.NoError(t, original.Resolve("org", "org", "env", "env", nil, false))

	// What the original run stores, and what a rerun replays verbatim.
	snapshot := original.Execution().ResolvedWorkflow
	require.NotNil(t, snapshot)

	step := snapshot.Spec.Steps[0]
	assert.Equal(t, `execution.lineage.baseId!=""`, step.Condition,
		"the condition must survive scheduling unresolved, or a rerun inherits the original's branch")

	assert.Contains(t, step.Shell, "execution.lineage.attempt",
		"lineage in a script must survive scheduling, to be resolved in the pod")
	assert.Contains(t, step.Shell, "execution.lineage.rootId")

	// Per-execution values that are not lineage still resolve here, which is
	// what makes the snapshot a record of the run that produced it.
	assert.Contains(t, step.Shell, "id exec-1")
	assert.NotContains(t, step.Shell, "execution.id")

	// Resolving that snapshot again as a rerun must not resolve lineage either,
	// because this pass is still scheduling - the pod does it.
	rerun := NewIntermediateExecution().SetWorkflow(testworkflowmappers.MapAPIToKube(snapshot))
	rerun.execution.Id = "exec-2"
	rerun.execution.GroupId = "group-2"
	rerun.execution.Name = "wf-2"
	rerun.execution.Number = 2
	rerun.SetLineage(&testkube.TestWorkflowExecutionLineage{BaseId: "exec-1", RootId: "exec-1", Attempt: 2})
	require.NoError(t, rerun.Resolve("org", "org", "env", "env", nil, false))

	rerunStep := rerun.Execution().ResolvedWorkflow.Spec.Steps[0]
	assert.Equal(t, `execution.lineage.baseId!=""`, rerunStep.Condition)

	assert.Contains(t, rerunStep.Shell, "execution.lineage.attempt")

	// The original's id must not have leaked into the rerun's script - that is
	// the same class of staleness, and the reason lineage cannot be resolved here.
	assert.Contains(t, rerunStep.Shell, "id exec-1",
		"a replayed snapshot keeps the ids of the run it came from; lineage must not join them")
}
