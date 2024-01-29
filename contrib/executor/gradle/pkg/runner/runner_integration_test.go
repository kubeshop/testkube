// TODO create integration environment with `gradle` binary installed on OS level
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

func TestRunGradle_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("run gradle wrapper project with explicit target arg", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-gradlew", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Command = []string{"gradle"}
		execution.Args = []string{"test", "--no-daemon", "<taskName>", "-p", "<projectDir>"}

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
	})

	t.Run("run gradle project test with envs", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)
		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))
		_ = cp.Copy("../../examples/hello-gradle", repoDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Command = []string{"gradle"}
		execution.Args = []string{"--no-daemon", "<taskName>", "-p", "<projectDir>"}
		assert.NoError(t, os.Setenv("TESTKUBE_GRADLE", "true"))

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunErrors_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("no RUNNER_DATADIR", func(t *testing.T) {
		t.Parallel()
		// given
		params := envs.Params{DataDir: "/unknown"}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"gradle"}
		execution.Args = []string{"--no-daemon", "<taskName>", "-p", "<projectDir>"}

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		assert.Error(t, err)
	})

	t.Run("unsupported file-content", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = testkube.NewStringTestContent("")
		execution.Command = []string{"gradle"}
		execution.Args = []string{"--no-daemon", "<taskName>", "-p", "<projectDir>"}

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		assert.EqualError(t, err, "gradle executor handles only repository based tests, but repository is nil")
	})

	t.Run("no settings.gradle", func(t *testing.T) {
		t.Parallel()
		// setup
		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		repoDir := filepath.Join(tempDir, "repo")
		assert.NoError(t, os.Mkdir(repoDir, 0755))

		// given
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		execution.Command = []string{"gradle"}
		execution.Args = []string{"--no-daemon", "<taskName>", "-p", "<projectDir>"}
		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Contains(t, result.ErrorMessage, "no")
	})
}
