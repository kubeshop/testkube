package testsuite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/test/fixtures"
)

func testGetTestWorkflowMetrics(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "metrics-test"

	for i := 0; i < 3; i++ {
		exec := fixtures.NewExecution(wfName,
			fixtures.WithNumber(int32(i+1)),
			fixtures.WithResult(fixtures.ResultWithDuration(1000+int32(i)*500)),
		)
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	metrics, err := repo.GetTestWorkflowMetrics(ctx, wfName, 10, 10)
	require.NoError(t, err)

	assert.Equal(t, int32(3), metrics.TotalExecutions)
}
