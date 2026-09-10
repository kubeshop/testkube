package testworkflowexecutor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
