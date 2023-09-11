package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
)

func TestRun(t *testing.T) {
	// setup
	os.Setenv("ZAP_HOME", "../../zap/")

	t.Run("Run successful API scan", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "simple-api-scan"
		execution.TestType = "zap/api"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-api-pass.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 2)
		assert.Equal(t, result.Steps[0].Name, "Vulnerable JS Library [10003]")
		assert.Equal(t, result.Steps[0].Status, string(testkube.PASSED_ExecutionStatus))

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})

	t.Run("Run API scan with PASS and WARN", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "warn-api-scan"
		execution.TestType = "zap/api"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-api-warn.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 2)
		assert.Equal(t, result.Steps[1].Name, "Re-examine Cache-control Directives [10015] x 12")
		assert.Equal(t, result.Steps[1].Status, string(testkube.PASSED_ExecutionStatus))

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})

	t.Run("Run API scan with WARN and FailOnWarn", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "fail-on-warn-api-scan"
		execution.TestType = "zap/api"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-api-fail-on-warn.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.Error(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Len(t, result.Steps, 2)
		assert.Equal(t, result.Steps[1].Name, "Re-examine Cache-control Directives [10015] x 12")
		assert.Equal(t, result.Steps[1].Status, string(testkube.FAILED_ExecutionStatus))

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})

	t.Run("Run API scan with FAIL", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "fail-api-scan"
		execution.TestType = "zap/api"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-api-fail.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.Error(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Len(t, result.Steps, 1)
		assert.Equal(t, result.Steps[0].Name, "Unknown issue")
		assert.Equal(t, result.Steps[0].Status, string(testkube.FAILED_ExecutionStatus))

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})

	t.Run("Run Baseline scan with PASS", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "baseline-scan"
		execution.TestType = "zap/baseline"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-baseline-pass.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 2)

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})

	t.Run("Run Baseline scan with WARN", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "baseline-warn-scan"
		execution.TestType = "zap/baseline"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-baseline-warn.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusPassed)
		assert.Len(t, result.Steps, 2)
		assert.Equal(t, result.Steps[1].Status, string(testkube.PASSED_ExecutionStatus))

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})

	t.Run("Run Full scan with FAIL", func(t *testing.T) {
		// given
		tempDir, err := os.MkdirTemp(os.TempDir(), "")
		assert.NoError(t, err)
		runner, err := NewRunner(context.TODO(), envs.Params{
			DataDir: tempDir,
		})
		assert.NoError(t, err)
		execution := testkube.NewQueuedExecution()
		execution.Command = []string{"<pythonScriptPath>"}
		execution.Args = []string{"<fileArgs>"}
		execution.TestName = "full-fail-scan"
		execution.TestType = "zap/full"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/test-full-fail.yaml")

		// when
		result, err := runner.Run(context.TODO(), *execution)

		// then
		assert.Error(t, err)
		assert.Equal(t, result.Status, testkube.ExecutionStatusFailed)
		assert.Len(t, result.Steps, 2)
		assert.Equal(t, result.Steps[0].Status, string(testkube.FAILED_ExecutionStatus))
		assert.Equal(t, result.Steps[1].Status, string(testkube.FAILED_ExecutionStatus))

		// clean-up
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err)
	})
}

func writeTestContent(t *testing.T, dir string, configFile string) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		assert.FailNow(t, "Unable to read ZAP config file")
	}

	err = os.WriteFile(filepath.Join(dir, "test-content"), data, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write ZAP test-content file")
	}
}
