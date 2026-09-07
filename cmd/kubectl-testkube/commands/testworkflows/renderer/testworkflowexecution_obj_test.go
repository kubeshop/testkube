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
	assert.Contains(t, output, "check your API endpoint configuration")
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

// executionWithTestResults builds an execution whose single step reports the
// given test-case counters.
func executionWithTestResults(results *testkube.TestWorkflowStepTestResults) testkube.TestWorkflowExecution {
	passed := testkube.PASSED_TestWorkflowStepStatus
	return testkube.TestWorkflowExecution{
		Id:        "exec-1",
		Workflow:  &testkube.TestWorkflow{Name: "suite"},
		Signature: []testkube.TestWorkflowSignature{{Ref: "step1", Name: "Run the suite"}},
		Result: &testkube.TestWorkflowResult{
			Steps: map[string]testkube.TestWorkflowStepResult{
				"step1": {Status: &passed, TestResults: results},
			},
		},
	}
}

func renderExecution(t *testing.T, execution testkube.TestWorkflowExecution) string {
	t.Helper()
	var buf bytes.Buffer
	require.NotPanics(t, func() {
		printPrettyOutput(ui.NewUI(false, &buf), execution)
	})
	return buf.String()
}

func TestPrintPrettyOutput_MutedCountsAreVisible(t *testing.T) {
	// A step that passed because its failures were muted looks identical to one
	// with nothing to fail unless the counts are shown. Muted is tolerated, not
	// hidden.
	output := renderExecution(t, executionWithTestResults(&testkube.TestWorkflowStepTestResults{
		Tests: 1043, Passed: 1035, Failed: 8, Muted: 8,
		Tolerated: true, RequirementApplied: true,
	}))

	assert.Contains(t, output, "Test cases:")
	assert.Contains(t, output, "Run the suite")
	assert.Contains(t, output, "1035/1043 passed")
	assert.Contains(t, output, "8 muted")
	assert.Contains(t, output, "within the pass requirement")
}

func TestPrintPrettyOutput_UnexpectedAndMissedRequirement(t *testing.T) {
	output := renderExecution(t, executionWithTestResults(&testkube.TestWorkflowStepTestResults{
		Tests: 100, Passed: 90, Failed: 10, Muted: 6, Unexpected: 4,
		Tolerated: false, RequirementApplied: true,
	}))

	assert.Contains(t, output, "90/100 passed")
	assert.Contains(t, output, "6 muted")
	assert.Contains(t, output, "4 unexpected")
	assert.Contains(t, output, "short of the pass requirement")
}

func TestPrintPrettyOutput_UnusedMutePatternsAreNamed(t *testing.T) {
	// Dead quarantine config, surfaced so mute lists do not outlive their bugs.
	output := renderExecution(t, executionWithTestResults(&testkube.TestWorkflowStepTestResults{
		Tests: 10, Passed: 10,
		UnusedMutePatterns: []string{"test_retired_*", "Tests.Old/**"},
	}))

	assert.Contains(t, output, "unused mute patterns")
	assert.Contains(t, output, "test_retired_*, Tests.Old/**")
}

func TestPrintPrettyOutput_ExplainsWhyMuteDidNotApply(t *testing.T) {
	// Without the note this reads as mute being broken, rather than the report
	// being unusable for muting.
	output := renderExecution(t, executionWithTestResults(&testkube.TestWorkflowStepTestResults{
		Tests: 24, Passed: 20, Failed: 4,
		IdentitiesIncomplete: true, Unrepresented: 23,
	}))

	assert.Contains(t, output, "does not name")
	assert.Contains(t, output, "mute was not applied")
}

func TestPrintPrettyOutput_NoTestCasesSectionWithoutAPolicy(t *testing.T) {
	// Workflows that do not use the feature must not grow an empty section.
	assert.NotContains(t, renderExecution(t, executionWithTestResults(nil)), "Test cases:")

	// Nor should a policy that produced no test cases at all.
	assert.NotContains(t, renderExecution(t, executionWithTestResults(
		&testkube.TestWorkflowStepTestResults{})), "Test cases:")
}
