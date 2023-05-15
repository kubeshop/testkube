package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestRun(t *testing.T) {
	test.IntegrationTest(t)
	t.Skipf("Skipping integration test %s until it is installed in CI", t.Name())

	ctx := context.Background()

	t.Run("runner should run test based on execution data", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "test-content"), []byte("hello I'm test content"), 0644)
		if err != nil {
			assert.FailNow(t, "Unable to write template runner test content file")
		}

		// given
		runner := NewRunner(envs.Params{DataDir: tempDir})
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
	})

}
