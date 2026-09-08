package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// The scheduler records the rerun policy on the execution rather than handing it
// to the runner, so this path has to read it back off the record. Losing it here
// would store the caller's selection faithfully and never act on it: the suite
// would run in full, with nothing to say the selection had been dropped.
func TestRerunConfigFromExecution(t *testing.T) {
	config := rerunConfigFromExecution(&testkube.TestWorkflowRerun{
		ExecutionId: "exec-1",
		OnlyFailed:  true,
		TestCases:   []string{"s/c/one", "s/c/two"},
	})

	require.NotNil(t, config)
	assert.Equal(t, "exec-1", config.ExecutionId)
	assert.True(t, config.OnlyFailed)
	assert.Equal(t, []string{"s/c/one", "s/c/two"}, config.TestCases)
}

// Almost every execution is not a rerun, and nil has to stay nil: the pod reads
// a non-nil policy as "this run was narrowed".
func TestRerunConfigFromExecutionWithoutAPolicy(t *testing.T) {
	assert.Nil(t, rerunConfigFromExecution(nil))
}

// The record's slice is not ours to hand out - the execution object outlives
// this call and is written back to the control plane.
func TestRerunConfigFromExecutionClonesTheTestCases(t *testing.T) {
	rerun := &testkube.TestWorkflowRerun{TestCases: []string{"s/c/one"}}

	config := rerunConfigFromExecution(rerun)
	require.NotNil(t, config)
	config.TestCases[0] = "mutated"

	assert.Equal(t, []string{"s/c/one"}, rerun.TestCases, "the execution record must not be written through")
}
