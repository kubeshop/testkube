package testresults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// previousRun is the report a first attempt left behind: one pass, one flaky
// failure, one real failure, one error, one skip.
func previousRun() Report {
	return Report{Cases: []TestCase{
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_ok", Status: StatusPassed},
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_flaky_a", Status: StatusFailed},
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_real_bug", Status: StatusFailed},
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_boom", Status: StatusErrored},
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_later", Status: StatusSkipped},
	}}
}

func TestSelection_DefaultsToFailedAndErrored(t *testing.T) {
	entries, err := Selection{}.Resolve(previousRun())
	require.NoError(t, err)

	assert.Equal(t, []string{
		"s/c/test_flaky_a",
		"s/c/test_real_bug",
		"s/c/test_boom",
	}, entries, "a pass tells us nothing on a re-run, and a skip did not fail")
}

func TestSelection_ExcludesMutedByDefault(t *testing.T) {
	// The convergence property: without this the selection can never shrink
	// below the muted set, so a narrowing retry re-runs them forever.
	entries, err := Selection{
		Mute: Selector{Include: []string{"test_flaky_*"}},
	}.Resolve(previousRun())
	require.NoError(t, err)

	assert.Equal(t, []string{"s/c/test_real_bug", "s/c/test_boom"}, entries)
}

func TestSelection_IncludeMutedIsTheDeliberateProbe(t *testing.T) {
	entries, err := Selection{
		Mute:         Selector{Include: []string{"test_flaky_*"}},
		IncludeMuted: true,
	}.Resolve(previousRun())
	require.NoError(t, err)

	assert.Contains(t, entries, "s/c/test_flaky_a",
		"opting in is how you find out whether a muted test has started passing")
}

func TestSelection_ConvergesToEmptyWhenEveryFailureIsMuted(t *testing.T) {
	report := Report{Cases: []TestCase{
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_ok", Status: StatusPassed},
		{SuitePath: []string{"s"}, Classname: "c", Name: "test_flaky_a", Status: StatusFailed},
	}}

	entries, err := Selection{Mute: Selector{Include: []string{"test_flaky_*"}}}.Resolve(report)
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing left worth re-running, so the retry stops narrowing")
}

func TestSelection_StatusesAndFilter(t *testing.T) {
	t.Run("explicit statuses", func(t *testing.T) {
		entries, err := Selection{Statuses: []Status{StatusSkipped}}.Resolve(previousRun())
		require.NoError(t, err)
		assert.Equal(t, []string{"s/c/test_later"}, entries)
	})

	t.Run("filter narrows further", func(t *testing.T) {
		entries, err := Selection{
			Filter: Selector{Include: []string{"test_real_*"}},
		}.Resolve(previousRun())
		require.NoError(t, err)
		assert.Equal(t, []string{"s/c/test_real_bug"}, entries)
	})

	t.Run("filter exclusion", func(t *testing.T) {
		entries, err := Selection{
			Filter: Selector{Exclude: []string{"test_boom"}},
		}.Resolve(previousRun())
		require.NoError(t, err)
		assert.Equal(t, []string{"s/c/test_flaky_a", "s/c/test_real_bug"}, entries)
	})
}

func TestSelection_ProjectionShapesForRealTools(t *testing.T) {
	report := Report{Cases: []TestCase{
		{SuitePath: []string{"s"}, Classname: "tests.test_login", Name: "test_bad_password", Status: StatusFailed},
		{SuitePath: []string{"s"}, Classname: "tests.test_cart", Name: "test_empty", Status: StatusFailed},
	}}

	t.Run("pytest", func(t *testing.T) {
		entries, err := Selection{As: `{{ testcase.classname }}::{{ testcase.name }}`}.Resolve(report)
		require.NoError(t, err)
		assert.Equal(t, []string{
			"tests.test_login::test_bad_password",
			"tests.test_cart::test_empty",
		}, entries)
	})

	t.Run("maven, collapsed into one argument", func(t *testing.T) {
		entries, err := Selection{
			As:       `{{ testcase.classname }}#{{ testcase.name }}`,
			Collapse: `-Dtest={{ join(selected, ",") }}`,
		}.Resolve(report)
		require.NoError(t, err)
		assert.Equal(t, []string{
			"-Dtest=tests.test_login#test_bad_password,tests.test_cart#test_empty",
		}, entries, "one entry, so it reaches the tool as a single argument")
	})

	t.Run("go test regex", func(t *testing.T) {
		entries, err := Selection{
			As:       `^{{ testcase.name }}$`,
			Collapse: `-run={{ join(selected, "|") }}`,
		}.Resolve(report)
		require.NoError(t, err)
		assert.Equal(t, []string{"-run=^test_bad_password$|^test_empty$"}, entries)
	})
}

func TestSelection_CollapseIsSkippedForAnEmptySelection(t *testing.T) {
	// This is what lets one `args:` line serve both a full run and a rerun: an
	// empty selection contributes no arguments at all, rather than a stray flag
	// with nothing after it.
	entries, err := Selection{
		Statuses: []Status{StatusFailed},
		Filter:   Selector{Include: []string{"nothing_matches_*"}},
		Collapse: `-Dtest={{ join(selected, ",") }}`,
	}.Resolve(previousRun())
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSelection_VerbatimCasesAreNotReshaped(t *testing.T) {
	entries, err := Selection{
		Statuses: []Status{StatusErrored},
		As:       `{{ testcase.name }}`,
		Cases:    []string{"already/in/the::shape I want"},
	}.Resolve(previousRun())
	require.NoError(t, err)

	assert.Equal(t, []string{"test_boom", "already/in/the::shape I want"}, entries,
		"an explicit entry is taken as written; `as` applies only to what came from the report")
}

func TestSelection_RefusesAReportThatDidNotNameItsCases(t *testing.T) {
	// Narrowing to the handful of cases a broken report happened to spell out
	// would quietly run almost nothing.
	report := Report{
		Cases:    []TestCase{{SuitePath: []string{"s"}, Classname: "c", Name: "only_one", Status: StatusFailed}},
		Declared: Summary{Tests: 40, Failed: 12},
	}

	_, err := Selection{}.Resolve(report)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not name")
}

func TestSelection_RejectsUnusableTemplatesAndPatterns(t *testing.T) {
	t.Run("bad as template", func(t *testing.T) {
		_, err := Selection{As: `{{ testcase.nonexistent }}`}.Resolve(previousRun())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "projecting")
	})

	t.Run("bad collapse template", func(t *testing.T) {
		_, err := Selection{Collapse: `{{ unknown_function(selected) }}`}.Resolve(previousRun())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collapsing")
	})

	t.Run("bad mute pattern", func(t *testing.T) {
		_, err := Selection{Mute: Selector{Include: []string{"["}}}.Resolve(previousRun())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mute")
	})

	t.Run("bad filter pattern", func(t *testing.T) {
		_, err := Selection{Filter: Selector{Include: []string{"["}}}.Resolve(previousRun())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "select")
	})
}

func TestParseStatuses(t *testing.T) {
	statuses, err := ParseStatuses([]string{"failed", "errored", " skipped "})
	require.NoError(t, err)
	assert.Equal(t, []Status{StatusFailed, StatusErrored, StatusSkipped}, statuses)

	_, err = ParseStatuses([]string{"passed"})
	require.Error(t, err, "selecting passing tests is a mistake worth naming, not a no-op")
	assert.Contains(t, err.Error(), "tells us nothing")

	_, err = ParseStatuses([]string{"exploded"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a test case outcome")

	empty, err := ParseStatuses(nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// Addresses are the canonical ids behind the projection, which is what the
// verdict checks the report against. The projection itself may be
// unrecognisable as an address.
func TestSelection_AddressesAreCanonicalRegardlessOfProjection(t *testing.T) {
	selection := Selection{As: `^{{ testcase.name }}$`}

	entries, err := selection.Resolve(previousRun())
	require.NoError(t, err)
	addresses, err := selection.Addresses(previousRun())
	require.NoError(t, err)

	require.Len(t, addresses, len(entries))
	for _, address := range addresses {
		assert.Contains(t, address, "/", "an address keeps its segments")
		assert.NotContains(t, address, "^", "the projection is not an address")
	}
}

// An explicit list is user-supplied text in the tool's own shape, so there is
// no address to check it against and none is invented.
func TestSelection_AddressesExcludeExplicitCases(t *testing.T) {
	selection := Selection{Cases: []string{"tests.a::test_one"}}

	addresses, err := selection.Addresses(Report{})
	require.NoError(t, err)
	assert.Empty(t, addresses)
}

// The same filters apply as to Resolve, so the two never describe different
// sets of test cases.
func TestSelection_AddressesRespectTheMuteExclusion(t *testing.T) {
	selection := Selection{Mute: Selector{Include: []string{"test_flaky_*"}}}

	addresses, err := selection.Addresses(previousRun())
	require.NoError(t, err)
	for _, address := range addresses {
		assert.NotContains(t, address, "test_flaky_")
	}
}
