package runner

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("runner should run test based on execution data", func(t *testing.T) {
		assert.NoError(t, os.Setenv("RUNNER_DATADIR", "./testdir"))

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("hello I'm  test content")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusRunning)
	})

}
