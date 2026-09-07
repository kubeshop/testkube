package testresults

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
}

// Digest is the parse of a report, small enough to travel with the execution
// record and to be queried without downloading the report itself.
//
// It deliberately carries nothing about muting. A report says which test cases
// failed; whether a failure was tolerated is the step's verdict, decided in a
// different container from the one that uploads artifacts. The step result
// carries that, and the two are joined by step reference.
type Digest struct {
	Counts   Summary
	Failures []DigestFailure
	// Truncated is set when the cap was reached, so a reader knows Failures is
	// a prefix rather than the whole story.
	Truncated bool
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
