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

func TestDigest_CarriesNothingAboutMuting(t *testing.T) {
	// The artifacts step uploads reports; the verdict runs in a different
	// container. Whether a failure was tolerated belongs to the step result,
	// joined by step reference - not guessed at here.
	report, err := Parse(strings.NewReader(
		`<testsuite name="s" tests="1"><testcase name="flaky" classname="c"><failure/></testcase></testsuite>`))
	require.NoError(t, err)

	digest := report.Digest()
	require.Len(t, digest.Failures, 1)
	assert.Equal(t, "s/c/flaky", digest.Failures[0].Id)
	// There is deliberately no muted field to assert on; this test exists to
	// make that a decision rather than an omission.
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
