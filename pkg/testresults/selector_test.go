package testresults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func caseWith(suite, classname, name string) TestCase {
	return TestCase{SuitePath: []string{suite}, Classname: classname, Name: name, Status: StatusFailed}
}

func TestNormalizePattern(t *testing.T) {
	// A pattern naming no separator is anchored to the name segment. Without
	// this, `*` cannot cross a separator and the pattern matches nothing.
	assert.Equal(t, "**/test_flaky_*", normalizePattern("test_flaky_*"))
	assert.Equal(t, "BBB/**", normalizePattern("BBB/**"), "a pattern with a separator is left alone")
	assert.Equal(t, "**/a/b", normalizePattern("**/a/b"))
}

func TestSelectorMatches_ByNameAlone(t *testing.T) {
	selector := Selector{Include: []string{"test_flaky_*"}}
	matcher, err := selector.Matcher()
	require.NoError(t, err)

	assert.True(t, matcher.Matches(caseWith("BBB", "com.example.Suite", "test_flaky_checkout")),
		"the shorthand matches on the name regardless of suite or classname")
	assert.False(t, matcher.Matches(caseWith("BBB", "com.example.Suite", "test_stable")))

	// The leading segments are exactly what shard suffixes and inconsistent
	// classnames churn, so matching by name has to survive them.
	assert.True(t, matcher.Matches(caseWith("BBB-shard-3", "", "test_flaky_checkout")))

	// An ID with empty segments still matches - a bare <testcase> with no suite
	// and no classname is "//name".
	assert.True(t, matcher.Matches(TestCase{Name: "test_flaky_x"}))
}

func TestSelectorMatches_QuarantinedArea(t *testing.T) {
	matcher, err := Selector{Include: []string{"Tests.Integration.Payments/**"}}.Matcher()
	require.NoError(t, err)

	assert.True(t, matcher.Matches(caseWith("Tests.Integration.Payments", "cls", "t1")))
	assert.False(t, matcher.Matches(caseWith("Tests.Integration.Orders", "cls", "t1")))
}

func TestSelectorMatches_ExcludeWins(t *testing.T) {
	matcher, err := Selector{
		Include: []string{"test_flaky_*"},
		Exclude: []string{"test_flaky_checkout_total"},
	}.Matcher()
	require.NoError(t, err)

	assert.True(t, matcher.Matches(caseWith("s", "c", "test_flaky_login")))
	assert.False(t, matcher.Matches(caseWith("s", "c", "test_flaky_checkout_total")),
		"an exclusion carves a case back out of a broad mute")
}

func TestSelectorMatchesVersusKeeps_EmptyInclude(t *testing.T) {
	// The two questions differ precisely here, and conflating them would either
	// mute everything or discard every candidate.
	matcher, err := Selector{}.Matcher()
	require.NoError(t, err)

	subject := caseWith("s", "c", "anything")
	assert.False(t, matcher.Matches(subject), "no mute patterns mutes nothing")
	assert.True(t, matcher.Keeps(subject), "no filter patterns narrows nothing")
}

func TestSelectorKeeps_AsNarrowingFilter(t *testing.T) {
	matcher, err := Selector{
		Include: []string{"Tests.Payments/**"},
		Exclude: []string{"**/t_slow"},
	}.Matcher()
	require.NoError(t, err)

	assert.True(t, matcher.Keeps(caseWith("Tests.Payments", "c", "t1")))
	assert.False(t, matcher.Keeps(caseWith("Tests.Orders", "c", "t1")), "outside the include")
	assert.False(t, matcher.Keeps(caseWith("Tests.Payments", "c", "t_slow")), "explicitly excluded")
}

func TestMatcher_UnusedIncludes(t *testing.T) {
	matcher, err := Selector{Include: []string{
		"test_flaky_*",
		"test_gone_*",
		"Tests.Old/**",
	}}.Matcher()
	require.NoError(t, err)

	for _, testCase := range []TestCase{
		caseWith("s", "c", "test_flaky_a"),
		caseWith("s", "c", "test_stable"),
	} {
		matcher.Matches(testCase)
	}

	// Dead quarantine config is what makes mute lists outlive their bugs, so
	// the patterns that never matched are reported in the order written.
	assert.Equal(t, []string{"test_gone_*", "Tests.Old/**"}, matcher.UnusedIncludes())
}

func TestMatcher_UnusedIncludesDoesNotAccuseRedundantPatterns(t *testing.T) {
	// Two patterns both matching the same case: neither is unused. Stopping at
	// the first match would wrongly report the second.
	matcher, err := Selector{Include: []string{"test_*", "**/test_flaky_a"}}.Matcher()
	require.NoError(t, err)

	require.True(t, matcher.Matches(caseWith("s", "c", "test_flaky_a")))
	assert.Empty(t, matcher.UnusedIncludes())
}

func TestMatcher_Partition(t *testing.T) {
	matcher, err := Selector{Include: []string{"test_flaky_*"}}.Matcher()
	require.NoError(t, err)

	muted, unmuted := matcher.Partition([]TestCase{
		caseWith("s", "c", "test_flaky_a"),
		caseWith("s", "c", "test_real_bug"),
		caseWith("s", "c", "test_flaky_b"),
	})

	assert.Equal(t, []string{"s/c/test_flaky_a", "s/c/test_flaky_b"}, idsOf(muted))
	assert.Equal(t, []string{"s/c/test_real_bug"}, idsOf(unmuted))
}

func idsOf(cases []TestCase) []string {
	ids := make([]string, 0, len(cases))
	for _, testCase := range cases {
		ids = append(ids, testCase.ID())
	}
	return ids
}

func TestSelectorValidate(t *testing.T) {
	assert.NoError(t, Selector{Include: []string{"a", "b/**", "**/c"}}.Validate())
	assert.NoError(t, Selector{}.Validate())

	err := Selector{Include: []string{"["}}.Validate()
	require.Error(t, err, "an unusable glob must be rejected at processing time")
	assert.Contains(t, err.Error(), `invalid pattern "["`)

	err = Selector{Exclude: []string{"  "}}.Validate()
	require.Error(t, err, "a blank pattern is a mistake, not a match-nothing")
	assert.Contains(t, err.Error(), "empty pattern")

	_, err = Selector{Include: []string{"["}}.Matcher()
	assert.Error(t, err, "Matcher refuses what Validate refuses")
}

func TestSelectorEmpty(t *testing.T) {
	assert.True(t, Selector{}.Empty())
	assert.False(t, Selector{Include: []string{"a"}}.Empty())
	assert.False(t, Selector{Exclude: []string{"a"}}.Empty())
}
