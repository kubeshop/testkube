//go:build integration

package runner

import (
	"context"
	"github.com/kubeshop/testkube/pkg/envs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	cp "github.com/otiai10/copy"
)

func TestRun(t *testing.T) {
	t.Parallel()
	// setup
	tempDir, err := os.MkdirTemp("", "*")
	assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	assert.NoError(t, os.Mkdir(repoDir, 0755))
	_ = cp.Copy("../../examples", repoDir)

	ctx := context.Background()

	params := envs.Params{DataDir: tempDir}
	runner, err := NewPlaywrightRunner(ctx, "pnpm", params)
	if err != nil {
		t.Fail()
	}

	result, err := runner.Run(
		ctx,
		testkube.Execution{
			Content: &testkube.TestContent{
				Type_: string(testkube.TestContentTypeGitDir),
				Repository: &testkube.Repository{
					Type_:  "git",
					Uri:    "",
					Branch: "master",
					Path:   "",
				},
			},
		})

	assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
	assert.NoError(t, err)
}
