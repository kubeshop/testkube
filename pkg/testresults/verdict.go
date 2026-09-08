package testresults

import (
	"fmt"
	"strings"
)

// Enforce says how far the policy may move the step's verdict.
type Enforce string

const (
	// EnforceOnFailure only ever downgrades a failure into a pass. A zero exit
	// code is left alone. This is the default because overriding a green run is
	// surprising.
	EnforceOnFailure Enforce = "onFailure"

	// EnforceAlways additionally fails a zero-exit run that misses the bar,
	// which is what catches tools that exit 0 while reporting failures.
	EnforceAlways Enforce = "always"
)

// Tolerance is the pass requirement. Every field that is set must hold.
//
// It is applied only to a run that measured the whole suite - see Evaluate.
type Tolerance struct {
	MaxFailed        *int32
	MaxFailedPercent *int32
	MinPassed        *int32
	MinPassedPercent *int32
}

// Empty reports whether the tolerance demands anything.
func (t *Tolerance) Empty() bool {
	return t == nil || (t.MaxFailed == nil && t.MaxFailedPercent == nil &&
		t.MinPassed == nil && t.MinPassedPercent == nil)
}

// Policy is the step's testCases block, reduced to what the verdict needs.
type Policy struct {
	Mute     Selector
	Tolerate *Tolerance
	Enforce  Enforce
}

// Input is what the run produced, alongside the report.
type Input struct {
	// ExitCode is what the test tool returned.
	ExitCode int

	// Narrowed says the run measured a subset because `select` produced a
	// non-empty selection. It is the single condition gating Tolerate, because
	// the invariant is about population rather than ordering: a threshold is
	// only meaningful over the full suite. Keying on "is this attempt 1" would
	// wrongly apply the threshold to a separate `from: step:<id>` step, which is
	// attempt 1 of its own step yet still ran a subset.
	Narrowed bool

	// Selected are the canonical addresses the run was narrowed to, from
	// Selection.Addresses. The verdict checks them against the report the run
	// produced, because a narrowed run that tested nothing looks exactly like a
	// narrowed run that passed.
	Selected []string
}

// Verdict is the outcome of applying a policy to a report.
type Verdict struct {
	// Success is what the step status should be built from, before `negative`.
	Success bool

	// Summary counts whatever the verdict believed - the named cases normally,
	// the report's own counters when identities were incomplete.
	Summary Summary

	// Muted are the failing cases the mute selector covered.
	Muted []TestCase
	// Unexpected are the failing cases it did not.
	Unexpected []TestCase

	// Tolerated is whether the pass requirement was met.
	Tolerated bool
	// ToleranceApplied is false when no requirement was evaluated, either
	// because the run was narrowed or because none was set. In both cases the
	// bar was simply "no unexpected failures".
	ToleranceApplied bool
	// Narrowed echoes the input, so a reader can tell why a requirement was
	// skipped rather than merely that it was.
	Narrowed bool

	// UnusedMutePatterns are mute patterns that matched nothing: dead quarantine
	// config, and the thing that stops mute lists outliving their bugs.
	UnusedMutePatterns []string

	// SelectedMissing are addresses the run was narrowed to that the report does
	// not name. A few of them mean the suite shrank; all of them mean the tool
	// ran no tests at all, and NothingSelectedRan says so.
	SelectedMissing []string
	// NothingSelectedRan is set when the run was narrowed to test cases and the
	// report names none of them. The step fails on it however green the tool
	// looked, because a filter it spells differently than the report does
	// matches nothing and exits zero.
	NothingSelectedRan bool

	// IdentitiesIncomplete is set when the report described more tests than it
	// named, so mute could not be applied. See Unrepresented.
	IdentitiesIncomplete bool
	// Unrepresented is how many tests the report claimed beyond those it named.
	Unrepresented int32
}

// Evaluate applies a policy to a report.
//
// The order is fixed: classify, mute, then tolerate. `negative` is applied by
// the caller afterwards - the verdict decides what "failed" means, and negative
// inverts that.
func Evaluate(report Report, policy Policy, input Input) (Verdict, error) {
	matcher, err := policy.Mute.Matcher()
	if err != nil {
		return Verdict{}, fmt.Errorf("mute: %w", err)
	}

	verdict := Verdict{
		Summary:       report.Counts(),
		Unrepresented: report.Unrepresented(),
		Narrowed:      input.Narrowed,
	}

	// A report that describes more tests than it names cannot be muted against:
	// there is nothing to match a pattern to. Trusting only the named cases is
	// how a report declaring 4 failures and naming one passing case turns green,
	// so fall back to its own counters and leave mute out of it entirely.
	if verdict.Unrepresented > 0 {
		verdict.IdentitiesIncomplete = true
		verdict.Summary = report.Declared
	} else {
		failing := make([]TestCase, 0, len(report.Cases))
		for _, testCase := range report.Cases {
			if testCase.Status.Failing() {
				failing = append(failing, testCase)
			}
		}
		// Only failing cases can be muted; muting a passing case means nothing.
		verdict.Muted, verdict.Unexpected = matcher.Partition(failing)
		verdict.UnusedMutePatterns = matcher.UnusedIncludes()
	}

	unexpected := int32(len(verdict.Unexpected))
	if verdict.IdentitiesIncomplete {
		unexpected = verdict.Summary.Failing()
	}

	// Tolerate is consulted only for a run that measured the whole suite.
	// Otherwise the bar is simply that nothing unexpected failed.
	switch {
	case input.Narrowed, policy.Tolerate.Empty():
		verdict.Tolerated = unexpected == 0
	default:
		verdict.ToleranceApplied = true
		verdict.Tolerated = policy.Tolerate.satisfied(verdict.Summary, unexpected)
	}

	if policy.Enforce == EnforceAlways {
		verdict.Success = verdict.Tolerated
	} else {
		// Downgrade only: a passing exit code is never turned into a failure.
		verdict.Success = input.ExitCode == 0 || verdict.Tolerated
	}

	// Checked last, and allowed to override a pass under either enforce mode.
	// This is not a threshold the workflow chose to relax - it is the evidence
	// that the run measured anything at all, and without it a narrowing retry
	// whose filter matched nothing reports a green step having tested nothing.
	//
	// Skipped for a report whose identities were unusable: it can neither
	// confirm nor deny an address, and IdentitiesIncomplete already says so.
	// Note that an *empty* report is not that case - a tool that ran nothing
	// declares nothing and names nothing, which is exactly what this catches.
	if !verdict.IdentitiesIncomplete {
		verdict.SelectedMissing = missingFromReport(report, input.Selected)
		if len(input.Selected) > 0 && len(verdict.SelectedMissing) == len(input.Selected) {
			verdict.NothingSelectedRan = true
			verdict.Success = false
		}
	}

	return verdict, nil
}

// missingFromReport returns the selected addresses the report does not name.
func missingFromReport(report Report, selected []string) []string {
	if len(selected) == 0 {
		return nil
	}

	present := make(map[string]bool, len(report.Cases))
	for _, testCase := range report.Cases {
		present[testCase.ID()] = true
	}

	var missing []string
	for _, address := range selected {
		if !present[address] {
			missing = append(missing, address)
		}
	}
	return missing
}

// satisfied reports whether every threshold that was set holds.
func (t *Tolerance) satisfied(summary Summary, unexpected int32) bool {
	if t.MaxFailed != nil && unexpected > *t.MaxFailed {
		return false
	}
	if t.MinPassed != nil && summary.Passed < *t.MinPassed {
		return false
	}
	// Percentages are taken against the tests the report accounted for. With no
	// tests at all a percentage bar cannot be met, which is the safe reading:
	// an empty report is not evidence of success.
	if t.MaxFailedPercent != nil {
		if summary.Tests <= 0 || unexpected*100 > *t.MaxFailedPercent*summary.Tests {
			return false
		}
	}
	if t.MinPassedPercent != nil {
		if summary.Tests <= 0 || summary.Passed*100 < *t.MinPassedPercent*summary.Tests {
			return false
		}
	}
	return true
}

// Describe renders the verdict for the step log. Muted must never mean silent,
// so this is printed whether the step passed or failed.
func (v Verdict) Describe() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d test cases: %d passed, %d failed",
		v.Summary.Tests, v.Summary.Passed, v.Summary.Failing())
	if len(v.Muted) > 0 {
		fmt.Fprintf(&builder, " (%d muted)", len(v.Muted))
	}
	if v.Summary.Skipped > 0 {
		fmt.Fprintf(&builder, ", %d skipped", v.Summary.Skipped)
	}

	if v.IdentitiesIncomplete {
		fmt.Fprintf(&builder, " — the report describes %d tests it does not name, "+
			"so mute patterns were not applied and its own counters were used instead",
			v.Unrepresented)
		return builder.String()
	}

	fmt.Fprintf(&builder, ", %d unexpected", len(v.Unexpected))

	// Said before the pass requirement, because it explains a failure the
	// requirement cannot: the numbers above may all look fine.
	if v.NothingSelectedRan {
		fmt.Fprintf(&builder, " — this run was narrowed to %d test cases and the report names none of them,"+
			" so nothing was actually tested; check that the selection reaches your runner in the shape it expects",
			len(v.SelectedMissing))
		return builder.String()
	}
	if len(v.SelectedMissing) > 0 {
		fmt.Fprintf(&builder, " — %d selected test cases did not run", len(v.SelectedMissing))
	}

	switch {
	case v.ToleranceApplied && v.Tolerated:
		builder.WriteString(" — within the pass requirement")
	case v.ToleranceApplied:
		builder.WriteString(" — short of the pass requirement")
	case v.Narrowed:
		// Say why the requirement was skipped, not merely that the run passed:
		// a reader comparing this against a threshold they wrote needs to know
		// it was not measured here.
		builder.WriteString(" — this run measured a subset, so the pass requirement was not evaluated" +
			" and every remaining failure had to be muted or gone")
	case v.Tolerated:
		builder.WriteString(" — no unexpected failures")
	default:
		builder.WriteString(" — unexpected failures remain")
	}
	return builder.String()
}
