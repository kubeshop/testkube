package runner_test

import (
	"testing"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/runner"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {

	t.Run("runner should fail if no env var is provided", func(t *testing.T) {
		// given
		runner, err := runner.NewRunner()
		require.NoError(t, err)

		runner.Params.DataDir = "/tmp"

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("hello I'm test content")

		// when
		_, err = runner.Run(*execution)

		// then
		require.Error(t, err)
		require.Equal(t, "could not find variables to run the test with Tracetest or Tracetest Cloud", err.Error())
	})

}
