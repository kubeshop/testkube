package renderer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func TestPrintPrettyOutput_NilWorkflow(t *testing.T) {
	var buf bytes.Buffer
	testUI := ui.NewUI(false, &buf)

	execution := testkube.TestWorkflowExecution{
		Id:        "test-id",
		Name:      "test-name",
		Namespace: "test-namespace",
		Workflow:  nil,
	}

	require.NotPanics(t, func() {
		printPrettyOutput(testUI, execution)
	})

	output := buf.String()
	assert.Contains(t, output, "incomplete execution data received from API")
	assert.Contains(t, output, "missing Workflow field")
	assert.Contains(t, output, "Hint:")
	assert.Contains(t, output, "test-id")
	assert.Contains(t, output, "test-name")
}

func TestPrintPrettyOutput_NilWorkflowWithInitError(t *testing.T) {
	var buf bytes.Buffer
	testUI := ui.NewUI(false, &buf)

	initErrorMsg := "connection refused: failed to connect to API server"
	execution := testkube.TestWorkflowExecution{
		Id:        "test-id",
		Name:      "test-name",
		Namespace: "test-namespace",
		Workflow:  nil,
		Result: &testkube.TestWorkflowResult{
			Initialization: &testkube.TestWorkflowStepResult{
				ErrorMessage: initErrorMsg,
			},
		},
	}

	require.NotPanics(t, func() {
		printPrettyOutput(testUI, execution)
	})

	output := buf.String()
	assert.Contains(t, output, "incomplete execution data received from API")
	assert.Contains(t, output, initErrorMsg)
}

func TestPrintPrettyOutput_ValidWorkflow(t *testing.T) {
	var buf bytes.Buffer
	testUI := ui.NewUI(false, &buf)

	execution := testkube.TestWorkflowExecution{
		Id:        "test-id",
		Name:      "test-name",
		Namespace: "test-namespace",
		Workflow: &testkube.TestWorkflow{
			Name: "test-workflow",
		},
	}

	require.NotPanics(t, func() {
		printPrettyOutput(testUI, execution)
	})

	output := buf.String()
	assert.Contains(t, output, "test-workflow")
	assert.Contains(t, output, "Test Workflow Execution:")
	assert.NotContains(t, output, "incomplete execution data")
}
