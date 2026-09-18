package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// The legacy path rebuilds the pod's config from the execution record, so it has
// to read lineage back off it. Losing it here costs no error: the pod simply
// cannot resolve execution("rerun"), and a workflow written against its own
// previous run silently loses the reference.
//
// The typed path does the same from its proto - see
// TestExecutionConfigFromStart_CarriesLineage in pkg/runner/grpc.
func TestLineageConfigFromExecution(t *testing.T) {
	config := lineageConfigFromExecution(&testkube.TestWorkflowExecutionLineage{
		BaseId:  "exec-2",
		RootId:  "exec-1",
		Attempt: 3,
	})

	require.NotNil(t, config)
	assert.Equal(t, "exec-2", config.BaseId)
	assert.Equal(t, "exec-1", config.RootId)
	assert.Equal(t, int32(3), config.Attempt)
}

// A record written before lineage existed carries none, and nil has to stay nil
// so the pod falls back to the rerun policy instead of reading a zeroed record.
func TestLineageConfigFromExecutionWithoutLineage(t *testing.T) {
	assert.Nil(t, lineageConfigFromExecution(nil))
}
