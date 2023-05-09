package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestRun(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("runner should run test based on execution data", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples", repoDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewArtilleryRunner(ctx, params)
		assert.NoError(t, err)

		repoURI := "https://github.com/kubeshop/testkube-executor-artillery.git"
		result, err := runner.Run(
			ctx,
			testkube.Execution{
				Content: &testkube.TestContent{
					Type_: string(testkube.TestContentTypeGitFile),
					Repository: &testkube.Repository{
						Type_:  "git",
						Uri:    repoURI,
						Branch: "main",
						Path:   "examples/test.yaml",
					},
				},
			})
		if err != nil {
			t.Errorf("Artillery Test Failed: ResultErr: %v, Err: %v ", result.ErrorMessage, err)
		}
		// then
		assert.NoError(t, err)
		assert.Equal(t, *result.Status, testkube.PASSED_ExecutionStatus)
	})
}
