package testresults

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common/testdata"
)

func ptr(v int32) *int32 { return &v }

// synth builds a report of the requested shape. Names are stable so mute
// patterns in the tests can address them.
func synth(passed, failed, flaky int) Report {
	var report Report
	for i := 0; i < passed; i++ {
		report.Cases = append(report.Cases, TestCase{
			SuitePath: []string{"s"}, Classname: "c", Name: nameOf("test_ok", i), Status: StatusPassed,
		})
	}
	for i := 0; i < failed; i++ {
		report.Cases = append(report.Cases, TestCase{
			SuitePath: []string{"s"}, Classname: "c", Name: nameOf("test_real", i), Status: StatusFailed,
		})
	}
	for i := 0; i < flaky; i++ {
		report.Cases = append(report.Cases, TestCase{
			SuitePath: []string{"s"}, Classname: "c", Name: nameOf("test_flaky", i), Status: StatusFailed,
		})
	}
	return report
}

func nameOf(prefix string, i int) string {
	return prefix + "_i" + strconv.Itoa(i)
}

func TestEvaluate_MutedFailuresPassTheStep(t *testing.T) {
	verdict, err := Evaluate(
		synth(10, 0, 3),
		Policy{Mute: Selector{Include: []string{"test_flaky_*"}}},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)

	assert.True(t, verdict.Success, "every failure was muted, so the step passes")
	assert.Len(t, verdict.Muted, 3)
	assert.Empty(t, verdict.Unexpected)
	assert.True(t, verdict.Tolerated)
	assert.False(t, verdict.ToleranceApplied, "no requirement was set")

	// Muted must never mean silent.
	assert.Contains(t, verdict.Describe(), "(3 muted)")
	assert.Contains(t, verdict.Describe(), "0 unexpected")
}

func TestEvaluate_UnmutedFailureStillFails(t *testing.T) {
	verdict, err := Evaluate(
		synth(10, 1, 3),
		Policy{Mute: Selector{Include: []string{"test_flaky_*"}}},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)

	assert.False(t, verdict.Success)
	assert.Len(t, verdict.Muted, 3)
	require.Len(t, verdict.Unexpected, 1)
	assert.Equal(t, "s/c/test_real_i0", verdict.Unexpected[0].ID())
	assert.Contains(t, verdict.Describe(), "unexpected failures remain")
}

func TestEvaluate_OnlyFailingCasesAreMuted(t *testing.T) {
	// A mute pattern that also matches passing cases must not reclassify them.
	verdict, err := Evaluate(
		synth(4, 0, 2),
		Policy{Mute: Selector{Include: []string{"**/*"}}},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)
	assert.Len(t, verdict.Muted, 2, "only the two failures are muted, not the four passes")
	assert.Equal(t, int32(4), verdict.Summary.Passed)
}

func TestEvaluate_EnforceOnFailureNeverBreaksAGreenRun(t *testing.T) {
	// A zero exit code with an unmet requirement stays green under the default.
	verdict, err := Evaluate(
		synth(10, 5, 0),
		Policy{Tolerate: &Tolerance{MaxFailed: ptr(0)}, Enforce: EnforceOnFailure},
		Input{ExitCode: 0},
	)
	require.NoError(t, err)
	assert.True(t, verdict.Success, "onFailure only ever downgrades a failure")
	assert.False(t, verdict.Tolerated, "the requirement was genuinely missed")
}

func TestEvaluate_EnforceAlwaysCatchesAToolExitingZero(t *testing.T) {
	verdict, err := Evaluate(
		synth(10, 5, 0),
		Policy{Tolerate: &Tolerance{MaxFailed: ptr(0)}, Enforce: EnforceAlways},
		Input{ExitCode: 0},
	)
	require.NoError(t, err)
	assert.False(t, verdict.Success,
		"enforce: always is the only way to fail a tool that reports failures but exits 0")
}

func TestEvaluate_Thresholds(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tolerance Tolerance
		passed    int
		failed    int
		tolerated bool
	}{
		{"minPassed met", Tolerance{MinPassed: ptr(9)}, 9, 1, true},
		{"minPassed missed", Tolerance{MinPassed: ptr(10)}, 9, 1, false},
		{"maxFailed met", Tolerance{MaxFailed: ptr(1)}, 9, 1, true},
		{"maxFailed missed", Tolerance{MaxFailed: ptr(0)}, 9, 1, false},
		{"maxFailedPercent met", Tolerance{MaxFailedPercent: ptr(10)}, 9, 1, true},
		{"maxFailedPercent missed", Tolerance{MaxFailedPercent: ptr(5)}, 9, 1, false},
		{"minPassedPercent met", Tolerance{MinPassedPercent: ptr(90)}, 9, 1, true},
		{"minPassedPercent missed", Tolerance{MinPassedPercent: ptr(95)}, 9, 1, false},
		// Every set field must hold, not just one.
		{"one of two missed", Tolerance{MinPassed: ptr(9), MaxFailed: ptr(0)}, 9, 1, false},
		{"both met", Tolerance{MinPassed: ptr(9), MaxFailed: ptr(1)}, 9, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tolerance := tc.tolerance
			verdict, err := Evaluate(
				synth(tc.passed, tc.failed, 0),
				Policy{Tolerate: &tolerance},
				Input{ExitCode: 1},
			)
			require.NoError(t, err)
			assert.Equal(t, tc.tolerated, verdict.Tolerated)
			assert.True(t, verdict.ToleranceApplied)
			assert.Equal(t, tc.tolerated, verdict.Success)
		})
	}
}

func TestEvaluate_MutedFailuresDoNotCountAgainstAThreshold(t *testing.T) {
	// 9000 pass, 1000 muted failures: the requirement is about what actually
	// went wrong, so the muted ones must not be held against it.
	verdict, err := Evaluate(
		synth(9000, 0, 1000),
		Policy{
			Mute:     Selector{Include: []string{"test_flaky_*"}},
			Tolerate: &Tolerance{MaxFailed: ptr(0)},
		},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)
	assert.True(t, verdict.Tolerated)
	assert.True(t, verdict.Success)
	assert.Len(t, verdict.Muted, 1000)
}

func TestEvaluate_ToleranceIsSkippedForANarrowedRun(t *testing.T) {
	tolerance := Tolerance{MinPassed: ptr(9000)}

	full, err := Evaluate(synth(40, 0, 0), Policy{Tolerate: &tolerance}, Input{Narrowed: false})
	require.NoError(t, err)
	assert.True(t, full.ToleranceApplied)
	assert.False(t, full.Tolerated, "40 passing is short of 9000 over a full run")

	narrowed, err := Evaluate(synth(40, 0, 0), Policy{Tolerate: &tolerance}, Input{Narrowed: true})
	require.NoError(t, err)
	assert.False(t, narrowed.ToleranceApplied,
		"a threshold over a subset is meaningless, so it is not evaluated")
	assert.True(t, narrowed.Tolerated, "the bar becomes: nothing unexpected failed")
	assert.True(t, narrowed.Success)
	assert.Contains(t, narrowed.Describe(), "measured a subset")
}

func TestEvaluate_NarrowedRunStillFailsOnAnUnexpectedFailure(t *testing.T) {
	verdict, err := Evaluate(
		synth(39, 1, 0),
		Policy{Tolerate: &Tolerance{MinPassed: ptr(1)}},
		Input{ExitCode: 1, Narrowed: true},
	)
	require.NoError(t, err)
	assert.False(t, verdict.Success,
		"a narrowed run may not borrow a threshold to excuse a real failure")
}

func TestEvaluate_ThresholdsDoNotAccumulateAcrossNarrowingRetries(t *testing.T) {
	// The documented limitation, pinned deliberately so it is not later
	// "fixed" as a regression. minPassed 9000 over 10,000 cases:
	mute := Policy{Tolerate: &Tolerance{MinPassed: ptr(9000)}}

	// Attempt 1 measures everything and misses the bar, so the step fails and
	// the retry narrows.
	first, err := Evaluate(synth(8800, 1200, 0), mute, Input{ExitCode: 1, Narrowed: false})
	require.NoError(t, err)
	assert.True(t, first.ToleranceApplied)
	assert.False(t, first.Success)

	// Attempt 2 re-runs the 1200 and fixes 1150. Overall 9950 have now passed,
	// which would clear the bar - but nothing accumulates, so the 50 that remain
	// fail the step.
	second, err := Evaluate(synth(1150, 50, 0), mute, Input{ExitCode: 1, Narrowed: true})
	require.NoError(t, err)
	assert.False(t, second.ToleranceApplied)
	assert.False(t, second.Success,
		"tolerate is first-pass only: cross-attempt totals are not summed")
}

func TestEvaluate_ReportThatNamesFewerTestsThanItDeclares(t *testing.T) {
	// The failure mode the whole Unrepresented check exists for: 24 declared
	// tests with 4 failures and 5 errors, one named case, and it passed.
	report, err := Parse(strings.NewReader(strings.ReplaceAll(
		testdata.BasicJUnit, `<testsuites time="15.682687">`,
		`<testsuites tests="99" failures="7" errors="0" skipped="0" time="15.682687">`)))
	require.NoError(t, err)

	verdict, err := Evaluate(report,
		Policy{Mute: Selector{Include: []string{"**/*"}}},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)

	assert.True(t, verdict.IdentitiesIncomplete)
	assert.Equal(t, int32(90), verdict.Unrepresented)
	assert.Empty(t, verdict.Muted,
		"a mute-everything pattern must not rescue a report whose cases are unnamed")
	assert.False(t, verdict.Success)
	assert.Equal(t, int32(99), verdict.Summary.Tests, "the report's own counters are believed")
	assert.Contains(t, verdict.Describe(), "does not name")
}

func TestEvaluate_UnrepresentedReportCanStillBeTolerated(t *testing.T) {
	// Counter-based thresholds remain usable when identities are missing; only
	// mute is off the table.
	report := Report{Declared: Summary{Tests: 100, Failed: 4}}
	verdict, err := Evaluate(report,
		Policy{Tolerate: &Tolerance{MaxFailed: ptr(5)}},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)
	assert.True(t, verdict.IdentitiesIncomplete)
	assert.True(t, verdict.Tolerated)
	assert.True(t, verdict.Success)
}

func TestEvaluate_UnusedMutePatternsAreReported(t *testing.T) {
	verdict, err := Evaluate(
		synth(3, 0, 1),
		Policy{Mute: Selector{Include: []string{"test_flaky_*", "test_retired_*"}}},
		Input{ExitCode: 1},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"test_retired_*"}, verdict.UnusedMutePatterns,
		"dead quarantine config has to be visible or mute lists rot")
}

func TestEvaluate_PercentageBarsRejectAnEmptyReport(t *testing.T) {
	// An empty report is not evidence of success.
	for _, bar := range []Tolerance{
		{MinPassedPercent: ptr(90)},
		{MaxFailedPercent: ptr(10)},
	} {
		tolerance := bar
		verdict, err := Evaluate(Report{}, Policy{Tolerate: &tolerance}, Input{ExitCode: 0})
		require.NoError(t, err)
		assert.False(t, verdict.Tolerated)
	}
}

func TestEvaluate_RejectsAnUnusableMutePattern(t *testing.T) {
	_, err := Evaluate(synth(1, 0, 0), Policy{Mute: Selector{Include: []string{"["}}}, Input{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mute:")
}

func TestEvaluate_SkippedCasesDoNotFailTheStep(t *testing.T) {
	report := Report{Cases: []TestCase{
		{SuitePath: []string{"s"}, Classname: "c", Name: "a", Status: StatusPassed},
		{SuitePath: []string{"s"}, Classname: "c", Name: "b", Status: StatusSkipped},
	}}
	verdict, err := Evaluate(report, Policy{}, Input{ExitCode: 0})
	require.NoError(t, err)
	assert.True(t, verdict.Success)
	assert.Empty(t, verdict.Unexpected, "a test that never ran is not a failure")
	assert.Contains(t, verdict.Describe(), "1 skipped")
}

func TestEvaluate_ErroredCountsAsFailing(t *testing.T) {
	report := Report{Cases: []TestCase{
		{SuitePath: []string{"s"}, Classname: "c", Name: "boom", Status: StatusErrored},
	}}
	verdict, err := Evaluate(report, Policy{}, Input{ExitCode: 1})
	require.NoError(t, err)
	assert.False(t, verdict.Success)
	require.Len(t, verdict.Unexpected, 1)

	muted, err := Evaluate(report, Policy{Mute: Selector{Include: []string{"boom"}}}, Input{ExitCode: 1})
	require.NoError(t, err)
	assert.True(t, muted.Success, "an errored case is mutable like a failed one")
}
