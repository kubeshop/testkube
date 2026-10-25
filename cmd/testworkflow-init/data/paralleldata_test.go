package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/paralleldata"
)

func TestParallelDataMachineReadsRecordedGroups(t *testing.T) {
	ClearState()
	t.Cleanup(ClearState)

	GetState().SetOutput("step", paralleldata.InstructionName("shard"), paralleldata.Group{
		Count: 2,
		Workers: []paralleldata.Worker{
			{Index: 0, Status: testkube.PASSED_TestWorkflowStatus, Outputs: map[string]string{"duration": "12"}},
			{Index: 1, Status: testkube.PASSED_TestWorkflowStatus, Outputs: map[string]string{"duration": "7"}},
		},
	})
	GetState().SetOutput("step", paralleldata.InstructionName("broken"), "not a group")

	groups := ParallelGroups()
	assert.Len(t, groups, 1)

	compiled, err := expressions.CompileAndResolve(`parallel("shard", 1).outputs.duration`, ParallelDataMachine())
	require.NoError(t, err)
	require.NotNil(t, compiled.Static())
	assert.Equal(t, "7", compiled.Static().Value())
}

func TestParallelDataInConditionAndResult(t *testing.T) {
	ClearState()
	t.Cleanup(ClearState)

	GetState().SetOutput("step", paralleldata.InstructionName("shard"), paralleldata.Group{
		Count: 2,
		Workers: []paralleldata.Worker{
			{Index: 0, Status: testkube.PASSED_TestWorkflowStatus},
			{Index: 1, Status: testkube.FAILED_TestWorkflowStatus},
		},
	})

	passed, err := (&StepData{Condition: `parallel("shard", 0).status == "passed"`}).ResolveCondition()
	require.NoError(t, err)
	assert.True(t, passed)

	passed, err = (&StepData{Condition: `parallel("shard", 1).status == "passed"`}).ResolveCondition()
	require.NoError(t, err)
	assert.False(t, passed)

	status, err := (&StepData{Result: `parallel("shard", 1).status != "failed"`}).ResolveResult()
	require.NoError(t, err)
	assert.Equal(t, constants.StepStatusFailed, status)
}
