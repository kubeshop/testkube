package runner

import (
	"context"
	"os"
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

	t.Run("runner with pre and post run scripts should run test", func(t *testing.T) {
		t.Parallel()

		params := envs.Params{DataDir: "./testdir"}
		runner := NewRunner(params)
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("hello I'm  test content")
		execution.PreRunScript = "echo \"===== pre-run script\""
		execution.Command = []string{"command.sh"}
		execution.PostRunScript = "echo \"===== pre-run script\""

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusRunning)

		expected := `#!/bin/sh
"testdir/prerun.sh"
prerun_exit_code=$?
if [ $prerun_exit_code -ne 0 ]; then
  exit $prerun_exit_code
fi
"testdir/command.sh" $@
command_exit_code=$?
"testdir/postrun.sh"
postrun_exit_code=$?
if [ $command_exit_code -ne 0 ]; then
  exit $command_exit_code
fi
exit $postrun_exit_code
`

		data, err := os.ReadFile("testdir/entrypoint.sh")
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		assert.Equal(t, string(data), expected)
	})

}
