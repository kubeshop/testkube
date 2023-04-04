// TODO create integration environment with `gradle` binary installed on OS level
package runner

import (
	"context"
	"fmt"
	"github.com/kubeshop/testkube/pkg/utils/test"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/testkube/pkg/envs"

	cp "github.com/otiai10/copy"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
		runner := NewRunner(params)
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
		runner := NewRunner(params)
		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/lreimer/hands-on-testkube.git",
				Branch: "main",
			},
		}
		assert.NoError(t, os.Setenv("TESTKUBE_GRADLE", "true"))

		// when
		result, err := runner.Run(ctx, *execution)

		fmt.Printf("%+v\n", result)

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
		runner := NewRunner(params)
		execution := testkube.NewQueuedExecution()

		// when
		_, err := runner.Run(ctx, *execution)

		// then
		assert.Error(t, err)
	})

	t.Run("unsupported file-content", func(t *testing.T) {
		t.Parallel()

		tempDir := os.TempDir()

		// given
		params := envs.Params{DataDir: tempDir}
		runner := NewRunner(params)
		execution := testkube.NewQueuedExecution()
		execution.TestType = "gradle/project"
		execution.Content = testkube.NewStringTestContent("")

		// when
		_, err := runner.Run(ctx, *execution)

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
		runner := NewRunner(params)
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
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Contains(t, result.ErrorMessage, "no")
	})
}
