//go:build integration

package runner

import (
	"os"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Run("run successful jmeter test", func(t *testing.T) {
		testData, err := os.ReadFile("../../examples/kubeshop.jmx")
		assert.NoError(t, err)

		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  string(testData),
		}

		execution.Variables = map[string]testkube.Variable{}

		result, err := runner.Run(*execution)

		assert.NoError(t, err)
		assert.Empty(t, result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup()
		assert.NoError(t, err)
	})
	t.Run("run failing jmeter test", func(t *testing.T) {
		testData, err := os.ReadFile("../../examples/kubeshop_failed.jmx")
		assert.NoError(t, err)

		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  string(testData),
		}

		execution.Variables = map[string]testkube.Variable{}

		result, err := runner.Run(*execution)

		assert.NoError(t, err)
		assert.Equal(t, "Test failed: text expected to contain /SOME_NONExisting_String/", result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup()
		assert.NoError(t, err)
	})
	t.Run("run successful jmeter test with variables", func(t *testing.T) {
		testData, err := os.ReadFile("../../examples/kubeshop.jmx")
		assert.NoError(t, err)

		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  string(testData),
		}

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

		err = cleanup()
		assert.NoError(t, err)
	})
	t.Run("run successful jmeter test with arguments", func(t *testing.T) {
		testData, err := os.ReadFile("../../examples/kubeshop.jmx")
		assert.NoError(t, err)

		runner, err := NewRunner()
		assert.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.TestType = "jmeter/test"
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeString),
			Data:  string(testData),
		}

		execution.Args = []string{"-Jthreads", "10", "-Jrampup", "0", "-Jloopcount", "1", "-Jip", "sampleip", "-Jport", "1234"}

		result, err := runner.Run(*execution)

		assert.NoError(t, err)
		assert.Empty(t, result.ErrorMessage)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)

		err = cleanup()
		assert.NoError(t, err)
	})

}

func cleanup() error {
	err := os.Remove("jmeter.log")
	if err != nil {
		return err
	}
	return os.Remove("report.jtl")
}
