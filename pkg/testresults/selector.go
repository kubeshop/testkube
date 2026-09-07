package testresults

import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Selector names test cases by glob over TestCase.ID().
//
// The same shape serves two jobs with deliberately different treatment of an
// empty Include - see Matches and Keeps.
type Selector struct {
	Include []string
	Exclude []string
}

// Empty reports whether the selector says nothing at all.
func (s Selector) Empty() bool { return len(s.Include) == 0 && len(s.Exclude) == 0 }

// normalizePattern applies the shorthand that makes patterns usable in practice:
// a pattern naming no separator matches on the test name alone.
//
// Without it, `test_flaky_*` would never match anything, because an ID always
// has three segments and `*` does not cross a separator. Matching by name is
// also what survives the churn that makes IDs unstable - shard suffixes and
// inconsistently populated classnames both live in the leading segments.
func normalizePattern(pattern string) string {
	if strings.Contains(pattern, "/") {
		return pattern
	}
	return "**/" + pattern
}

// Validate reports the first pattern that is not a usable glob, so a workflow
// can be rejected at processing time rather than behaving oddly in the pod.
func (s Selector) Validate() error {
	for _, group := range [][]string{s.Include, s.Exclude} {
		for _, pattern := range group {
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("empty pattern: a pattern must name something")
			}
			if !doublestar.ValidatePattern(normalizePattern(pattern)) {
				return fmt.Errorf("invalid pattern %q", pattern)
			}
		}
	}
	return nil
}

// pattern is one compiled entry, tracking whether it ever matched.
type pattern struct {
	source     string
	normalized string
	used       bool
}

// Matcher applies a Selector across a whole report, remembering which patterns
// were actually needed.
//
// It is stateful for one reason: a mute pattern that matches nothing is dead
// quarantine config, and reporting it is what stops mute lists outliving the
// bugs they were written for. That cannot be answered by a pure predicate over
// a single case.
type Matcher struct {
	include []pattern
	exclude []pattern
}

// Matcher compiles the selector, or fails the same way Validate does.
func (s Selector) Matcher() (*Matcher, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	compile := func(sources []string) []pattern {
		compiled := make([]pattern, 0, len(sources))
		for _, source := range sources {
			compiled = append(compiled, pattern{source: source, normalized: normalizePattern(source)})
		}
		return compiled
	}
	return &Matcher{include: compile(s.Include), exclude: compile(s.Exclude)}, nil
}

// anyMatch reports whether any pattern matches, and marks the ones that did.
//
// It deliberately does not stop at the first match: short-circuiting would
// leave later patterns looking unused when they were merely redundant, and
// UnusedIncludes would then accuse patterns that do match real tests.
func anyMatch(patterns []pattern, id string) bool {
	matched := false
	for i := range patterns {
		// A pattern that validated cannot error here.
		if ok, _ := doublestar.Match(patterns[i].normalized, id); ok {
			patterns[i].used = true
			matched = true
		}
	}
	return matched
}

// Matches reports whether the selector names this case - the question a mute
// list asks. An empty Include names nothing, so a policy with no mute patterns
// mutes nothing.
func (m *Matcher) Matches(t TestCase) bool {
	id := t.ID()
	if !anyMatch(m.include, id) {
		return false
	}
	return !anyMatch(m.exclude, id)
}

// Keeps reports whether this case survives the selector used as a narrowing
// filter - the question `select.include`/`exclude` asks. An empty Include here
// means "do not narrow", the opposite of Matches, because a filter nobody wrote
// must not throw everything away.
func (m *Matcher) Keeps(t TestCase) bool {
	id := t.ID()
	if len(m.include) > 0 && !anyMatch(m.include, id) {
		return false
	}
	return !anyMatch(m.exclude, id)
}

// UnusedIncludes are the Include patterns that matched no case, in the order
// they were written. Meaningful only after a full pass over a report.
func (m *Matcher) UnusedIncludes() []string {
	var unused []string
	for _, p := range m.include {
		if !p.used {
			unused = append(unused, p.source)
		}
	}
	return unused
}

// Partition splits cases by whether the selector names them. Used to separate
// muted failures from the ones that still count.
func (m *Matcher) Partition(cases []TestCase) (matched, unmatched []TestCase) {
	for _, testCase := range cases {
		if m.Matches(testCase) {
			matched = append(matched, testCase)
		} else {
			unmatched = append(unmatched, testCase)
		}
	}
	return matched, unmatched
}
