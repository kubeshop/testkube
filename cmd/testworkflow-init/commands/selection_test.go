package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/expressions"
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
	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}, t.TempDir())
	require.NoError(t, err)

	assert.False(t, selection.Narrowed)
	assert.Empty(t, selection.Entries)
	assert.Empty(t, selection.File)
}

func TestResolveSelection_SecondAttemptTakesThePreviousFailures(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}, dir)
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

	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
		Select:      &lite.ActionTestCasesSelect{From: "self"},
	}, dir)
	require.NoError(t, err)

	assert.NotContains(t, selection.Entries, "s/tests.a/test_flaky_a",
		"re-running a muted failure costs time and cannot change the verdict")
	assert.Len(t, selection.Entries, 2)
}

func TestResolveSelection_ProjectionReachesTheTool(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From: "self",
			As:   `{{ testcase.classname }}::{{ testcase.name }}`,
		},
	}, dir)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"tests.a::test_flaky_a",
		"tests.b::test_real_bug",
		"tests.b::test_boom",
	}, selection.Entries)
}

func TestResolveSelection_WritesTheOverflowFile(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From:      "self",
			As:        `{{ testcase.name }}`,
			WritePath: "run/selected.txt",
		},
	}, dir)
	require.NoError(t, err)

	require.NotEmpty(t, selection.File, "the path is reported so the step can name it")
	content, err := os.ReadFile(selection.File)
	require.NoError(t, err, "the parent directory has to be created")
	assert.Equal(t, "test_flaky_a\ntest_real_bug\ntest_boom\n", string(content))
	assert.Equal(t, filepath.Join(dir, "run", "selected.txt"), filepath.Clean(selection.File))
}

func TestResolveSelection_CustomSeparator(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select: &lite.ActionTestCasesSelect{
			From: "self", As: `{{ testcase.name }}`,
			WritePath: "s.txt", WriteSep: ",",
		},
	}, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(selection.File)
	require.NoError(t, err)
	assert.Equal(t, "test_flaky_a,test_real_bug,test_boom,", string(content))
}

func TestResolveSelection_NoFileWrittenWhenNothingSelected(t *testing.T) {
	// Writing an empty file would make `[ -s file ]` style checks pass by
	// accident, so there is nothing to write and nothing to name.
	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", WritePath: "s.txt"},
	}, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, selection.File)
}

func TestResolveSelection_ExplicitCasesNeedNoReport(t *testing.T) {
	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{Cases: []string{"tests.a::test_one"}},
	}, t.TempDir())
	require.NoError(t, err)

	assert.True(t, selection.Narrowed)
	assert.Equal(t, []string{"tests.a::test_one"}, selection.Entries)
}

func TestResolveSelection_RejectsABadStatus(t *testing.T) {
	dir := reportWith(t, previousAttempt)

	_, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", Status: []string{"passed"}},
	}, dir)
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
	t.Setenv(EnvSelectedTestsCount, "stale")

	selection := &testCasesSelection{Entries: []string{"a", "b"}, Narrowed: true, File: "/data/s.txt"}
	require.NoError(t, selection.Export())
	assert.Equal(t, "/data/s.txt", os.Getenv(EnvSelectedTests))
	assert.Equal(t, "2", os.Getenv(EnvSelectedTestsCount))

	// An empty selection has to clear the variables, not leave the previous
	// attempt's values for the tool to pick up.
	require.NoError(t, (&testCasesSelection{}).Export())
	assert.Empty(t, os.Getenv(EnvSelectedTests))
	assert.Equal(t, "0", os.Getenv(EnvSelectedTestsCount))
}

func TestResolveSelection_AbsoluteWritePath(t *testing.T) {
	dir := reportWith(t, previousAttempt)
	target := filepath.ToSlash(filepath.Join(t.TempDir(), "abs.txt"))
	if !strings.HasPrefix(target, "/") {
		t.Skip("absolute paths are the container's shape, not this platform's")
	}

	selection, err := resolveTestCaseSelection(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		Select:      &lite.ActionTestCasesSelect{From: "self", WritePath: target},
	}, dir)
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
	first, err := resolveTestCaseSelection(policy, dir)
	require.NoError(t, err)
	require.False(t, first.Narrowed, "the first attempt cannot narrow against nothing")

	// The tool runs and writes its report: one pass, one muted failure, two real.
	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="4">
  <testcase name="test_ok" classname="c"/>
  <testcase name="test_flaky_a" classname="c"><failure message="flaked"/></testcase>
  <testcase name="test_one" classname="c"><failure message="broken"/></testcase>
  <testcase name="test_two" classname="c"><failure message="broken"/></testcase>
</testsuite>`), 0o600))

	verdict := applyTestCases(policy, dir, 1, first.Narrowed)
	assert.False(t, verdict.Success, "two unmuted failures, so the step fails and the retry runs")
	require.NotNil(t, verdict.Results)
	assert.True(t, verdict.Results.RequirementApplied, "a full run is measured against the requirement")

	// Attempt 2: the selection now reads attempt 1's report and takes only the
	// unmuted failures - the muted one is not re-run.
	second, err := resolveTestCaseSelection(policy, dir)
	require.NoError(t, err)
	require.True(t, second.Narrowed)
	assert.Equal(t, []string{"test_one", "test_two"}, second.Entries,
		"the pass and the muted failure are both left out")

	// The tool re-runs just those two and both now pass, overwriting the report.
	require.NoError(t, os.WriteFile(report, []byte(`<testsuite name="s" tests="2">
  <testcase name="test_one" classname="c"/>
  <testcase name="test_two" classname="c"/>
</testsuite>`), 0o600))

	retried := applyTestCases(policy, dir, 0, second.Narrowed)
	assert.True(t, retried.Success)
	require.NotNil(t, retried.Results)
	assert.False(t, retried.Results.RequirementApplied,
		"minPassed:3 is not applied to a report of 2, because the run measured a subset")
	assert.Contains(t, retried.Details, "measured a subset")

	// And a third resolution finds nothing left, so the retry has converged.
	third, err := resolveTestCaseSelection(policy, dir)
	require.NoError(t, err)
	assert.False(t, third.Narrowed, "nothing failed, so there is nothing left to narrow to")
}
