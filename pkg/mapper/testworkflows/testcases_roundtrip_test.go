package testworkflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TestStepTestCasesRoundTrip is the regression guard for a bug this test would
// have caught and did not exist to.
//
// Every scheduling path takes a workflow through the API model - the resolved
// spec is stored and shipped as JSON of testkube.TestWorkflow, and the enqueuer
// maps it back with MapAPIToKube before the processor ever sees it. A field
// present on the CRD type but absent from the API model is therefore silently
// dropped somewhere in the middle, and unit tests that build a
// testworkflowsv1.Step directly never cross that boundary.
//
// Assert on the whole struct rather than field by field: a field added to the
// CRD and forgotten in the mappers has to fail here, and a per-field assertion
// list would simply not mention it.
func TestStepTestCasesRoundTrip(t *testing.T) {
	policy := &testworkflowsv1.StepTestCases{
		Report: &testworkflowsv1.TestCaseReport{
			Format:    "junit",
			Paths:     []string{"reports/**/*.xml"},
			OnMissing: "fail",
		},
		Mute: &testworkflowsv1.TestCaseSelector{
			Include: []string{"test_flaky_*", "Tests.Integration/**"},
			Exclude: []string{"test_flaky_checkout_total"},
		},
		Tolerate: &testworkflowsv1.TestCaseTolerance{
			MaxFailed:        common.Ptr(int32(0)),
			MaxFailedPercent: common.Ptr(int32(5)),
			MinPassed:        common.Ptr(int32(9000)),
			MinPassedPercent: common.Ptr(int32(90)),
		},
		Enforce: "always",
		Select: &testworkflowsv1.TestCaseSelection{
			From:         "self",
			Paths:        []string{"reports/first.xml"},
			Status:       []string{"failed", "errored"},
			Include:      []string{"tests.payments/**"},
			Exclude:      []string{"**/t_slow"},
			Cases:        []string{"tests.a::test_one"},
			IncludeMuted: true,
			As:           `{{ testcase.classname }}::{{ testcase.name }}`,
			Collapse:     `-Dtest={{ join(selected, ",") }}`,
			Write: &testworkflowsv1.TestCaseSelectionWrite{
				Path:      "run/selected.txt",
				Separator: ",",
			},
			Empty: "skip",
		},
	}

	t.Run("through a step", func(t *testing.T) {
		step := testworkflowsv1.Step{
			StepMeta:  testworkflowsv1.StepMeta{Name: "Run the suite"},
			TestCases: policy,
		}

		got := MapStepAPIToKube(MapStepKubeToAPI(step))
		require.NotNil(t, got.TestCases, "the policy must survive the API model")
		assert.Equal(t, *policy, *got.TestCases)
	})

	t.Run("through an independent step", func(t *testing.T) {
		// Templates use the independent shape, and a template is the main way
		// this block gets shared, so it has to survive here too.
		step := testworkflowsv1.IndependentStep{
			StepMeta:  testworkflowsv1.StepMeta{Name: "Run the suite"},
			TestCases: policy,
		}

		got := MapIndependentStepAPIToKube(MapIndependentStepKubeToAPI(step))
		require.NotNil(t, got.TestCases)
		assert.Equal(t, *policy, *got.TestCases)
	})
}

func TestStepTestCasesRoundTrip_AbsentStaysAbsent(t *testing.T) {
	// The overwhelmingly common case: a step with no policy must not acquire an
	// empty one, which downstream code would read as "a policy was declared".
	step := testworkflowsv1.Step{StepMeta: testworkflowsv1.StepMeta{Name: "Run"}}

	got := MapStepAPIToKube(MapStepKubeToAPI(step))
	assert.Nil(t, got.TestCases)
}

// TestTestCaseToleranceRoundTrip_ZeroIsNotUnset covers why the thresholds are
// boxed on the wire.
//
// `maxFailed: 0` means "tolerate nothing"; not setting it means "no requirement
// was stated". A plain integer collapses those into the same value, and the
// verdict would then apply a bar nobody asked for.
func TestTestCaseToleranceRoundTrip_ZeroIsNotUnset(t *testing.T) {
	zero := testworkflowsv1.TestCaseTolerance{MaxFailed: common.Ptr(int32(0))}
	got := MapTestCaseToleranceAPIToKube(MapTestCaseToleranceKubeToAPI(zero))
	require.NotNil(t, got.MaxFailed, "a zero threshold must survive as a threshold")
	assert.Equal(t, int32(0), *got.MaxFailed)

	unset := testworkflowsv1.TestCaseTolerance{}
	got = MapTestCaseToleranceAPIToKube(MapTestCaseToleranceKubeToAPI(unset))
	assert.Nil(t, got.MaxFailed, "an unset threshold must stay unset")
	assert.Nil(t, got.MinPassed)
}

// TestExecutionResultTestCaseFieldsAreMapped covers the other two types this
// feature added that cross the CRD/API boundary.
//
// These map in one direction only - the API model is the primary shape and the
// CRD mirror exists for the execution's status - so there is no round trip to
// assert. What can still go wrong is the same thing that went wrong with the
// step: a field added to both types and forgotten in the mapper between them.
func TestExecutionResultTestCaseFieldsAreMapped(t *testing.T) {
	t.Run("step result carries the counters", func(t *testing.T) {
		result := testkube.TestWorkflowStepResult{
			TestResults: &testkube.TestWorkflowStepTestResults{
				Tests: 1043, Passed: 1035, Failed: 8, Muted: 8,
				Tolerated: true, RequirementApplied: true,
				IdentitiesIncomplete: true, Unrepresented: 23,
				UnusedMutePatterns: []string{"test_retired_*"},
			},
		}

		got := MapTestWorkflowStepResultAPIToKube(result)
		require.NotNil(t, got.TestResults)
		assert.Equal(t, int32(1043), got.TestResults.Tests)
		assert.Equal(t, int32(8), got.TestResults.Muted)
		assert.True(t, got.TestResults.Tolerated)
		assert.True(t, got.TestResults.RequirementApplied)
		assert.True(t, got.TestResults.IdentitiesIncomplete)
		assert.Equal(t, int32(23), got.TestResults.Unrepresented)
		assert.Equal(t, []string{"test_retired_*"}, got.TestResults.UnusedMutePatterns)
	})

	t.Run("report carries the failures", func(t *testing.T) {
		report := testkube.TestWorkflowReport{
			Ref:  "step-1",
			Kind: "junit",
			Summary: &testkube.TestWorkflowReportSummary{
				Tests: 10, Passed: 8, Failed: 2, Muted: 1, Unexpected: 1,
			},
			Failures: []testkube.TestWorkflowReportFailure{
				{Id: "s/c/broke", Status: "failed", Muted: true, Message: "flaked"},
			},
			FailuresTruncated: true,
		}

		got := MapTestWorkflowReportAPIToKube(report)
		require.Len(t, got.Failures, 1)
		assert.Equal(t, "s/c/broke", got.Failures[0].Id)
		assert.True(t, got.Failures[0].Muted)
		assert.True(t, got.FailuresTruncated)
		require.NotNil(t, got.Summary)
		assert.Equal(t, int32(1), got.Summary.Muted)
		assert.Equal(t, int32(1), got.Summary.Unexpected)
	})

	t.Run("absent stays absent", func(t *testing.T) {
		assert.Nil(t, MapTestWorkflowStepResultAPIToKube(testkube.TestWorkflowStepResult{}).TestResults)
		assert.Empty(t, MapTestWorkflowReportAPIToKube(testkube.TestWorkflowReport{}).Failures)
	})
}
