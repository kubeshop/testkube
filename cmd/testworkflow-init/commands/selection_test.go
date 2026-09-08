package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

const previousAttempt = `<testsuite name="s" tests="4">
  <testcase name="test_ok" classname="tests.a"/>
  <testcase name="test_flaky_a" classname="tests.a"><failure message="flaked"/></testcase>
  <testcase name="test_real_bug" classname="tests.b"><failure message="broken"/></testcase>
  <testcase name="test_boom" classname="tests.b"><error message="exploded"/></testcase>
</testsuite>`

func TestResolveSelection_FirstAttemptHasNothingToNarrowTo(t *testing.T) {
	// No report on disk yet, which is the normal state of the first attempt of a
	// narrowing retry. It must run the whole suite, not nothing.
	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}, t.TempDir(), nil, reportSource{})
	require.NoError(t, err)

	assert.False(t, selection.Narrowed)
	assert.Empty(t, selection.Entries)
	assert.Empty(t, selection.File)
}

func TestResolveSelection_SecondAttemptTakesThePreviousFailures(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}, dir, nil, reportSource{})
	require.NoError(t, err)

	assert.True(t, selection.Narrowed)
	assert.Equal(t, []string{
		"s/tests.a/test_flaky_a",
		"s/tests.b/test_real_bug",
		"s/tests.b/test_boom",
	}, selection.Entries, "the passing case is not re-run")
}

func TestResolveSelection_MutedCasesAreNotReRun(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}, dir, nil, reportSource{})
	require.NoError(t, err)

	assert.NotContains(t, selection.Entries, "s/tests.a/test_flaky_a",
		"re-running a muted failure costs time and cannot change the verdict")
	assert.Len(t, selection.Entries, 2)
}

func TestResolveSelection_ProjectionReachesTheTool(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From: "self",
			As:   `{{ testcase.classname }}::{{ testcase.name }}`,
		},
	}, dir, nil, reportSource{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"tests.a::test_flaky_a",
		"tests.b::test_real_bug",
		"tests.b::test_boom",
	}, selection.Entries)
}

func TestResolveSelection_WritesTheOverflowFile(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:      "self",
			As:        `{{ testcase.name }}`,
			WritePath: "run/selected.txt",
		},
	}, dir, nil, reportSource{})
	require.NoError(t, err)

	require.NotEmpty(t, selection.File, "the path is reported so the step can name it")
	content, err := os.ReadFile(selection.File)
	require.NoError(t, err, "the parent directory has to be created")
	assert.Equal(t, "test_flaky_a\ntest_real_bug\ntest_boom\n", string(content))
	assert.Equal(t, filepath.Join(dir, "run", "selected.txt"), filepath.Clean(selection.File))
}

func TestResolveSelection_CustomSeparator(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From: "self", As: `{{ testcase.name }}`,
			WritePath: "s.txt", WriteSep: ",",
		},
	}, dir, nil, reportSource{})
	require.NoError(t, err)

	content, err := os.ReadFile(selection.File)
	require.NoError(t, err)
	assert.Equal(t, "test_flaky_a,test_real_bug,test_boom,", string(content))
}

func TestResolveSelection_NoFileWrittenWhenNothingSelected(t *testing.T) {
	// Writing an empty file would make `[ -s file ]` style checks pass by
	// accident, so there is nothing to write and nothing to name.
	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", WritePath: "s.txt"},
	}, t.TempDir(), nil, reportSource{})
	require.NoError(t, err)
	assert.Empty(t, selection.File)
}

func TestResolveSelection_ExplicitCasesNeedNoReport(t *testing.T) {
	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{Cases: []string{"tests.a::test_one"}},
	}, t.TempDir(), nil, reportSource{})
	require.NoError(t, err)

	assert.True(t, selection.Narrowed)
	assert.Equal(t, []string{"tests.a::test_one"}, selection.Entries)
}

func TestResolveSelection_RejectsABadStatus(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	_, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", Status: []string{"passed"}},
	}, dir, nil, reportSource{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tells us nothing")
}

// TestSelectionMachine_EmptyListNeedsNoConditional is the property the whole
// ergonomic story rests on: `args: ['{{ testCases.selected }}']` has to work
// unchanged on a full run and on a narrowed one.
func TestSelectionMachine_EmptyListNeedsNoConditional(t *testing.T) {
	resolve := func(selection *testCasesSelection) []string {
		expr, err := expressions.CompileAndResolve("testCases.selected", selection.Machine(), expressions.FinalizerFail)
		require.NoError(t, err)
		items, err := expr.Static().SliceValue()
		require.NoError(t, err, "it must always be a list, so argv expansion applies")

		out := make([]string, 0, len(items))
		for _, item := range items {
			value, _ := expressions.NewValue(item).StringValue()
			out = append(out, value)
		}
		return out
	}

	assert.Empty(t, resolve(&testCasesSelection{}),
		"nothing selected contributes no arguments at all")
	assert.Equal(t, []string{"a", "b"},
		resolve(&testCasesSelection{Entries: []string{"a", "b"}, Narrowed: true}))
}

func TestSelectionMachine_ExposesCountAndFile(t *testing.T) {
	selection := &testCasesSelection{Entries: []string{"a", "b"}, Narrowed: true, File: "/data/s.txt"}
	machine := selection.Machine()

	for accessor, expected := range map[string]string{
		"testCases.selected.count": "2",
		"testCases.selecting":      "true",
		"testCases.selectedFile":   "/data/s.txt",
	} {
		expr, err := expressions.CompileAndResolve(accessor, machine, expressions.FinalizerFail)
		require.NoError(t, err, accessor)
		value, _ := expr.Static().StringValue()
		assert.Equal(t, expected, value, accessor)
	}
}

func TestSelection_Export(t *testing.T) {
	t.Setenv(EnvSelectedTests, "stale")
	t.Setenv(EnvSelectedTestsFile, "stale")
	t.Setenv(EnvSelectedTestsCount, "stale")

	selection := &testCasesSelection{Entries: []string{"a", "b"}, Narrowed: true, File: "/data/s.txt"}
	require.NoError(t, selection.Export())

	// The variable named for the selected tests holds the selected tests. It
	// used to hold the file path, which meant it was empty for every selection
	// that did not ask for a file - which is nearly all of them.
	assert.Equal(t, "a\nb", os.Getenv(EnvSelectedTests))
	assert.Equal(t, "/data/s.txt", os.Getenv(EnvSelectedTestsFile))
	assert.Equal(t, "2", os.Getenv(EnvSelectedTestsCount))

	// An empty selection has to clear the variables, not leave the previous
	// attempt's values for the tool to pick up.
	require.NoError(t, (&testCasesSelection{}).Export())
	assert.Empty(t, os.Getenv(EnvSelectedTests))
	assert.Empty(t, os.Getenv(EnvSelectedTestsFile))
	assert.Equal(t, "0", os.Getenv(EnvSelectedTestsCount))
}

// A selection is exported without a file whenever select.write is not asked
// for, which is the case the old behaviour got wrong.
func TestSelection_ExportWithoutAFile(t *testing.T) {
	selection := &testCasesSelection{Entries: []string{"Payments/CheckoutTest/test_total"}, Narrowed: true}
	require.NoError(t, selection.Export())

	assert.Equal(t, "Payments/CheckoutTest/test_total", os.Getenv(EnvSelectedTests))
	assert.Empty(t, os.Getenv(EnvSelectedTestsFile))
	assert.Equal(t, "1", os.Getenv(EnvSelectedTestsCount))
}

// Newlines, because a test case name may contain a space or a comma and any
// other separator would split one name into two.
func TestSelection_ExportSeparatesEntriesByNewline(t *testing.T) {
	selection := &testCasesSelection{
		Entries:  []string{"suite/Class/test_one[a, b]", "suite/Class/test two"},
		Narrowed: true,
	}
	require.NoError(t, selection.Export())

	assert.Equal(t, "suite/Class/test_one[a, b]\nsuite/Class/test two", os.Getenv(EnvSelectedTests))
	assert.Equal(t, "2", os.Getenv(EnvSelectedTestsCount))
}

func TestResolveSelection_AbsoluteWritePath(t *testing.T) {
	dir := reportWith(t, previousAttempt)
	target := filepath.ToSlash(filepath.Join(t.TempDir(), "abs.txt"))
	if !strings.HasPrefix(target, "/") {
		t.Skip("absolute paths are the container's shape, not this platform's")
	}

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", WritePath: target},
	}, dir, nil, reportSource{})
	require.NoError(t, err)
	assert.Equal(t, target, selection.File, "an absolute path is taken as written")
}

// TestNarrowingRetry_AcrossTwoAttempts is the mechanism the whole feature rests
// on, exercised in the order the retry loop actually runs it.
//
// commands.Run resolves the selection *before* the command and the verdict
// *after* it, so for a tool that overwrites its report the same file holds the
// previous attempt's results when the selection reads it and this attempt's when
// the verdict reads it. Nothing tracks attempt numbers.
func TestNarrowingRetry_AcrossTwoAttempts(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "junit.xml")
	policy := &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
		Tolerate:    &lite.ActionTestCasesTolerance{MinPassed: ptr(3)},
		Select:      &lite.ActionTestCasesSelect{From: "self", As: `{{ testcase.name }}`},
	}

	// Attempt 1: no report yet, so the whole suite runs.
	first, err := resolveTestCaseSelection(context.Background(), policy, dir, nil, reportSource{})
	require.NoError(t, err)
	require.False(t, first.Narrowed, "the first attempt cannot narrow against nothing")

	// The tool runs and writes its report: one pass, one muted failure, two real.
	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="4">
  <testcase name="test_ok" classname="c"/>
  <testcase name="test_flaky_a" classname="c"><failure message="flaked"/></testcase>
  <testcase name="test_one" classname="c"><failure message="broken"/></testcase>
  <testcase name="test_two" classname="c"><failure message="broken"/></testcase>
</testsuite>`), 0o600))

	verdict := applyTestCases("rtest", policy, dir, 1, first)
	assert.False(t, verdict.Success, "two unmuted failures, so the step fails and the retry runs")
	require.NotNil(t, verdict.Results)
	assert.True(t, verdict.Results.RequirementApplied, "a full run is measured against the requirement")

	// Attempt 2: the selection now reads attempt 1's report and takes only the
	// unmuted failures - the muted one is not re-run.
	second, err := resolveTestCaseSelection(context.Background(), policy, dir, nil, reportSource{})
	require.NoError(t, err)
	require.True(t, second.Narrowed)
	assert.Equal(t, []string{"test_one", "test_two"}, second.Entries,
		"the pass and the muted failure are both left out")

	// The tool re-runs just those two and both now pass, overwriting the report.
	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="2">
  <testcase name="test_one" classname="c"/>
  <testcase name="test_two" classname="c"/>
</testsuite>`), 0o600))

	retried := applyTestCases("rtest", policy, dir, 0, second)
	assert.True(t, retried.Success)
	require.NotNil(t, retried.Results)
	assert.False(t, retried.Results.RequirementApplied,
		"minPassed:3 is not applied to a report of 2, because the run measured a subset")
	assert.Contains(t, retried.Details, "measured a subset")

	// And a third resolution finds nothing left, so the retry has converged.
	third, err := resolveTestCaseSelection(context.Background(), policy, dir, nil, reportSource{})
	require.NoError(t, err)
	assert.False(t, third.Narrowed, "nothing failed, so there is nothing left to narrow to")
}

// TestResolveSelection_ReRunsAnotherStepsFailures covers the second in-execution
// shape: a later step that re-runs what an earlier one failed.
//
// The two steps share a file system, so this needs no reference between them -
// and it is why the selection reads its own paths rather than the step's report
// paths. A shared path could not work: the verdict has to judge *this* step's
// report while the selection draws from the other one.
func TestResolveSelection_ReRunsAnotherStepsFailures(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "first.xml"), []byte(previousAttempt), 0o600))

	policy := &lite.ActionTestCases{
		// This step's own report, which does not exist yet - the tool is about
		// to write it, and the verdict will read it afterwards.
		ReportPaths: []string{"second.xml"},
		Select: &lite.ActionTestCasesSelect{
			Paths: []string{"first.xml"},
			As:    `{{ testcase.name }}`,
			Empty: "skip",
		},
	}

	selection, err := resolveTestCaseSelection(context.Background(), policy, dir, nil, reportSource{})
	require.NoError(t, err)
	assert.True(t, selection.Narrowed)
	assert.Equal(t, []string{"test_flaky_a", "test_real_bug", "test_boom"}, selection.Entries,
		"the earlier step's failures, even though this step has written nothing")

	// The tool re-runs them and two now pass; the verdict reads this step's own
	// report, not the one the selection came from.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "second.xml"), []byte(`<testsuite name="s" tests="3">
  <testcase name="test_flaky_a" classname="tests.a"/>
  <testcase name="test_real_bug" classname="tests.b"/>
  <testcase name="test_boom" classname="tests.b"><error message="still broken"/></testcase>
</testsuite>`), 0o600))

	outcome := applyTestCases("rtest", policy, dir, 1, selection)
	require.NotNil(t, outcome.Results)
	assert.Equal(t, int32(3), outcome.Results.Tests, "the verdict judged the re-run, not the first pass")
	assert.Equal(t, int32(1), outcome.Results.Unexpected)
	assert.False(t, outcome.Success)
}

func TestResolveSelection_EmptyWhenTheOtherStepPassedEverything(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "first.xml"),
		[]byte(`<testsuite name="s" tests="1"><testcase name="ok" classname="c"/></testsuite>`), 0o600))

	selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
		ReportPaths: []string{"second.xml"},
		Select:      &lite.ActionTestCasesSelect{Paths: []string{"first.xml"}, Empty: "skip"},
	}, dir, nil, reportSource{})
	require.NoError(t, err)

	assert.False(t, selection.Narrowed,
		"nothing failed, so `empty: skip` will skip the re-run step entirely")
}

// TestResolveSelection_TestCasesNamedByTheScheduler covers the half of a rerun
// policy the pod can act on today: an explicit list carried with the execution,
// from an API or CLI caller who named the test cases rather than the workflow
// declaring them.
//
// The other half - narrowing to whatever an earlier execution failed - is read
// from that execution's report; see TestResolveSelection_SeedsFromTheRerunExecution.
func TestResolveSelection_TestCasesNamedByTheScheduler(t *testing.T) {
	rerun := &testworkflowconfig.RerunConfig{
		ExecutionId: "exec-1",
		OnlyFailed:  true,
		TestCases:   []string{"tests.a::test_one", "tests.b::test_two"},
	}

	t.Run("with no report of its own", func(t *testing.T) {
		selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Select:      &lite.ActionTestCasesSelect{From: "self"},
		}, t.TempDir(), rerun, reportSource{})
		require.NoError(t, err)

		assert.True(t, selection.Narrowed)
		assert.Equal(t, []string{"tests.a::test_one", "tests.b::test_two"}, selection.Entries,
			"taken verbatim: the caller already wrote them in the shape their tool wants")
	})

	t.Run("alongside what the report selected", func(t *testing.T) {
		dir := reportWith(t, previousAttempt)
		selection, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Select:      &lite.ActionTestCasesSelect{From: "self", As: `{{ testcase.name }}`},
		}, dir, rerun, reportSource{})
		require.NoError(t, err)

		assert.Equal(t, []string{
			"test_flaky_a", "test_real_bug", "test_boom",
			"tests.a::test_one", "tests.b::test_two",
		}, selection.Entries, "the scheduler's names join after the projected ones, unshaped")
	})

	// A policy that names an execution rather than test cases has to be read
	// from that execution's report. Running the whole suite instead would be the
	// wrong answer delivered confidently, so being unable to read it fails.
	t.Run("a reference the pod cannot read fails", func(t *testing.T) {
		_, err := resolveTestCaseSelection(context.Background(), &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Select:      &lite.ActionTestCasesSelect{From: "self"},
		}, t.TempDir(), &testworkflowconfig.RerunConfig{ExecutionId: "exec-1", OnlyFailed: true}, reportSource{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no connection to the control plane")
	})
}

// The guardrail, driven through the real path rather than the verdict alone: a
// retry narrows to the failures, the tool's filter matches nothing, and the tool
// exits zero having run no tests. Without this the step passes.
func TestNarrowingRetry_AStaleSelectionFailsTheStep(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "junit.xml")
	policy := &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}

	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="2">
  <testcase name="test_ok" classname="c"/>
  <testcase name="test_one" classname="c"><failure message="broken"/></testcase>
</testsuite>`), 0o600))

	// The retry narrows to the one failure.
	selection, err := resolveTestCaseSelection(context.Background(), policy, dir, nil, reportSource{})
	require.NoError(t, err)
	require.True(t, selection.Narrowed)
	require.Equal(t, []string{"s/c/test_one"}, selection.Addresses)

	// The tool ran nothing and wrote a report naming nobody the selection asked
	// for, then exited zero.
	require.NoError(t, os.WriteFile(report, []byte(
		`<testsuite name="s" tests="0"></testsuite>`), 0o600))

	outcome := applyTestCases("rtest", policy, dir, 0, selection)

	assert.False(t, outcome.Success, "a run that tested none of its selection cannot pass")
	assert.Contains(t, outcome.Details, "the report names none of them")
}

// The same path when the selection did run: the guardrail must not fire on the
// ordinary case it exists to protect.
func TestNarrowingRetry_ASelectionThatRanPasses(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "junit.xml")
	policy := &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}

	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="2">
  <testcase name="test_ok" classname="c"/>
  <testcase name="test_one" classname="c"><failure message="broken"/></testcase>
</testsuite>`), 0o600))

	selection, err := resolveTestCaseSelection(context.Background(), policy, dir, nil, reportSource{})
	require.NoError(t, err)
	require.Equal(t, []string{"s/c/test_one"}, selection.Addresses)

	// The retry re-ran it and it passed this time.
	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="1">
  <testcase name="test_one" classname="c"/>
</testsuite>`), 0o600))

	outcome := applyTestCases("rtest", policy, dir, 0, selection)

	assert.True(t, outcome.Success)
	assert.NotContains(t, outcome.Details, "did not run")
}
