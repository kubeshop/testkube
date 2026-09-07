package v1

// StepTestCases declares what a step's test report means: which failing test
// cases may be tolerated, what the pass requirement is, and which test cases
// this run should be narrowed to.
//
// It is a sibling of the step operations rather than part of StepControl,
// because StepControl is applied to the step's group stage while this has to
// land on the specific container that ran the tool. A step with both `shell`
// and `artifacts` produces two container stages, and a report policy on the
// artifacts one means nothing.
//
// Available in connected mode only. An open source deployment registers
// StubTestCases and rejects a workflow that sets this.
type StepTestCases struct {
	// report to read after the command finishes
	Report *TestCaseReport `json:"report,omitempty" expr:"include"`

	// test cases whose failure must not fail the step
	Mute *TestCaseSelector `json:"mute,omitempty" expr:"include"`

	// pass requirement, evaluated over the failures `mute` did not cover
	Tolerate *TestCaseTolerance `json:"tolerate,omitempty" expr:"include"`

	// how far the policy may move the verdict: onFailure (default) only turns a
	// failure into a pass, always additionally fails a zero-exit run that misses
	// the requirement
	// +kubebuilder:validation:Enum=onFailure;always
	Enforce string `json:"enforce,omitempty" expr:"template"`

	// test cases this run should be narrowed to
	Select *TestCaseSelection `json:"select,omitempty" expr:"include"`
}

// TestCaseReport says where the step's test report is and what to do when it is
// missing.
type TestCaseReport struct {
	// report format; only junit is understood today
	// +kubebuilder:validation:Enum=junit
	Format string `json:"format,omitempty" expr:"template"`

	// report paths, relative to the run container's working directory; globs are
	// allowed and absolute paths are accepted
	Paths []string `json:"paths,omitempty" expr:"template"`

	// what a missing report means: fail (default), warn, or ignore.
	//
	// Failing by default is the safety catch that stops `mute` masking a step
	// that crashed before it could write a report.
	// +kubebuilder:validation:Enum=fail;warn;ignore
	OnMissing string `json:"onMissing,omitempty" expr:"template"`
}

// TestCaseSelector names test cases by glob over "<suite>/<classname>/<name>".
//
// A pattern naming no separator is matched against the test name alone, which
// is both what users expect and what survives the churn that makes identities
// unstable - shard suffixes and inconsistently populated classnames both live
// in the leading segments.
type TestCaseSelector struct {
	Include []string `json:"include,omitempty" expr:"template"`
	Exclude []string `json:"exclude,omitempty" expr:"template"`
}

// TestCaseTolerance is the pass requirement. Every field that is set must hold.
//
// It is applied only to a run that measured the whole suite. A run narrowed by
// `select` skips it, because a threshold over a subset means nothing - so it
// does not accumulate across narrowing retries.
type TestCaseTolerance struct {
	// +kubebuilder:validation:Minimum=0
	MaxFailed *int32 `json:"maxFailed,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MaxFailedPercent *int32 `json:"maxFailedPercent,omitempty"`

	// +kubebuilder:validation:Minimum=0
	MinPassed *int32 `json:"minPassed,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MinPassedPercent *int32 `json:"minPassedPercent,omitempty"`
}

// TestCaseSelection narrows a run to specific test cases, so a retry re-runs
// only what failed instead of the whole suite.
type TestCaseSelection struct {
	// which execution the previous results come from. Defaults to self, this
	// execution. Reading another execution's results is not available yet.
	// +kubebuilder:validation:Enum=self
	From string `json:"from,omitempty" expr:"template"`

	// report paths the selection reads the previous results from, defaulting to
	// report.paths - this step's own previous attempt.
	//
	// Leaving it unset is what makes a retry re-run only what failed: the
	// selection is resolved before the command and the verdict after it, so for
	// a tool that overwrites its report the file holds the previous attempt's
	// results exactly when the selection reads it, and this attempt's by the
	// time the verdict does.
	//
	// Point it at an earlier step's report to re-run what *that* step failed.
	// The two steps share a file system, so this needs no reference between
	// them - and it keeps the file the selection reads separate from the one
	// the verdict judges, which a shared path could not.
	Paths []string `json:"paths,omitempty" expr:"template"`

	// which outcomes to take; defaults to failed and errored
	// +kubebuilder:validation:items:Enum=failed;errored;skipped
	Status []string `json:"status,omitempty" expr:"template"`

	// narrow further, same glob form as mute
	Include []string `json:"include,omitempty" expr:"template"`
	Exclude []string `json:"exclude,omitempty" expr:"template"`

	// an explicit list of test cases, instead of or alongside `from`
	Cases []string `json:"cases,omitempty" expr:"template"`

	// re-run cases that `mute` covers. Off by default: a muted failure is one we
	// have already decided not to care about, so re-running it costs time on
	// every attempt and can never change the verdict. It is also what lets a
	// narrowing retry converge - otherwise the selection never shrinks below the
	// muted set. Turn it on only to find out whether a muted test has started
	// passing again.
	IncludeMuted bool `json:"includeMuted,omitempty"`

	// per-entry projection handed to the tool; `testcase.{id,suite,classname,name,status}`
	// is in scope. Defaults to the canonical id.
	//
	// This is what keeps the feature tool-agnostic: Testkube never learns a
	// runner's filter flag, it renders identities in the shape the user asks for.
	As string `json:"as,omitempty" expr:"template"`

	// collapse the whole selection into a single entry; `selected` is in scope.
	// Only evaluated when the selection is not empty.
	Collapse string `json:"collapse,omitempty" expr:"template"`

	// write the entries to a file, for a selection too large to pass as
	// arguments or a runner that wants an arguments file
	Write *TestCaseSelectionWrite `json:"write,omitempty" expr:"include"`

	// what an empty selection means: all (default) runs everything, skip skips
	// the step, fail fails it.
	//
	// all is right for `from: self`, where the first attempt has produced no
	// report yet and must run the whole suite. It is usually wrong for a
	// dedicated re-run step, which wants skip.
	// +kubebuilder:validation:Enum=all;skip;fail
	Empty string `json:"empty,omitempty" expr:"template"`
}

// TestCaseSelectionWrite writes the selection to a file in the step's file system.
type TestCaseSelectionWrite struct {
	Path string `json:"path" expr:"template"`

	// entry separator; defaults to a newline
	Separator string `json:"separator,omitempty" expr:"template"`
}
