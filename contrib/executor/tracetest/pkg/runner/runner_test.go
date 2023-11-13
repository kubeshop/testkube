package runner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/runner"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
)

func TestRun(t *testing.T) {

	t.Run("runner should fail if no env var is provided", func(t *testing.T) {
		// given
		ctx := context.Background()
		params := envs.Params{
			DataDir: "/tmp",
		}

		runner, err := runner.NewRunner(ctx, params)
		require.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("hello I'm test content")

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		require.ErrorContains(t, err, "could not find variables to run the test with Tracetest or Tracetest Cloud")
	})

}
