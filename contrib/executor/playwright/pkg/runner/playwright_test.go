package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	cp "github.com/otiai10/copy"
)

func TestRun(t *testing.T) {
	// setup
	tempDir, _ := os.MkdirTemp("", "*")
	assert.NoError(t, os.Setenv("RUNNER_DATADIR", tempDir))
	repoDir := filepath.Join(tempDir, "repo")
	assert.NoError(t, os.Mkdir(repoDir, 0755))
	_ = cp.Copy("../../examples", repoDir)

	ctx := context.Background()

	runner, err := NewPlaywrightRunner(ctx, "pnpm")
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
