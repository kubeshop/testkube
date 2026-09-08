package testresults

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigest_NamesTheNonPassingCases(t *testing.T) {
	report, err := Parse(strings.NewReader(`<testsuite name="s" tests="4">
  <testcase name="ok" classname="c"/>
  <testcase name="broke" classname="c"><failure message="assert failed"/></testcase>
  <testcase name="boom" classname="c"><error message="panic"/></testcase>
  <testcase name="later" classname="c"><skipped/></testcase>
</testsuite>`))
	require.NoError(t, err)

	digest := report.Digest()
	assert.Equal(t, int32(4), digest.Counts.Tests)
	assert.Equal(t, int32(1), digest.Counts.Passed)
	assert.False(t, digest.Truncated)

	require.Len(t, digest.Failures, 3, "a passing case is not a failure worth naming")
	assert.Equal(t, DigestFailure{Id: "s/c/broke", Status: StatusFailed, Message: "assert failed"}, digest.Failures[0])
	assert.Equal(t, StatusErrored, digest.Failures[1].Status)
	assert.Equal(t, StatusSkipped, digest.Failures[2].Status, "a skip is non-passing, so it is named")
}

func TestDigest_CarriesNothingAboutMutingOnItsOwn(t *testing.T) {
	// A report says which test cases failed. Which of those failures the
	// workflow declared acceptable is not in the XML - it is the step's verdict,
	// decided in another container - so a bare digest must not claim to know.
	report, err := Parse(strings.NewReader(
		`<testsuite name="s" tests="1"><testcase name="flaky" classname="c"><failure/></testcase></testsuite>`))
	require.NoError(t, err)

	digest := report.Digest()
	require.Len(t, digest.Failures, 1)
	assert.Equal(t, "s/c/flaky", digest.Failures[0].Id)
	assert.False(t, digest.Failures[0].Muted)
	assert.False(t, digest.VerdictApplied, "nothing has told this digest what the verdict was")
	assert.Zero(t, digest.Muted)
	assert.Zero(t, digest.Unexpected)
}

func TestDigest_ApplyVerdictMarksTheMutedFailures(t *testing.T) {
	report, err := Parse(strings.NewReader(`<testsuite name="s" tests="3">
  <testcase name="ok" classname="c"/>
  <testcase name="known" classname="c"><failure message="expected"/></testcase>
  <testcase name="real" classname="c"><failure message="not expected"/></testcase>
</testsuite>`))
	require.NoError(t, err)

	digest := report.Digest().ApplyVerdict(ReportVerdict{
		Muted:     []string{"s/c/known"},
		Tolerated: true,
	})

	require.Len(t, digest.Failures, 2)
	byID := map[string]DigestFailure{}
	for _, failure := range digest.Failures {
		byID[failure.Id] = failure
	}
	assert.True(t, byID["s/c/known"].Muted)
	assert.False(t, byID["s/c/real"].Muted)

	assert.True(t, digest.VerdictApplied)
	assert.Equal(t, int32(1), digest.Muted)
	assert.Equal(t, int32(1), digest.Unexpected)
	assert.True(t, digest.Tolerated)
}

// A step may name several report files and the verdict covers their merge, so
// most of its muted ids belong to the other files. Counting those here would
// report more muted cases than this report has failures.
func TestDigest_ApplyVerdictIgnoresMutedIdsFromOtherReports(t *testing.T) {
	report, err := Parse(strings.NewReader(
		`<testsuite name="s" tests="1"><testcase name="known" classname="c"><failure/></testcase></testsuite>`))
	require.NoError(t, err)

	digest := report.Digest().ApplyVerdict(ReportVerdict{
		Muted: []string{"s/c/known", "other/c/elsewhere", "other/c/also-elsewhere"},
	})

	assert.Equal(t, int32(1), digest.Muted, "only the case this report names counts")
	assert.Equal(t, int32(0), digest.Unexpected)
}

// Muting cannot make a report have fewer failures than zero, whatever the
// verdict claims - a mismatched handoff must not produce a negative counter.
func TestDigest_ApplyVerdictNeverReportsNegativeUnexpected(t *testing.T) {
	report, err := Parse(strings.NewReader(
		`<testsuite name="s" tests="1"><testcase name="ok" classname="c"/></testsuite>`))
	require.NoError(t, err)

	digest := report.Digest().ApplyVerdict(ReportVerdict{Muted: []string{"s/c/ok"}})

	assert.Equal(t, int32(0), digest.Unexpected)
}

// Skipped cases are named as non-passing but are not failures, so muting must
// not count them and they must not inflate the unexpected total either.
func TestDigest_ApplyVerdictCountsFailuresNotSkips(t *testing.T) {
	report, err := Parse(strings.NewReader(`<testsuite name="s" tests="3">
  <testcase name="skipped" classname="c"><skipped/></testcase>
  <testcase name="known" classname="c"><failure/></testcase>
  <testcase name="boom" classname="c"><error/></testcase>
</testsuite>`))
	require.NoError(t, err)

	digest := report.Digest().ApplyVerdict(ReportVerdict{Muted: []string{"s/c/known"}})

	assert.Equal(t, int32(1), digest.Muted)
	assert.Equal(t, int32(1), digest.Unexpected, "the errored case, not the skipped one")
}

// ApplyVerdict must not write through to the digest it was called on, which the
// caller may still be holding.
func TestDigest_ApplyVerdictDoesNotMutateTheOriginal(t *testing.T) {
	report, err := Parse(strings.NewReader(
		`<testsuite name="s" tests="1"><testcase name="known" classname="c"><failure/></testcase></testsuite>`))
	require.NoError(t, err)

	original := report.Digest()
	_ = original.ApplyVerdict(ReportVerdict{Muted: []string{"s/c/known"}})

	require.Len(t, original.Failures, 1)
	assert.False(t, original.Failures[0].Muted)
	assert.False(t, original.VerdictApplied)
}

func TestDigest_BelievesDeclaredCountersWhenCasesAreUnnamed(t *testing.T) {
	// The same reading the verdict takes: a report naming one passing case while
	// declaring four failures is not a passing report.
	report, err := Parse(strings.NewReader(`<testsuites tests="10" failures="4">
  <testsuite name="s" tests="10" failures="4">
    <testcase name="ok" classname="c"/>
  </testsuite>
</testsuites>`))
	require.NoError(t, err)

	digest := report.Digest()
	assert.Equal(t, int32(10), digest.Counts.Tests)
	assert.Equal(t, int32(4), digest.Counts.Failed)
	assert.Empty(t, digest.Failures, "it named no failures, so none can be listed")
	assert.True(t, digest.Truncated, "the failures it has are ones it cannot name")
}

func TestDigest_CapsTheFailureList(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`<testsuite name="s">`)
	for i := 0; i < MaxDigestFailures+50; i++ {
		builder.WriteString(`<testcase name="t" classname="c"><failure message="x"/></testcase>`)
	}
	builder.WriteString(`</testsuite>`)

	report, err := Parse(strings.NewReader(builder.String()))
	require.NoError(t, err)

	digest := report.Digest()
	assert.Len(t, digest.Failures, MaxDigestFailures)
	assert.True(t, digest.Truncated, "a reader must know this is a prefix")
	assert.Equal(t, int32(MaxDigestFailures+50), digest.Counts.Tests,
		"the counts still describe the whole report")
}

func TestDigest_CapsTheMessageBudget(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`<testsuite name="s">`)
	for i := 0; i < 900; i++ {
		builder.WriteString(`<testcase name="t" classname="c"><failure message="`)
		builder.WriteString(strings.Repeat("x", MaxMessageBytes))
		builder.WriteString(`"/></testcase>`)
	}
	builder.WriteString(`</testsuite>`)

	report, err := Parse(strings.NewReader(builder.String()))
	require.NoError(t, err)

	digest := report.Digest()
	require.Len(t, digest.Failures, 900, "every failure is still named")

	var bytes int
	for _, failure := range digest.Failures {
		bytes += len(failure.Message)
	}
	assert.LessOrEqual(t, bytes, MaxDigestFailureBytes)
	assert.Empty(t, digest.Failures[len(digest.Failures)-1].Message, "the budget ran out before the end")
}

func TestDigest_EmptyReport(t *testing.T) {
	digest := Report{}.Digest()
	assert.Zero(t, digest.Counts.Tests)
	assert.Empty(t, digest.Failures)
	assert.False(t, digest.Truncated)
}
