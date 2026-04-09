package testsuite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func testGetNextExecutionNumber(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	name := "seq-test"

	n1, err := repo.GetNextExecutionNumber(ctx, name)
	require.NoError(t, err)
	assert.True(t, n1 >= 1, "first number should be >= 1, got %d", n1)

	n2, err := repo.GetNextExecutionNumber(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, n1+1, n2, "second number should be first+1")

	n3, err := repo.GetNextExecutionNumber(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, n2+1, n3, "third number should be second+1")
}
