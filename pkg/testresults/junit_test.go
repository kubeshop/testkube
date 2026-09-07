package testresults

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common/testdata"
)

// idsByStatus indexes the parsed cases so assertions can name what they mean.
func idsByStatus(report Report, status Status) []string {
	var ids []string
	for _, testCase := range report.Cases {
		if testCase.Status == status {
			ids = append(ids, testCase.ID())
		}
	}
	return ids
}

func TestParse_BasicJUnit(t *testing.T) {
	report, err := Parse(strings.NewReader(testdata.BasicJUnit))
	require.NoError(t, err)

	counts := report.Counts()
	assert.Equal(t, int32(9), counts.Tests)
	assert.Equal(t, int32(8), counts.Passed)
	assert.Equal(t, int32(1), counts.Failed)

	// The suites nest, and the case is identified by the innermost one.
	assert.Equal(t, []string{"Tests.Authentication", "Tests.Authentication.Login"},
		report.Cases[3].SuitePath, "testCase4 sits in a suite nested inside another")
	assert.Equal(t, "Tests.Authentication.Login/Tests.Authentication.Login/testCase4",
		report.Cases[3].ID())

	assert.Equal(t, []string{"Tests.Authentication/Tests.Authentication/testCase9"},
		idsByStatus(report, StatusFailed))

	failed := report.Cases[8]
	assert.Equal(t, "AssertionError: Assertion error message", failed.Message,
		"the failure type qualifies the message")
	assert.Equal(t, int64(982), failed.DurationMs)

	// Neither the root nor the suites declare totals, so nothing claims tests
	// beyond the ones named.
	assert.Zero(t, report.Unrepresented())
}

func TestParse_CompleteJUnit(t *testing.T) {
	report, err := Parse(strings.NewReader(testdata.CompleteJUnit))
	require.NoError(t, err)

	counts := report.Counts()
	assert.Equal(t, int32(8), counts.Tests)
	assert.Equal(t, int32(5), counts.Passed)
	assert.Equal(t, int32(1), counts.Failed)
	assert.Equal(t, int32(1), counts.Errored)
	assert.Equal(t, int32(1), counts.Skipped)

	assert.Equal(t, int32(8), report.Declared.Tests, "the root declares its own totals")
	assert.Zero(t, report.Unrepresented())

	assert.Equal(t, []string{"Tests.Registration/Tests.Registration/testCase5"},
		idsByStatus(report, StatusFailed))
	assert.Equal(t, []string{"Tests.Registration/Tests.Registration/testCase6"},
		idsByStatus(report, StatusErrored))
	assert.Equal(t, []string{"Tests.Registration/Tests.Registration/testCase4"},
		idsByStatus(report, StatusSkipped))

	// <properties>, <system-out> and <system-err> are not results and must not
	// change an outcome. testCase7 and testCase8 carry them and both passed.
	assert.Equal(t, StatusPassed, report.Cases[6].Status)
	assert.Equal(t, StatusPassed, report.Cases[7].Status)
}

func TestParse_RejectsMalformedXML(t *testing.T) {
	_, err := Parse(strings.NewReader(testdata.InvalidJUnit))
	require.Error(t, err, "a report we cannot read must not look like an empty one")
	assert.Contains(t, err.Error(), "reading report")
}

func TestParse_RootShapes(t *testing.T) {
	t.Run("testsuites wrapper without totals falls back to its suites", func(t *testing.T) {
		report, err := Parse(strings.NewReader(testdata.OneLineJUnit))
		require.NoError(t, err)
		assert.Equal(t, int32(2), report.Counts().Tests)
		assert.Equal(t, int32(2), report.Declared.Tests)
		assert.Equal(t, int32(1), report.Declared.Failed)
		assert.Equal(t, []string{"TestSuite/TestClass/Test1"}, idsByStatus(report, StatusFailed))
		assert.Equal(t, "Test failed", report.Cases[0].Message,
			"a failure with no type attribute reports the bare message")
	})

	t.Run("testsuites wrapper with totals", func(t *testing.T) {
		report, err := Parse(strings.NewReader(testdata.TestsuitesOnlyJUnit))
		require.NoError(t, err)
		assert.Equal(t, int32(2), report.Counts().Tests)
		assert.Equal(t, int32(2), report.Counts().Passed)
		assert.Equal(t, "smoke.spec.js/smoke.spec.js/Smoke 1 - has title", report.Cases[0].ID(),
			"test names contain spaces and the id keeps them verbatim")
	})

	t.Run("bare testsuite as the root", func(t *testing.T) {
		report, err := Parse(strings.NewReader(testdata.TestsuiteOnlyJUnit))
		require.NoError(t, err)
		assert.Equal(t, int32(1), report.Counts().Tests)
		assert.Equal(t, int32(1), report.Declared.Tests)
		assert.Zero(t, report.Unrepresented())
	})
}

// parsePregenerated reads one of the reports kept for the end-to-end suites.
// They are the record of shapes real tools have produced.
func parsePregenerated(t *testing.T, name string) Report {
	t.Helper()
	path := filepath.Join("..", "..", "test", "junit-pregenerated-reports", name)
	content, err := os.ReadFile(path)
	require.NoError(t, err, "fixture %s must exist", path)
	require.True(t, Sniff(content), "%s must be recognised as a JUnit report", name)
	report, err := Parse(strings.NewReader(string(content)))
	require.NoError(t, err)
	return report
}

func TestParse_HighLevelFailure(t *testing.T) {
	report := parsePregenerated(t, "high-level-failure.xml")

	counts := report.Counts()
	assert.Equal(t, int32(12), counts.Tests, "the file names 12 cases even though it declares 11")
	assert.Equal(t, int32(8), counts.Passed)
	assert.Equal(t, int32(1), counts.Failed)
	assert.Equal(t, int32(2), counts.Errored)
	assert.Equal(t, int32(1), counts.Skipped)

	assert.Equal(t, int32(11), report.Declared.Tests)
	assert.Zero(t, report.Unrepresented(),
		"naming more cases than declared is not a shortfall")

	assert.Equal(t, []string{
		"BBB/com.example.BBBTests/testError1",
		"BBB/com.example.BBBTests/testError2",
	}, idsByStatus(report, StatusErrored))

	// CDATA bodies are the reason to prefer the message attribute.
	var failure TestCase
	for _, testCase := range report.Cases {
		if testCase.Status == StatusFailed {
			failure = testCase
		}
	}
	assert.Equal(t, "AssertionError: Expected value X but got Y", failure.Message)
}

func TestParse_TestCaseWithBothErrorAndFailure(t *testing.T) {
	report := parsePregenerated(t, "high-level-testcase-both-error-and-failure.xml")

	counts := report.Counts()
	assert.Equal(t, int32(8), counts.Tests)
	assert.Equal(t, int32(6), counts.Passed)
	assert.Equal(t, int32(2), counts.Errored)
	assert.Zero(t, counts.Failed,
		"a case carrying both results resolves to the worse one, and is counted once")

	// Each of these has an empty <failure/> beside a populated <error>. The
	// status must come from the error, and so must the message - taking the
	// failure's would report nothing at all.
	both := report.Cases[6]
	assert.Equal(t, StatusErrored, both.Status)
	assert.Equal(t, "BBB/exampleClassname/Example Name 1", both.ID(),
		"the suite identifies the case; classname is a separate segment")
	assert.Contains(t, both.Message, "element click intercepted")
}

func TestParse_SuitesThatNameNoTestCases(t *testing.T) {
	// The critical case for muting. This report declares 24 tests including 4
	// failures and 5 errors, but names exactly one case, which passed. Acting on
	// the named cases alone would call it green.
	report := parsePregenerated(t, "high-level-without-testcases.xml")

	require.Len(t, report.Cases, 1)
	assert.Equal(t, "CCC/example-classname/Example name", report.Cases[0].ID())
	assert.Equal(t, StatusPassed, report.Cases[0].Status,
		"a case whose only child is <system-out> passed")

	assert.Equal(t, int32(1), report.Counts().Tests)
	assert.Equal(t, int32(24), report.Declared.Tests)
	assert.Equal(t, int32(4), report.Declared.Failed)
	assert.Equal(t, int32(5), report.Declared.Errored)
	assert.Equal(t, int32(23), report.Unrepresented(),
		"the shortfall is what stops a verdict trusting the named cases")
}

func TestSniff(t *testing.T) {
	assert.True(t, Sniff([]byte(testdata.BasicJUnit)))
	assert.True(t, Sniff([]byte(testdata.TestsuiteOnlyJUnit)))
	assert.False(t, Sniff([]byte(testdata.InvalidJUnit)))
	assert.False(t, Sniff([]byte(`{"tests": 4}`)))
	assert.False(t, Sniff(nil))

	// Only the head of a file is inspected, so a marker past the window is not
	// found - matching how the artifacts post-processor has always behaved.
	assert.False(t, Sniff([]byte(strings.Repeat(" ", sniffBytes+16)+"<testsuite>")))
}

func TestParseDurationMs(t *testing.T) {
	for input, expected := range map[string]int64{
		"":          0,
		"0":         0,
		"0.000":     0,
		"1.5":       1500,
		"2.113871":  2114,
		"  6.259  ": 6259,
		"1e-3":      1,
		"-4":        0,
		"NaN":       0,
		"later":     0,
	} {
		assert.Equal(t, expected, parseDurationMs(input), "input %q", input)
	}
}

func TestTestCaseID_PreservesEmptySegments(t *testing.T) {
	// A pattern must see the same number of separators no matter what the tool
	// filled in, so an absent suite or classname stays as an empty segment.
	assert.Equal(t, "//TestFoo", TestCase{Name: "TestFoo"}.ID())
	assert.Equal(t, "/pkg/TestFoo", TestCase{Classname: "pkg", Name: "TestFoo"}.ID())
	assert.Equal(t, "suite//TestFoo", TestCase{SuitePath: []string{"suite"}, Name: "TestFoo"}.ID())
}

func TestStatusPrecedence(t *testing.T) {
	assert.Equal(t, StatusErrored, worseOf(StatusFailed, StatusErrored))
	assert.Equal(t, StatusErrored, worseOf(StatusErrored, StatusFailed))
	assert.Equal(t, StatusFailed, worseOf(StatusSkipped, StatusFailed))
	assert.Equal(t, StatusSkipped, worseOf(StatusPassed, StatusSkipped))
	assert.Equal(t, StatusPassed, worseOf(StatusPassed, StatusPassed))

	// A skipped test neither passed nor counts against the step.
	assert.False(t, StatusSkipped.Passed())
	assert.False(t, StatusSkipped.Failing())
	assert.True(t, StatusErrored.Failing())
	assert.True(t, StatusFailed.Failing())
}

func TestMerge(t *testing.T) {
	// A step may point report.paths at a glob, so several files become one report.
	first, err := Parse(strings.NewReader(testdata.TestsuiteOnlyJUnit))
	require.NoError(t, err)
	second, err := Parse(strings.NewReader(testdata.OneLineJUnit))
	require.NoError(t, err)

	merged := Merge(first, second)
	assert.Equal(t, int32(3), merged.Counts().Tests)
	assert.Equal(t, int32(1), merged.Counts().Failed)
	assert.Equal(t, int32(3), merged.Declared.Tests, "declared totals add up too")
	assert.Zero(t, merged.Unrepresented())
	assert.Empty(t, Merge().Cases, "merging nothing is an empty report, not a panic")
}

func TestParse_MessageBudget(t *testing.T) {
	// A report full of verbose failures must not be able to grow without bound
	// in the init process. Statuses survive the budget; message text does not.
	var builder strings.Builder
	builder.WriteString(`<testsuite name="s">`)
	for i := 0; i < 2000; i++ {
		builder.WriteString(`<testcase name="t" classname="c"><failure message="`)
		builder.WriteString(strings.Repeat("x", MaxMessageBytes*2))
		builder.WriteString(`"/></testcase>`)
	}
	builder.WriteString(`</testsuite>`)

	report, err := Parse(strings.NewReader(builder.String()))
	require.NoError(t, err)
	require.Len(t, report.Cases, 2000)
	assert.Equal(t, int32(2000), report.Counts().Failed, "every status is still recorded")

	var retained int
	for _, testCase := range report.Cases {
		assert.LessOrEqual(t, len(testCase.Message), MaxMessageBytes)
		retained += len(testCase.Message)
	}
	assert.LessOrEqual(t, retained, MaxTotalMessageBytes+MaxMessageBytes)
	assert.Empty(t, report.Cases[len(report.Cases)-1].Message, "the budget ran out before the end")
}
