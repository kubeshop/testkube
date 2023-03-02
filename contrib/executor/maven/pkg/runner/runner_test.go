package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {

	t.Run("run maven wrapper test goal with envs", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)
		_ = cp.Copy("../../examples/hello-mvnw", repoDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "someuri",
				Branch: "main",
			},
		}

		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN_WRAPPER", Value: "true", Type_: testkube.VariableTypeBasic},
		}

		os.Setenv("TESTKUBE_MAVEN_WRAPPER", "true")
		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

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
				Uri:    "someur",
				Branch: "main",
			},
		}
		execution.Args = []string{"test"}
		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN", Value: "true", Type_: testkube.VariableTypeBasic},
		}

		os.Setenv("TESTKUBE_MAVEN", "true")
		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("run maven wrapper test goal with envs", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)
		_ = cp.Copy("../../examples/hello-mvnw", repoDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Envs = map[string]string{"TESTKUBE_MAVEN_WRAPPER": "true"}

		os.Setenv("TESTKUBE_MAVEN_WRAPPER", "true")
		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("run maven project with settings.xml", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)
		_ = cp.Copy("../../examples/hello-maven-settings", repoDir)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "someuri",
				Branch: "main",
			},
		}
		execution.Variables = map[string]testkube.Variable{
			"wrapper": {Name: "TESTKUBE_MAVEN", Value: "true", Type_: testkube.VariableTypeBasic},
		}
		settingsContent, err := os.ReadFile(fmt.Sprintf("%s/settings.xml", repoDir))
		assert.NoError(t, err)
		execution.VariablesFile = string(settingsContent)

		os.Setenv("TESTKUBE_MAVEN", "true")
		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
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
		execution.TestType = "maven/test"
		execution.Content = testkube.NewStringTestContent("")

		// when
		_, err := runner.Run(*execution)

		// then
		assert.Error(t, err)
	})

	t.Run("no pom.xml", func(t *testing.T) {
		// setup
		tempDir, _ := os.MkdirTemp("", "*")
		os.Setenv("RUNNER_DATADIR", tempDir)

		repoDir := filepath.Join(tempDir, "repo")
		os.Mkdir(repoDir, 0755)

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.TestType = "maven/test"
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
