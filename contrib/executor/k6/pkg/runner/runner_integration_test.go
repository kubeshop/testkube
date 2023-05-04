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

func TestRunFiles_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("Run k6 with simple script", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}
		writeTestContent(t, tempDir, "../../examples/k6-test-script.js")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Run k6 with simple failing script", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}
		writeTestContent(t, tempDir, "../../examples/k6-test-failing-script.js")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Run k6 with arguments and simple script", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/run"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
			"--vus",
			"2",
			"--duration",
			"1s",
		}
		writeTestContent(t, tempDir, "../../examples/k6-test-script.js")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Run k6 with ENV variables and script", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}
		execution.Envs = map[string]string{"TARGET_HOSTNAME": "kubeshop.github.io"}
		writeTestContent(t, tempDir, "../../examples/k6-test-environment.js")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunAdvanced_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("Run k6 with scenarios", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/run"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}
		writeTestContent(t, tempDir, "../../examples/k6-test-scenarios.js")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 2)
	})

	t.Run("Run k6 with checks and thresholds", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}
		writeTestContent(t, tempDir, "../../examples/k6-test-thresholds.js")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, "some thresholds have failed", result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunDirs_Integtaion(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()
	// setup
	tempDir, err := os.MkdirTemp("", "*")
	assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	assert.NoError(t, os.Mkdir(repoDir, 0755))

	k6Script, err := os.ReadFile("../../examples/k6-test-script.js")
	if err != nil {
		assert.FailNow(t, "Unable to read k6 test script")
	}

	err = os.WriteFile(filepath.Join(repoDir, "k6-test-script.js"), k6Script, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write k6 runner test content file")
	}

	t.Run("Run k6 from directory with script argument", func(t *testing.T) {
		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/kubeshop/testkube-executor-k6.git",
				Branch: "main",
			},
		}
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
			"--duration",
			"1s",
			"k6-test-script.js",
		}

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunErrors_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	t.Run("Run k6 with no script", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "255")
	})

	t.Run("Run k6 with invalid arguments", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
			"--vues",
			"2",
			"--duration",
			"5",
		}

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "255")
	})

	t.Run("Run k6 from directory with missing script arg", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(context.Background(), params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/kubeshop/testkube-executor-k6.git",
				Branch: "main",
				Path:   "examples",
			},
		}
		execution.TestType = "k6/script"
		execution.Command = []string{
			"k6",
		}
		execution.Args = []string{
			"<k6Command>",
			"<envVars>",
			"<runPath>",
		}

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "not found")
	})
}

func writeTestContent(t *testing.T, dir string, testScript string) {
	k6Script, err := os.ReadFile(testScript)
	if err != nil {
		assert.FailNow(t, "Unable to read k6 test script")
	}

	err = os.WriteFile(filepath.Join(dir, "test-content"), k6Script, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write k6 runner test content file")
	}
}
