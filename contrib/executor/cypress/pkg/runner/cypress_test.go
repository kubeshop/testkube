package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	t.Skip("move this test to e2e test suite with valid environment setup")

	ctx := context.Background()

	// setup
	tempDir, _ := os.MkdirTemp("", "*")
	assert.NoError(t, os.Setenv("RUNNER_DATADIR", tempDir))
	repoDir := filepath.Join(tempDir, "repo")
	assert.NoError(t, os.Mkdir(repoDir, 0755))
	_ = cp.Copy("../../examples", repoDir)

	runner, err := NewCypressRunner(ctx, "npm")
	if err != nil {
		t.Fail()
	}

	repoURI := "https://github.com/kubeshop/testkube-executor-cypress.git"
	result, err := runner.Run(
		ctx,
		testkube.Execution{
			Content: &testkube.TestContent{
				Type_: string(testkube.TestContentTypeGitDir),
				Repository: &testkube.Repository{
					Type_:  "git",
					Uri:    repoURI,
					Branch: "jacek/feature/json-output",
					Path:   "",
				},
			},
		})

	fmt.Printf("RESULT: %+v\n", result)
	fmt.Printf("ERROR:  %+v\n", err)

	t.Fail()

}

func TestRunErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("no RUNNER_DATADIR", func(t *testing.T) {
		assert.NoError(t, os.Setenv("RUNNER_DATADIR", "/unknown"))

		// given
		runner, err := NewCypressRunner(ctx, "yarn")
		if err != nil {
			t.Fail()
		}

		execution := testkube.NewQueuedExecution()

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		assert.Error(t, err)
	})

}
