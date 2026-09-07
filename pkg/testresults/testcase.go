// Package testresults parses test reports produced by test tools and evaluates
// them against a workflow's policy.
//
// It exists because a workflow step's pass/fail verdict is decided inside the
// test pod, milliseconds after the tool exits. Anything that needs to influence
// that verdict - muting an expected failure, applying a pass threshold - has to
// be computed there, so the parser lives here rather than in the control plane.
//
// The package is deliberately open source even though the feature it serves is
// available in connected mode only. The gate is the processor preset: an open
// source deployment registers StubTestCases and so can never emit a policy for
// this code to act on. See pkg/testworkflows/testworkflowprocessor/presets.
package testresults

import (
	"fmt"
	"strings"
)

// Status is the outcome of a single test case.
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusErrored Status = "errored"
	StatusSkipped Status = "skipped"
)

// severity orders outcomes so that a test case carrying contradictory result
// elements resolves the same way every time. Tools do emit such cases - see
// test/junit-pregenerated-reports/high-level-testcase-both-error-and-failure.xml,
// where every case has both an empty <failure/> and a populated <error>.
//
// Higher wins: errored > failed > skipped > passed.
func (s Status) severity() int {
	switch s {
	case StatusErrored:
		return 3
	case StatusFailed:
		return 2
	case StatusSkipped:
		return 1
	default:
		return 0
	}
}

// worseOf returns whichever status the report should be believed on.
func worseOf(a, b Status) Status {
	if b.severity() > a.severity() {
		return b
	}
	return a
}

// Passed reports whether the case counts towards the passing total. A skipped
// case did not fail, but it did not pass either.
func (s Status) Passed() bool { return s == StatusPassed }

// Failing reports whether the case counts against the step. Skipped does not:
// a test that never ran cannot be the reason a pipeline is red.
func (s Status) Failing() bool { return s == StatusFailed || s == StatusErrored }

// TestCase is a single test case read out of a report.
type TestCase struct {
	// SuitePath is the chain of suites the case sits under, outermost first.
	// Suites nest, so this can be longer than one element.
	SuitePath []string
	// Classname is the tool's own grouping - a class, a file, a spec. Frequently
	// but not always equal to the innermost suite name, and frequently absent.
	Classname string
	// Name is the test case name. It may contain spaces.
	Name string

	Status     Status
	Message    string
	DurationMs int64
}

// Suite is the innermost suite the case belongs to, which is the one that
// identifies it. Empty when the case sits directly under the root.
func (t TestCase) Suite() string {
	if len(t.SuitePath) == 0 {
		return ""
	}
	return t.SuitePath[len(t.SuitePath)-1]
}

// ID is the canonical, tool-agnostic address of a test case: the address mute
// patterns match against and the one a selection is written out as.
//
// Empty segments are preserved rather than collapsed, so that a pattern always
// sees the same number of separators regardless of what the tool filled in -
// a case with no suite and no classname is "//TestFoo", not "TestFoo".
func (t TestCase) ID() string {
	return strings.Join([]string{t.Suite(), t.Classname, t.Name}, "/")
}

// String renders the case for a log line.
func (t TestCase) String() string {
	return fmt.Sprintf("%s (%s)", t.ID(), t.Status)
}

// Summary counts test cases by outcome.
type Summary struct {
	Tests      int32 `json:"tests"`
	Passed     int32 `json:"passed"`
	Failed     int32 `json:"failed"`
	Errored    int32 `json:"errored"`
	Skipped    int32 `json:"skipped"`
	DurationMs int64 `json:"durationMs"`
}

// add accumulates a single case.
func (s *Summary) add(t TestCase) {
	s.Tests++
	s.DurationMs += t.DurationMs
	switch t.Status {
	case StatusPassed:
		s.Passed++
	case StatusFailed:
		s.Failed++
	case StatusErrored:
		s.Errored++
	case StatusSkipped:
		s.Skipped++
	}
}

// Failing is the number of cases counting against the step: failures and
// errors, but not skips.
func (s Summary) Failing() int32 { return s.Failed + s.Errored }

// Report is a parsed test report.
type Report struct {
	// Cases are the test cases the report named, in the order the report named
	// them. A report may describe more tests than it names - see Unrepresented.
	Cases []TestCase

	// Declared is what the report's own counters claim, independent of the
	// cases it actually spells out.
	Declared Summary
}

// Counts summarises the cases the report actually named.
func (r Report) Counts() Summary {
	var summary Summary
	for _, testCase := range r.Cases {
		summary.add(testCase)
	}
	return summary
}

// Unrepresented is how many tests the report claims to have run beyond the ones
// it named.
//
// This is not a corner case: test/junit-pregenerated-reports/high-level-without-testcases.xml
// declares 24 tests with 4 failures and 5 errors while naming exactly one case,
// which passed. Believing only the named cases would turn that report green.
// Nothing may mute or select what a report never identified, so callers must
// check this before acting on Cases alone.
func (r Report) Unrepresented() int32 {
	if delta := r.Declared.Tests - int32(len(r.Cases)); delta > 0 {
		return delta
	}
	return 0
}

// Merge combines reports read from several files into one, since a step may
// point `report.paths` at a glob.
func Merge(reports ...Report) Report {
	var merged Report
	for _, report := range reports {
		merged.Cases = append(merged.Cases, report.Cases...)
		merged.Declared.Tests += report.Declared.Tests
		merged.Declared.Passed += report.Declared.Passed
		merged.Declared.Failed += report.Declared.Failed
		merged.Declared.Errored += report.Declared.Errored
		merged.Declared.Skipped += report.Declared.Skipped
		merged.Declared.DurationMs += report.Declared.DurationMs
	}
	return merged
}
