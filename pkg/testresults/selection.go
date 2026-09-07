package testresults

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/pkg/expressions"
)

// DefaultSelectStatuses is what a selection takes when it does not say: the
// outcomes worth re-running. A skipped test did not fail, so re-running it
// would not tell us anything new.
var DefaultSelectStatuses = []Status{StatusFailed, StatusErrored}

// Selection narrows a run to specific test cases, drawn from a report an
// earlier run produced.
type Selection struct {
	// Statuses to take; DefaultSelectStatuses when empty.
	Statuses []Status
	// Filter narrows the taken cases further.
	Filter Selector
	// Mute is the step's mute policy. Its cases are dropped unless IncludeMuted,
	// because a failure we have already decided not to care about cannot change
	// the verdict, and re-running it every attempt is what stops a narrowing
	// retry from ever converging.
	Mute         Selector
	IncludeMuted bool
	// Cases are entries supplied directly rather than read from a report. They
	// are taken verbatim: As does not apply, because the caller has already
	// written them in the shape their tool wants.
	Cases []string
	// As projects one test case into the string handed to the tool.
	// `testcase.{id,suite,classname,name,status}` is in scope. Defaults to the
	// canonical id.
	As string
	// Collapse folds the whole projection into a single entry, for a tool that
	// wants one delimited argument. `selected` is in scope. Only applied when
	// the projection is non-empty.
	Collapse string
}

// Resolve picks the test cases out of a report and projects them into the
// entries a tool is invoked with.
//
// An empty result is not an error: on the first attempt of a narrowing retry
// there is no previous report, and the caller's `empty` policy decides what
// that means.
func (s Selection) Resolve(report Report) ([]string, error) {
	picked, err := s.pick(report)
	if err != nil {
		return nil, err
	}

	entries, err := s.project(picked)
	if err != nil {
		return nil, err
	}

	// Verbatim entries join after projection, so they are never reshaped.
	entries = append(entries, s.Cases...)

	if len(entries) == 0 {
		return nil, nil
	}
	return s.collapse(entries)
}

// pick applies the status filter, the mute exclusion and the include/exclude
// filter, in that order.
func (s Selection) pick(report Report) ([]TestCase, error) {
	// Nothing may be selected from a report that did not name its test cases -
	// there is no identity to hand a tool. Saying so beats narrowing a run to
	// the handful of cases a broken report happened to spell out.
	if report.Unrepresented() > 0 {
		return nil, fmt.Errorf("the report describes %d test cases it does not name, so a selection cannot address them",
			report.Unrepresented())
	}

	wanted := s.Statuses
	if len(wanted) == 0 {
		wanted = DefaultSelectStatuses
	}
	byStatus := make(map[Status]bool, len(wanted))
	for _, status := range wanted {
		byStatus[status] = true
	}

	muted, err := s.Mute.Matcher()
	if err != nil {
		return nil, fmt.Errorf("mute: %w", err)
	}
	filter, err := s.Filter.Matcher()
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}

	var picked []TestCase
	for _, testCase := range report.Cases {
		if !byStatus[testCase.Status] {
			continue
		}
		if !s.IncludeMuted && muted.Matches(testCase) {
			continue
		}
		if !filter.Keeps(testCase) {
			continue
		}
		picked = append(picked, testCase)
	}
	return picked, nil
}

// project renders each case through As.
func (s Selection) project(cases []TestCase) ([]string, error) {
	if len(cases) == 0 {
		return nil, nil
	}

	template := strings.TrimSpace(s.As)
	entries := make([]string, 0, len(cases))
	for _, testCase := range cases {
		if template == "" {
			entries = append(entries, testCase.ID())
			continue
		}
		rendered, err := expressions.CompileAndResolveTemplate(template, testCaseMachine(testCase), expressions.FinalizerFail)
		if err != nil {
			return nil, fmt.Errorf("projecting %s through %q: %w", testCase.ID(), template, err)
		}
		value, _ := rendered.Static().StringValue()
		entries = append(entries, value)
	}
	return entries, nil
}

// collapse folds the entries into one, when asked.
func (s Selection) collapse(entries []string) ([]string, error) {
	template := strings.TrimSpace(s.Collapse)
	if template == "" {
		return entries, nil
	}

	list := make([]interface{}, len(entries))
	for i := range entries {
		list[i] = entries[i]
	}
	machine := expressions.NewMachine().Register("selected", list)

	rendered, err := expressions.CompileAndResolveTemplate(template, machine, expressions.FinalizerFail)
	if err != nil {
		return nil, fmt.Errorf("collapsing the selection through %q: %w", template, err)
	}
	value, _ := rendered.Static().StringValue()
	return []string{value}, nil
}

// testCaseMachine exposes one test case to an `as` template.
func testCaseMachine(testCase TestCase) expressions.Machine {
	return expressions.NewMachine().
		Register("testcase.id", testCase.ID()).
		Register("testcase.suite", testCase.Suite()).
		Register("testcase.classname", testCase.Classname).
		Register("testcase.name", testCase.Name).
		Register("testcase.status", string(testCase.Status))
}

// ParseStatuses converts the spec's status names, rejecting anything that is
// not an outcome a report can carry.
func ParseStatuses(names []string) ([]Status, error) {
	statuses := make([]Status, 0, len(names))
	for _, name := range names {
		switch status := Status(strings.TrimSpace(name)); status {
		case StatusFailed, StatusErrored, StatusSkipped:
			statuses = append(statuses, status)
		case StatusPassed:
			return nil, fmt.Errorf("status %q cannot be selected: re-running a test that passed tells us nothing", name)
		default:
			return nil, fmt.Errorf("status %q is not a test case outcome", name)
		}
	}
	return statuses, nil
}
