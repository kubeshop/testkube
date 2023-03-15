//go:build integration

package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tempDir := os.TempDir()
	os.Setenv("RUNNER_DATADIR", tempDir)

	t.Run("run successful jmeter test", func(t *testing.T) {
		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/kubeshop.jmx")

		execution.Variables = map[string]testkube.Variable{}

		result, err := runner.Run(*execution)

		assert.NoError(t, err)
		assert.Empty(t, result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup(tempDir)
		assert.NoError(t, err)
	})

	t.Run("run failing jmeter test", func(t *testing.T) {
		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/kubeshop_failed.jmx")

		execution.Variables = map[string]testkube.Variable{}

		result, err := runner.Run(*execution)

		assert.NoError(t, err)
		assert.Equal(t, "Test failed: text expected to contain /SOME_NONExisting_String/", result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup(tempDir)
		assert.NoError(t, err)
	})

	t.Run("run successful jmeter test with variables", func(t *testing.T) {
		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/kubeshop.jmx")

		execution.Variables = map[string]testkube.Variable{
			"threads":   {Name: "threads", Value: "10", Type_: testkube.VariableTypeBasic},
			"rampup":    {Name: "rampup", Value: "0", Type_: testkube.VariableTypeBasic},
			"loopcount": {Name: "loopcount", Value: "1", Type_: testkube.VariableTypeBasic},
			"ip":        {Name: "ip", Value: "sampleip", Type_: testkube.VariableTypeBasic},
			"port":      {Name: "port", Value: "1234", Type_: testkube.VariableTypeBasic},
		}

		result, err := runner.Run(*execution)

		assert.NoError(t, err)
		assert.Empty(t, result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup(tempDir)
		assert.NoError(t, err)
	})

	t.Run("run successful jmeter test with arguments", func(t *testing.T) {
		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = testkube.NewStringTestContent("")
		writeTestContent(t, tempDir, "../../examples/kubeshop.jmx")

		execution.Args = []string{"-Jthreads", "10", "-Jrampup", "0", "-Jloopcount", "1", "-Jip", "sampleip", "-Jport", "1234"}

		result, err := runner.Run(*execution)

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
