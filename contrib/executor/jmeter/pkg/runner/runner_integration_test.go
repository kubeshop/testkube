package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/envs"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestRun_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	ctx := context.Background()

	assert.NoError(t, os.Setenv("ENTRYPOINT_CMD", "jmeter"))

	t.Run("run successful jmeter test", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(ctx, params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		execution.Command = []string{
			"<entryPoint>",
		}
		execution.Args = []string{
			"-n",
			"-j",
			"<logFile>",
			"-t",
			"<runPath>",
			"-l",
			"<jtlFile>",
			"-e",
			"-o",
			"<reportFile>",
			"<envVars>",
		}
		writeTestContent(t, tempDir, "../../examples/kubeshop.jmx")

		execution.Variables = map[string]testkube.Variable{}

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Empty(t, result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup(tempDir)
		assert.NoError(t, err)
	})

	t.Run("run failing jmeter test", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(ctx, params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		execution.Command = []string{
			"<entryPoint>",
		}
		execution.Args = []string{
			"-n",
			"-j",
			"<logFile>",
			"-t",
			"<runPath>",
			"-l",
			"<jtlFile>",
			"-e",
			"-o",
			"<reportFile>",
			"<envVars>",
		}
		writeTestContent(t, tempDir, "../../examples/kubeshop_failed.jmx")

		execution.Variables = map[string]testkube.Variable{}

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Equal(t, "Test failed: text expected to contain /SOME_NONExisting_String/", result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup(tempDir)
		assert.NoError(t, err)
	})

	t.Run("run successful jmeter test with arguments", func(t *testing.T) {
		t.Parallel()

		tempDir, err := os.MkdirTemp("", "*")
		assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := NewRunner(ctx, params)
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		execution.Command = []string{
			"<entryPoint>",
		}
		execution.Args = []string{
			"-n",
			"-j",
			"<logFile>",
			"-t",
			"<runPath>",
			"-l",
			"<jtlFile>",
			"-e",
			"-o",
			"<reportFile>",
			"<envVars>",
			"-Jthreads",
			"10",
			"-Jrampup",
			"0",
			"-Jloopcount",
			"1",
			"-Jip",
			"sampleip",
			"-Jport",
			"1234",
		}
		writeTestContent(t, tempDir, "../../examples/kubeshop.jmx")

		result, err := runner.Run(ctx, *execution)

		assert.NoError(t, err)
		assert.Empty(t, result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup(tempDir)
		assert.NoError(t, err)
	})

}

func cleanup(tempDir string) error {
	return os.RemoveAll(filepath.Join(tempDir, "output"))
}

func writeTestContent(t *testing.T, dir string, testScript string) {
	jmeterScript, err := os.ReadFile(testScript)
	if err != nil {
		assert.FailNow(t, "Unable to read jmeter test script")
	}

	err = os.WriteFile(filepath.Join(dir, "test-content"), jmeterScript, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write jmeter runner test content file")
	}
}
