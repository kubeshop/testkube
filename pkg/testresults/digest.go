package testresults

import "strings"

const (
	// MaxDigestFailures caps how many non-passing test cases a digest names.
	//
	// An execution with more failures than this is not a "re-run the failures"
	// scenario, and the complete set is always in the report file the digest
	// accompanies. The cap is what stops one pathological report from dominating
	// an execution record.
	MaxDigestFailures = 2000

	// MaxDigestFailureBytes bounds the messages a digest carries, for the same
	// reason. Past the budget a failure is still named; only its message is
	// dropped.
	MaxDigestFailureBytes = 256 * 1024
)

// DigestFailure is one non-passing test case, as the execution record holds it.
type DigestFailure struct {
	// Id is the canonical address, "<suite>/<classname>/<name>".
	Id string
	// Status is failed, errored or skipped.
	Status Status
	// Message is the failure text, truncated.
	Message string
	// Muted is whether the step's mute patterns covered this case. Only set once
	// ApplyVerdict has been called; a report on its own cannot know it.
	Muted bool
}

// Digest is the parse of a report, small enough to travel with the execution
// record and to be queried without downloading the report itself.
//
// A report on its own says which test cases failed and nothing about which
// failures were expected: muting is a property of the workflow's policy, not of
// the XML. The verdict fields below are filled in by ApplyVerdict from what the
// container that decided the step's status left behind, and stay zero when no
// verdict was recorded - which is every step that declares no testCases policy.
type Digest struct {
	Counts   Summary
	Failures []DigestFailure
	// Truncated is set when the cap was reached, so a reader knows Failures is
	// a prefix rather than the whole story.
	Truncated bool

	// Muted is how many failing test cases the mute patterns covered.
	Muted int32
	// Unexpected is how many they did not.
	Unexpected int32
	// Tolerated is whether the step met its pass requirement.
	Tolerated bool
	// VerdictApplied distinguishes "no failures were muted" from "no verdict was
	// recorded", which are the same three zeroes otherwise.
	VerdictApplied bool
}

// ApplyVerdict fills in what the report could not know on its own.
//
// Muted ids that name no case in this digest are ignored rather than counted:
// a step may name several report files, and the verdict covers their merge, so
// most of its muted ids belong to the other files. Counting them here would
// report more muted cases than the report has failures.
func (d Digest) ApplyVerdict(verdict ReportVerdict) Digest {
	muted := make(map[string]bool, len(verdict.Muted))
	for _, id := range verdict.Muted {
		muted[id] = true
	}

	failures := make([]DigestFailure, len(d.Failures))
	copy(failures, d.Failures)

	var count int32
	for i := range failures {
		if muted[failures[i].Id] {
			failures[i].Muted = true
			count++
		}
	}

	d.Failures = failures
	d.Muted = count
	// Derived here rather than taken from the verdict for the same reason: the
	// verdict's own count spans every report file the step named.
	d.Unexpected = d.Counts.Failed + d.Counts.Errored - count
	if d.Unexpected < 0 {
		d.Unexpected = 0
	}
	d.Tolerated = verdict.Tolerated
	d.VerdictApplied = true
	return d
}

// Digest summarises a report for the execution record.
//
// When the report described more test cases than it named, the declared
// counters are believed over the named ones - the same reading the verdict
// takes, and for the same reason: a report naming one passing case while
// declaring four failures is not a passing report.
func (r Report) Digest() Digest {
	digest := Digest{Counts: r.Counts()}
	if r.Unrepresented() > 0 {
		digest.Counts = r.Declared
	}

	var bytes int
	for _, testCase := range r.Cases {
		if testCase.Status == StatusPassed {
			continue
		}
		if len(digest.Failures) >= MaxDigestFailures {
			digest.Truncated = true
			break
		}

		failure := DigestFailure{Id: testCase.ID(), Status: testCase.Status}
		if bytes+len(testCase.Message) <= MaxDigestFailureBytes {
			failure.Message = testCase.Message
			bytes += len(testCase.Message)
		}
		digest.Failures = append(digest.Failures, failure)
	}

	// A report that named fewer cases than it declared has failures it cannot
	// name, so what it did name is a prefix of the truth either way.
	if r.Unrepresented() > 0 {
		digest.Truncated = true
	}

	return digest
}

// ReportFromFailures rebuilds a report from a stored failure list.
//
// The control plane keeps the non-passing test cases of a report alongside the
// execution record, which outlives the report file: artifacts are pruned on a
// retention schedule and the row is not. So a selection narrowed against an
// earlier execution can be resolved from this rather than by downloading the
// report, which also spares the pod an artifact download and the deployment an
// artifact-read capability.
//
// Only non-passing cases are stored, which costs a selection nothing: passing
// cases can never be selected - see ParseStatuses.
//
// Declared is deliberately left zero. It exists so a report that describes more
// tests than it names can be recognised, and a failure list names everything it
// describes; leaving it set would make Unrepresented misreport the difference
// and refuse the selection.
func ReportFromFailures(failures []DigestFailure) Report {
	report := Report{Cases: make([]TestCase, 0, len(failures))}
	for _, failure := range failures {
		report.Cases = append(report.Cases, TestCaseFromID(failure.Id, failure.Status))
	}
	return report
}

// TestCaseFromID reverses TestCase.ID.
//
// The id is three fields joined by "/", so splitting into exactly three parts
// puts each one back and any "/" inside the test's own name stays with it -
// which matters, because parameterized names contain all sorts of things. The
// round trip through ID is therefore exact, and that is what the glob matching
// depends on.
//
// A suite name containing "/" is the one shape this cannot take apart. The id
// still reconstructs, so patterns keep working; only `as` reading
// testcase.suite or testcase.classname would see the split in the wrong place.
func TestCaseFromID(id string, status Status) TestCase {
	parts := strings.SplitN(id, "/", 3)
	for len(parts) < 3 {
		parts = append(parts, "")
	}

	testCase := TestCase{Classname: parts[1], Name: parts[2], Status: status}
	if parts[0] != "" {
		testCase.SuitePath = []string{parts[0]}
	}
	return testCase
}
