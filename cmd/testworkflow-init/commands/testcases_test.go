package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/testresults"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// TestMain redirects the verdict handoff away from the real internal volume.
// applyTestCases records its verdict for the container that uploads the report,
// and without this every test in this package would write into /.tktw on the
// machine running them.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tktw-commands")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Setenv("TESTKUBE_TW_INTERNAL_PATH", dir); err != nil {
		panic(err)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

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

	outcome := applyTestCases("rtest", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
	}, dir, 1, false)

	assert.True(t, outcome.Success, "the only failure was muted, so the tool's exit code is overridden")
	assert.Contains(t, outcome.Details, "(1 muted)", "muted must never mean silent")
	assert.Contains(t, outcome.Details, "0 unexpected")

	require.NotNil(t, outcome.Results, "the counters go into the execution record")
	assert.Equal(t, int32(2), outcome.Results.Tests)
	assert.Equal(t, int32(1), outcome.Results.Passed)
	assert.Equal(t, int32(1), outcome.Results.Failed)
	assert.Equal(t, int32(1), outcome.Results.Muted)
	assert.Zero(t, outcome.Results.Unexpected)
	assert.True(t, outcome.Results.Tolerated)
}

func TestApplyTestCases_UnmutedFailureStillFails(t *testing.T) {
	dir := reportWith(t, mixedReport)

	outcome := applyTestCases("rtest", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
	}, dir, 1, false)

	assert.False(t, outcome.Success)
	assert.Contains(t, outcome.Details, "(2 muted)")
	assert.Contains(t, outcome.Details, "1 unexpected")

	require.NotNil(t, outcome.Results)
	assert.Equal(t, int32(2), outcome.Results.Muted)
	assert.Equal(t, int32(1), outcome.Results.Unexpected)
	assert.False(t, outcome.Results.Tolerated)
}

func TestApplyTestCases_MissingReport(t *testing.T) {
	empty := t.TempDir()

	t.Run("fails by default", func(t *testing.T) {
		// The safety catch: without it, a mute policy would rescue a step that
		// crashed before writing anything.
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			MuteInclude: []string{"**/*"},
		}, empty, 1, false)
		assert.False(t, outcome.Success)
		assert.Contains(t, outcome.Details, "no test report found")
		assert.Contains(t, outcome.Details, "onMissing is fail")
		assert.Nil(t, outcome.Results, "there were no counters to report")
	})

	t.Run("warn keeps the exit code", func(t *testing.T) {
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"}, OnMissing: "warn",
		}, empty, 0, false)
		assert.True(t, outcome.Success)
		assert.Contains(t, outcome.Details, "no test report found")

		outcome = applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"}, OnMissing: "warn",
		}, empty, 1, false)
		assert.False(t, outcome.Success, "warn does not rescue a genuine failure")
	})

	t.Run("ignore is silent", func(t *testing.T) {
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"}, OnMissing: "ignore",
		}, empty, 0, false)
		assert.True(t, outcome.Success)
		assert.Empty(t, outcome.Details)
	})
}

func TestApplyTestCases_Thresholds(t *testing.T) {
	dir := reportWith(t, mixedReport) // 1 passed, 3 failed, 4 total

	t.Run("met", func(t *testing.T) {
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(3)},
		}, dir, 1, false)
		assert.True(t, outcome.Success)
		require.NotNil(t, outcome.Results)
		assert.True(t, outcome.Results.RequirementApplied,
			"a full run evaluates the requirement")
	})

	t.Run("missed", func(t *testing.T) {
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(2)},
		}, dir, 1, false)
		assert.False(t, outcome.Success)
		assert.Contains(t, outcome.Details, "short of the pass requirement")
	})

	t.Run("enforce always fails a tool that exited zero", func(t *testing.T) {
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(0)},
			Enforce:     "always",
		}, dir, 0, false)
		assert.False(t, outcome.Success)
	})

	t.Run("enforce onFailure leaves a green run alone", func(t *testing.T) {
		outcome := applyTestCases("rtest", &lite.ActionTestCases{
			ReportPaths: []string{"junit.xml"},
			Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(0)},
		}, dir, 0, false)
		assert.True(t, outcome.Success, "the default only ever downgrades a failure")
	})
}

func TestApplyTestCases_UnusedMutePatternsAreSurfaced(t *testing.T) {
	dir := reportWith(t, allMutedReport)

	outcome := applyTestCases("rtest", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*", "test_retired_*"},
	}, dir, 1, false)

	assert.Contains(t, outcome.Details, "Mute patterns matching nothing: test_retired_*",
		"dead quarantine config has to be visible or mute lists rot")
	require.NotNil(t, outcome.Results)
	assert.Equal(t, []string{"test_retired_*"}, outcome.Results.UnusedMutePatterns,
		"and it has to reach the execution record, not only the log")
}

func TestApplyTestCases_GlobAndMultipleReports(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "reports"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reports", "a.xml"),
		[]byte(`<testsuite name="a" tests="1"><testcase name="t1" classname="c"/></testsuite>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reports", "b.xml"),
		[]byte(`<testsuite name="b" tests="1"><testcase name="t2" classname="c"><failure message="x"/></testcase></testsuite>`), 0o600))

	outcome := applyTestCases("rtest", &lite.ActionTestCases{
		ReportPaths: []string{"reports/**/*.xml"},
	}, dir, 1, false)

	assert.False(t, outcome.Success)
	assert.Contains(t, outcome.Details, "2 test cases", "both files are merged into one report")
	require.NotNil(t, outcome.Results)
	assert.Equal(t, int32(2), outcome.Results.Tests)
}

func TestApplyTestCases_UnparseableReportFailsLoudly(t *testing.T) {
	dir := reportWith(t, `<testsuite><testcase name="a"</testsuite>`)

	outcome := applyTestCases("rtest", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"**/*"},
	}, dir, 1, false)

	assert.False(t, outcome.Success, "a report we cannot read must not be treated as empty and muted")
	assert.Contains(t, outcome.Details, "could not read the test report")
	assert.Nil(t, outcome.Results)
}

func TestApplyTestCases_ReportNamingFewerTestsThanDeclared(t *testing.T) {
	// A suite with counters and no cases: mute cannot apply, and believing the
	// named cases alone would call this green.
	dir := reportWith(t, `<testsuites tests="10" failures="4">
  <testsuite name="s" tests="10" failures="4"></testsuite>
</testsuites>`)

	outcome := applyTestCases("rtest", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"**/*"},
	}, dir, 1, false)

	assert.False(t, outcome.Success)
	assert.Contains(t, outcome.Details, "does not name")

	require.NotNil(t, outcome.Results)
	assert.True(t, outcome.Results.IdentitiesIncomplete)
	assert.Equal(t, int32(10), outcome.Results.Unrepresented)
	assert.Zero(t, outcome.Results.Muted, "nothing could be muted, so nothing is claimed as muted")
}

// The verdict has to reach the container that uploads the report, because that
// container can read which cases failed out of the XML but not which failures
// this step declared acceptable. Keyed by the report file, since the artifacts
// stage carries a step reference of its own.
func TestApplyTestCases_RecordsTheVerdictForTheReportUpload(t *testing.T) {
	t.Setenv("TESTKUBE_TW_INTERNAL_PATH", t.TempDir())
	dir := reportWith(t, mixedReport)

	outcome := applyTestCases("rrun1", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
	}, dir, 1, false)
	assert.False(t, outcome.Success, "one unmuted failure remains")

	verdicts, err := testresults.ReadVerdicts()
	require.NoError(t, err)

	recorded, ok := verdicts[filepath.Clean(filepath.Join(dir, "junit.xml"))]
	require.True(t, ok, "the report this verdict was read from should resolve to it: %v", verdicts)

	assert.ElementsMatch(t, []string{"s/c/test_flaky_a", "s/c/test_flaky_b"}, recorded.Muted)
	assert.Equal(t, int32(1), recorded.Unexpected)
	assert.False(t, recorded.Tolerated)
}

// A step that met its pass requirement has to say so, since that is the one part
// of the verdict the uploader could not derive even if it knew the mute patterns.
func TestApplyTestCases_RecordsToleratedWhenTheRequirementWasMet(t *testing.T) {
	t.Setenv("TESTKUBE_TW_INTERNAL_PATH", t.TempDir())
	dir := reportWith(t, mixedReport)

	outcome := applyTestCases("rrun1", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		MuteInclude: []string{"test_flaky_*"},
		Tolerate:    &lite.ActionTestCasesTolerance{MaxFailed: ptr(1)},
	}, dir, 1, false)
	assert.True(t, outcome.Success)

	verdicts, err := testresults.ReadVerdicts()
	require.NoError(t, err)

	recorded := verdicts[filepath.Clean(filepath.Join(dir, "junit.xml"))]
	assert.True(t, recorded.Tolerated)
}

// Nothing is recorded when the step produced no report to reach a verdict on,
// so the uploader has nothing to misread.
func TestApplyTestCases_RecordsNothingWithoutAReport(t *testing.T) {
	t.Setenv("TESTKUBE_TW_INTERNAL_PATH", t.TempDir())

	applyTestCases("rrun1", &lite.ActionTestCases{
		ReportPaths: []string{"junit.xml"},
		OnMissing:   "ignore",
	}, t.TempDir(), 0, false)

	verdicts, err := testresults.ReadVerdicts()
	require.NoError(t, err)
	assert.Empty(t, verdicts)
}
