// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func ptr(v int32) *int32 { return &v }

// processStep validates a policy on a step that can actually carry one - the
// shell is what the policy attaches to, and a step without it is refused. See
// TestProcessTestCases_RejectsAPolicyThatCannotTakeEffect.
func processStep(policy *testworkflowsv1.StepTestCases) error {
	_, err := ProcessTestCases(nil, nil, nil, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{Shell: "pytest"},
		TestCases:      policy,
	})
	return err
}

func report() *testworkflowsv1.TestCaseReport {
	return &testworkflowsv1.TestCaseReport{Paths: []string{"junit.xml"}}
}

func TestProcessTestCases_NoPolicyIsFine(t *testing.T) {
	stage, err := ProcessTestCases(nil, nil, nil, testworkflowsv1.Step{})
	require.NoError(t, err)
	assert.Nil(t, stage, "the operation contributes no stage; the policy rides on the run container")
}

func TestProcessTestCases_AcceptsAValidPolicy(t *testing.T) {
	require.NoError(t, processStep(&testworkflowsv1.StepTestCases{
		Report:   &testworkflowsv1.TestCaseReport{Format: "junit", Paths: []string{"reports/**/*.xml"}, OnMissing: "fail"},
		Mute:     &testworkflowsv1.TestCaseSelector{Include: []string{"test_flaky_*"}, Exclude: []string{"test_flaky_critical"}},
		Tolerate: &testworkflowsv1.TestCaseTolerance{MinPassed: ptr(9000)},
		Enforce:  "always",
	}))
}

func TestProcessTestCases_RequiresAReport(t *testing.T) {
	// Everything the policy expresses is a statement about a report, so a policy
	// without one could never act.
	err := processStep(&testworkflowsv1.StepTestCases{
		Mute: &testworkflowsv1.TestCaseSelector{Include: []string{"a"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report.paths is required")

	err = processStep(&testworkflowsv1.StepTestCases{Report: &testworkflowsv1.TestCaseReport{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report.paths is required")
}

func TestProcessTestCases_RejectsUnknownEnums(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy *testworkflowsv1.StepTestCases
		want   string
	}{
		{
			"report format",
			&testworkflowsv1.StepTestCases{Report: &testworkflowsv1.TestCaseReport{Format: "trx", Paths: []string{"a.xml"}}},
			"report.format",
		},
		{
			"onMissing",
			&testworkflowsv1.StepTestCases{Report: &testworkflowsv1.TestCaseReport{Paths: []string{"a.xml"}, OnMissing: "shrug"}},
			"report.onMissing",
		},
		{
			"enforce",
			&testworkflowsv1.StepTestCases{Report: report(), Enforce: "sometimes"},
			"enforce",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := processStep(tc.policy)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestProcessTestCases_RejectsUnusableMuteGlobs(t *testing.T) {
	// Refusing at processing time is the point: a pattern that cannot compile
	// would otherwise fail inside the pod, after a runner had been consumed.
	err := processStep(&testworkflowsv1.StepTestCases{
		Report: report(),
		Mute:   &testworkflowsv1.TestCaseSelector{Include: []string{"["}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mute")
	assert.Contains(t, err.Error(), "invalid pattern")
}

func TestProcessTestCases_RejectsIncoherentTolerance(t *testing.T) {
	t.Run("empty states no requirement", func(t *testing.T) {
		err := processStep(&testworkflowsv1.StepTestCases{
			Report:   report(),
			Tolerate: &testworkflowsv1.TestCaseTolerance{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tolerate is empty")
	})

	t.Run("percentage out of range", func(t *testing.T) {
		err := processStep(&testworkflowsv1.StepTestCases{
			Report:   report(),
			Tolerate: &testworkflowsv1.TestCaseTolerance{MaxFailedPercent: ptr(150)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "between 0 and 100")
	})

	t.Run("negative count", func(t *testing.T) {
		err := processStep(&testworkflowsv1.StepTestCases{
			Report:   report(),
			Tolerate: &testworkflowsv1.TestCaseTolerance{MinPassed: ptr(-1)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

func TestProcessTestCases_AcceptsASelection(t *testing.T) {
	require.NoError(t, processStep(&testworkflowsv1.StepTestCases{
		Report: report(),
		Mute:   &testworkflowsv1.TestCaseSelector{Include: []string{"test_flaky_*"}},
		Select: &testworkflowsv1.TestCaseSelection{
			From:     "self",
			Status:   []string{"failed", "errored"},
			Include:  []string{"tests.payments/**"},
			As:       `{{ testcase.classname }}::{{ testcase.name }}`,
			Collapse: `-Dtest={{ join(selected, ",") }}`,
			Write:    &testworkflowsv1.TestCaseSelectionWrite{Path: "run/selected.txt"},
			Empty:    "skip",
		},
	}))
}

func TestProcessTestCases_RejectsUnavailableSelectionSources(t *testing.T) {
	t.Run("step reference points at the simpler way", func(t *testing.T) {
		// A step reference would need the whole workflow to resolve, and is
		// unnecessary: the steps share a file system, so the error names the
		// field that does the same job rather than only refusing.
		err := processStep(&testworkflowsv1.StepTestCases{
			Report: report(),
			Select: &testworkflowsv1.TestCaseSelection{From: "step:first_pass"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "select.paths")
	})

	// Another execution's report is one of its artifacts, and nothing here can
	// guess which. Refusing beats resolving the reference at runtime only to
	// select nothing out of it.
	t.Run("another execution without paths to its report", func(t *testing.T) {
		for _, from := range []string{"rerun", "parent", "some-workflow", "exec-1"} {
			t.Run(from, func(t *testing.T) {
				err := processStep(&testworkflowsv1.StepTestCases{
					Report: report(),
					Select: &testworkflowsv1.TestCaseSelection{From: from},
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "select.paths has to say")
			})
		}
	})

	// Whether an execution exists is not knowable at processing time, so any
	// reference that names paths is accepted here and resolved in the pod.
	t.Run("another execution with paths is accepted", func(t *testing.T) {
		for _, from := range []string{"rerun", "parent", "some-workflow"} {
			t.Run(from, func(t *testing.T) {
				require.NoError(t, processStep(&testworkflowsv1.StepTestCases{
					Report: report(),
					Select: &testworkflowsv1.TestCaseSelection{
						From:  from,
						Paths: []string{"reports/**/*.xml"},
					},
				}))
			})
		}
	})
}

func TestProcessTestCases_RejectsBadSelectionFields(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selection *testworkflowsv1.TestCaseSelection
		want      string
	}{
		{
			"selecting passing tests",
			&testworkflowsv1.TestCaseSelection{Status: []string{"passed"}},
			"tells us nothing",
		},
		{
			"unknown status",
			&testworkflowsv1.TestCaseSelection{Status: []string{"exploded"}},
			"not a test case outcome",
		},
		{
			"unknown empty policy",
			&testworkflowsv1.TestCaseSelection{Empty: "shrug"},
			"select.empty",
		},
		{
			"unusable filter glob",
			&testworkflowsv1.TestCaseSelection{Include: []string{"["}},
			"invalid pattern",
		},
		{
			"write without a path",
			&testworkflowsv1.TestCaseSelection{Write: &testworkflowsv1.TestCaseSelectionWrite{}},
			"select.write.path is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := processStep(&testworkflowsv1.StepTestCases{Report: report(), Select: tc.selection})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

// A policy the step cannot act on used to validate cleanly and then do nothing:
// the step ran, the report was never read, and a mute policy the author wrote
// was silently absent. That is the failure mode the feature exists to avoid.
func TestProcessTestCases_RejectsAPolicyThatCannotTakeEffect(t *testing.T) {
	policy := &testworkflowsv1.StepTestCases{Report: report()}

	for _, tc := range []struct {
		name string
		step testworkflowsv1.Step
		says string
	}{
		{
			name: "a parallel block",
			step: testworkflowsv1.Step{
				Parallel:  &testworkflowsv1.StepParallel{},
				TestCases: policy,
			},
			says: "move it onto the step inside `parallel`",
		},
		{
			name: "a step that only groups others",
			step: testworkflowsv1.Step{
				Steps:     []testworkflowsv1.Step{{}},
				TestCases: policy,
			},
			says: "move the policy onto the nested step",
		},
		{
			name: "a setup group",
			step: testworkflowsv1.Step{
				Setup:     []testworkflowsv1.Step{{}},
				TestCases: policy,
			},
			says: "move the policy onto the nested step",
		},
		{
			name: "artifacts with no command",
			step: testworkflowsv1.Step{
				StepOperations: testworkflowsv1.StepOperations{
					Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"reports/**"}},
				},
				TestCases: policy,
			},
			says: "needs `run` or `shell`",
		},
		{
			name: "nothing at all",
			step: testworkflowsv1.Step{TestCases: policy},
			says: "needs `run` or `shell`",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ProcessTestCases(nil, nil, nil, tc.step)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.says, "the message has to name where the policy belongs")
		})
	}
}

// The two placements that do work. A step with both a command and artifacts is
// the ordinary shape - the policy lands on the container that ran the tool, not
// on the one that uploads its output.
func TestProcessTestCases_AcceptsAPolicyOnACommand(t *testing.T) {
	policy := &testworkflowsv1.StepTestCases{Report: report()}

	t.Run("shell", func(t *testing.T) {
		_, err := ProcessTestCases(nil, nil, nil, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{Shell: "pytest"},
			TestCases:      policy,
		})
		require.NoError(t, err)
	})

	t.Run("run alongside artifacts", func(t *testing.T) {
		_, err := ProcessTestCases(nil, nil, nil, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{
				Run:       &testworkflowsv1.StepRun{},
				Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"reports/**"}},
			},
			TestCases: policy,
		})
		require.NoError(t, err)
	})
}
