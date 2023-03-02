//go:build integration

// TODO create integration environemnt with `gradle` binary installed on OS level
package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestRunGradle(t *testing.T) {

	t.Run("run gradle wrapper project with explicit target arg", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)
		_ = cp.Copy("../../examples/hello-gradlew", repoDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Args = []string{"test"}

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("run gradle project test with envs", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)
		_ = cp.Copy("../../examples/hello-gradle", repoDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Envs = map[string]string{"TESTKUBE_GRADLE": "true"}

		// when
		result, err := runner.Run(*execution)

		fmt.Printf("%+v\n", result)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunErrors(t *testing.T) {

	t.Run("no RUNNER_DATADIR", func(t *testing.T) {
		os.Setenv("RUNNER_DATADIR", "/unknown")

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()

		// when
		_, err := runner.Run(*execution)

		// then
		assert.Error(t, err)
	})

	t.Run("unsupported file-content", func(t *testing.T) {
		tempDir := os.TempDir()
		os.Setenv("RUNNER_DATADIR", tempDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = testkube.NewStringTestContent("")

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
	})

	t.Run("no settings.gradle", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)

		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Contains(t, result.ErrorMessage, "no")
	})
}
