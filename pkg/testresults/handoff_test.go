package testresults

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useHandoffDir points the handoff at a temporary directory, the way the
// integration test framework points both binaries at one.
func useHandoffDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TESTKUBE_TW_INTERNAL_PATH", dir)
	return dir
}

func TestWriteVerdictRoundTripsByReportPath(t *testing.T) {
	useHandoffDir(t)

	written := ReportVerdict{
		Reports:    []string{"/data/reports/first.xml", "/data/reports/second.xml"},
		Muted:      []string{"suite/Class/flaky", "suite/Class/known"},
		Unexpected: 2,
		Tolerated:  true,
	}
	require.NoError(t, WriteVerdict("rabc1", written))

	verdicts, err := ReadVerdicts()
	require.NoError(t, err)

	// Every report the verdict covers resolves to it, which is what lets the
	// uploader look one up by the file it is about to send.
	require.Len(t, verdicts, 2)
	for _, report := range written.Reports {
		assert.Equal(t, written, verdicts[filepath.Clean(report)], "report %s", report)
	}
}

// The overwhelmingly common case: no step declared a testCases policy, so
// nothing was ever written. That is not an error, and must not cost the upload.
func TestReadVerdictsWithoutHandoffDirectory(t *testing.T) {
	dir := useHandoffDir(t)
	require.NoDirExists(t, filepath.Join(dir, handoffDirName))

	verdicts, err := ReadVerdicts()
	require.NoError(t, err)
	assert.Empty(t, verdicts)
}

// Each step writes its own file, so two steps in one pod do not overwrite each
// other and their reports resolve independently.
func TestReadVerdictsKeepsStepsApart(t *testing.T) {
	useHandoffDir(t)

	require.NoError(t, WriteVerdict("rfirst", ReportVerdict{
		Reports: []string{"/data/first.xml"},
		Muted:   []string{"suite/Class/first-muted"},
	}))
	require.NoError(t, WriteVerdict("rsecond", ReportVerdict{
		Reports:   []string{"/data/second.xml"},
		Muted:     []string{"suite/Class/second-muted"},
		Tolerated: true,
	}))

	verdicts, err := ReadVerdicts()
	require.NoError(t, err)

	require.Len(t, verdicts, 2)
	assert.Equal(t, []string{"suite/Class/first-muted"}, verdicts[filepath.Clean("/data/first.xml")].Muted)
	assert.False(t, verdicts[filepath.Clean("/data/first.xml")].Tolerated)
	assert.Equal(t, []string{"suite/Class/second-muted"}, verdicts[filepath.Clean("/data/second.xml")].Muted)
	assert.True(t, verdicts[filepath.Clean("/data/second.xml")].Tolerated)
}

// A handoff we cannot read costs the muted counts on that one report. It must
// not cost the reports that were written correctly, and it must not fail the
// upload - the artifact is the record either way.
func TestReadVerdictsSkipsUnreadableFiles(t *testing.T) {
	dir := useHandoffDir(t)

	require.NoError(t, WriteVerdict("rgood", ReportVerdict{
		Reports: []string{"/data/good.xml"},
		Muted:   []string{"suite/Class/muted"},
	}))
	require.NoError(t, os.WriteFile(filepath.Join(dir, handoffDirName, "broken.json"), []byte("{not json"), 0o666))

	verdicts, err := ReadVerdicts()
	require.NoError(t, err)

	require.Len(t, verdicts, 1)
	assert.Equal(t, []string{"suite/Class/muted"}, verdicts[filepath.Clean("/data/good.xml")].Muted)
}

// Both sides record absolute paths, but only one of them cleans them. Matching
// on the cleaned form is what keeps "/data/./out.xml" and "/data/out.xml" the
// same file.
func TestReadVerdictsCleansReportPaths(t *testing.T) {
	useHandoffDir(t)

	require.NoError(t, WriteVerdict("rclean", ReportVerdict{
		Reports: []string{"/data/reports/../reports/./out.xml"},
		Muted:   []string{"suite/Class/muted"},
	}))

	verdicts, err := ReadVerdicts()
	require.NoError(t, err)

	_, ok := verdicts[filepath.Clean("/data/reports/out.xml")]
	assert.True(t, ok, "the path should have been cleaned before it was keyed: %v", verdicts)
}
