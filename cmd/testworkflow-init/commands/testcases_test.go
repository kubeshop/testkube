package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func ptr(v int32) *int32 { return &v }

// reportWith writes a JUnit report naming the given outcomes and returns the
// directory holding it.
func reportWith(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "junit.xml"), []byte(body), 0o600))
	return dir
}

const mixedReport = `<testsuite name="s" tests="4">
  <testcase name="test_ok" classname="c"/>
  <testcase name="test_flaky_a" classname="c"><failure message="flaked"/></testcase>
  <testcase name="test_flaky_b" classname="c"><failure message="flaked"/></testcase>
  <testcase name="test_real_bug" classname="c"><failure message="broken"/></testcase>
</testsuite>`

const allMutedReport = `<testsuite name="s" tests="2">
  <testcase name="test_ok" classname="c"/>
  <testcase name="test_flaky_a" classname="c"><failure message="flaked"/></testcase>
</testsuite>`

func TestApplyTestCases_MutedFailuresPassTheStep(t *testing.T) {
	dir := reportWith(t, allMutedReport)

	success, details := applyTestCases(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
	}, dir, 1)

	assert.True(t, success, "the only failure was muted, so the tool's exit code is overridden")
	assert.Contains(t, details, "(1 muted)", "muted must never mean silent")
	assert.Contains(t, details, "0 unexpected")
}

func TestApplyTestCases_UnmutedFailureStillFails(t *testing.T) {
	dir := reportWith(t, mixedReport)

	success, details := applyTestCases(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
	}, dir, 1)

	assert.False(t, success)
	assert.Contains(t, details, "(2 muted)")
	assert.Contains(t, details, "1 unexpected")
}

func TestApplyTestCases_MissingReport(t *testing.T) {
	empty := t.TempDir()

	t.Run("fails by default", func(t *testing.T) {
		// The safety catch: without it, a mute policy would rescue a step that
		// crashed before writing anything.
		success, details := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			MuteInclude: []string{"**/*"},
		}, empty, 1)
		assert.False(t, success)
		assert.Contains(t, details, "no test report found")
		assert.Contains(t, details, "onMissing is fail")
	})

	t.Run("warn keeps the exit code", func(t *testing.T) {
		success, details := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"}, OnMissing: "warn",
		}, empty, 0)
		assert.True(t, success)
		assert.Contains(t, details, "no test report found")

		success, _ = applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"}, OnMissing: "warn",
		}, empty, 1)
		assert.False(t, success, "warn does not rescue a genuine failure")
	})

	t.Run("ignore is silent", func(t *testing.T) {
		success, details := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"}, OnMissing: "ignore",
		}, empty, 0)
		assert.True(t, success)
		assert.Empty(t, details)
	})
}

func TestApplyTestCases_Thresholds(t *testing.T) {
	dir := reportWith(t, mixedReport) // 1 passed, 3 failed, 4 total

	t.Run("met", func(t *testing.T) {
		success, _ := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(3)},
		}, dir, 1)
		assert.True(t, success)
	})

	t.Run("missed", func(t *testing.T) {
		success, details := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(2)},
		}, dir, 1)
		assert.False(t, success)
		assert.Contains(t, details, "short of the pass requirement")
	})

	t.Run("enforce always fails a tool that exited zero", func(t *testing.T) {
		success, _ := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(0)},
			Enforce:     "always",
		}, dir, 0)
		assert.False(t, success)
	})

	t.Run("enforce onFailure leaves a green run alone", func(t *testing.T) {
		success, _ := applyTestCases(&lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(0)},
		}, dir, 0)
		assert.True(t, success, "the default only ever downgrades a failure")
	})
}

func TestApplyTestCases_UnusedMutePatternsAreSurfaced(t *testing.T) {
	dir := reportWith(t, allMutedReport)

	_, details := applyTestCases(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*", "test_retired_*"},
	}, dir, 1)

	assert.Contains(t, details, "Mute patterns matching nothing: test_retired_*",
		"dead quarantine config has to be visible or mute lists rot")
}

func TestApplyTestCases_GlobAndMultipleReports(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "reports"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reports", "a.xml"),
		[]byte(`<testsuite name="a" tests="1"><testcase name="t1" classname="c"/></testsuite>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reports", "b.xml"),
		[]byte(`<testsuite name="b" tests="1"><testcase name="t2" classname="c"><failure message="x"/></testcase></testsuite>`), 0o600))

	success, details := applyTestCases(&lite.ActionTestCases{
		ReportPaths: []string{"reports/**/*.xml"},
	}, dir, 1)

	assert.False(t, success)
	assert.Contains(t, details, "2 test cases", "both files are merged into one report")
}

func TestApplyTestCases_UnparseableReportFailsLoudly(t *testing.T) {
	dir := reportWith(t, `<testsuite><testcase name="a"</testsuite>`)

	success, details := applyTestCases(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"**/*"},
	}, dir, 1)

	assert.False(t, success, "a report we cannot read must not be treated as empty and muted")
	assert.Contains(t, details, "could not read the test report")
}

func TestApplyTestCases_ReportNamingFewerTestsThanDeclared(t *testing.T) {
	// A suite with counters and no cases: mute cannot apply, and believing the
	// named cases alone would call this green.
	dir := reportWith(t, `<testsuites tests="10" failures="4">
  <testsuite name="s" tests="10" failures="4"></testsuite>
</testsuites>`)

	success, details := applyTestCases(&lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"**/*"},
	}, dir, 1)

	assert.False(t, success)
	assert.Contains(t, details, "does not name")
}
