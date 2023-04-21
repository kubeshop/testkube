package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
)

func TestRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should run test based on execution data", func(t *testing.T) {
		t.Parallel()

		params := envs.Params{DataDir: "./testdir"}
		runner := NewRunner(params)
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("hello I'm  test content")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusRunning)
	})

}
