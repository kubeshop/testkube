//go:build integraion

package runner

import (
	"os"
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestRunMavenProject(t *testing.T) {

	t.Run("run maven project with test task and envs", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)
		_ = cp.Copy("../../examples/hello-maven", repoDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Args = []string{"test"}
		execution.Envs = map[string]string{"TESTKUBE_MAVEN": "true"}

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 1)
	})

}
