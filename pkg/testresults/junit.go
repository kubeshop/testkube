package testresults

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const (
	// MaxMessageBytes caps a single failure message. The full text always remains
	// in the uploaded report artifact; what is kept here only has to be enough to
	// tell a reader which failure this was.
	MaxMessageBytes = 1024

	// MaxTotalMessageBytes bounds what one report may retain in messages. The
	// parser runs inside the init process alongside the test tool, so a report
	// with tens of thousands of verbose failures must not be able to consume the
	// pod's memory. Past the budget, statuses are still recorded and only the
	// message text is dropped.
	MaxTotalMessageBytes = 1024 * 1024

	// sniffBytes is how much of a file is inspected to decide whether it is a
	// JUnit report at all.
	sniffBytes = 8 * 1024
)

// Sniff reports whether a chunk of a file looks like a JUnit report.
//
// It matches the detection the artifacts post-processor has always used, so a
// file that was previously picked up as a report still is.
func Sniff(head []byte) bool {
	if len(head) > sniffBytes {
		head = head[:sniffBytes]
	}
	content := string(head)
	return strings.Contains(content, "<testsuite") || strings.Contains(content, "<testsuites")
}

// xmlResult is a <failure>, <error> or <skipped> element.
type xmlResult struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// text is the human-readable reason, preferring the attribute over the body.
// Tools emit either, and some emit an empty element with neither.
func (r xmlResult) text() string {
	if message := strings.TrimSpace(r.Message); message != "" {
		if kind := strings.TrimSpace(r.Type); kind != "" {
			return kind + ": " + message
		}
		return message
	}
	return strings.TrimSpace(r.Body)
}

// xmlTestCase is a <testcase> element. Result elements are slices because a
// single case may carry more than one, including contradictory ones.
type xmlTestCase struct {
	Name      string      `xml:"name,attr"`
	Classname string      `xml:"classname,attr"`
	Time      string      `xml:"time,attr"`
	Failures  []xmlResult `xml:"failure"`
	Errors    []xmlResult `xml:"error"`
	Skipped   []xmlResult `xml:"skipped"`
}

// Parse reads a JUnit report.
//
// It streams: each <testcase> subtree is decoded and discarded in turn rather
// than unmarshalling the whole document into one structure. A suite of ten
// thousand cases is several megabytes of XML, and the init process holds the
// test tool's memory alongside its own.
func Parse(r io.Reader) (Report, error) {
	decoder := xml.NewDecoder(r)
	var (
		report       Report
		suites       []string
		depth        int
		rootDeclared bool
		messageBytes int
	)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Report{}, fmt.Errorf("reading report: %w", err)
		}

		switch element := token.(type) {
		case xml.StartElement:
			switch strings.ToLower(element.Name.Local) {
			case "testsuites":
				// A root that declares its own totals is authoritative; the
				// suites below it are already included in them.
				if declared := declaredSummary(element); declared.Tests > 0 {
					report.Declared = declared
					rootDeclared = true
				}

			case "testsuite":
				depth++
				suites = append(suites, attr(element, "name"))
				// Only top-level suites are added up. A nested suite's counters
				// are already part of its parent's, so counting every depth
				// would double-count - BasicJUnit nests Tests.Authentication.Login
				// inside Tests.Authentication.
				if !rootDeclared && depth == 1 {
					addDeclared(&report.Declared, declaredSummary(element))
				}

			case "testcase":
				var parsed xmlTestCase
				if err := decoder.DecodeElement(&parsed, &element); err != nil {
					return Report{}, fmt.Errorf("reading test case: %w", err)
				}
				report.Cases = append(report.Cases, convertTestCase(parsed, suites, &messageBytes))
			}

		case xml.EndElement:
			// Self-closing elements produce a synthetic EndElement too, so this
			// stays balanced for <testsuite ... /> as well.
			if strings.ToLower(element.Name.Local) == "testsuite" {
				depth--
				if len(suites) > 0 {
					suites = suites[:len(suites)-1]
				}
			}
		}
	}

	return report, nil
}

// convertTestCase resolves one parsed element into a test case.
func convertTestCase(parsed xmlTestCase, suites []string, messageBytes *int) TestCase {
	testCase := TestCase{
		// The stack is reused as the parser descends, so it has to be copied.
		SuitePath:  append([]string(nil), suites...),
		Classname:  parsed.Classname,
		Name:       parsed.Name,
		Status:     StatusPassed,
		DurationMs: parseDurationMs(parsed.Time),
	}

	// A case with no result element passed. Otherwise the worst result wins, and
	// the message comes from whichever element that was - so a case carrying an
	// empty <failure/> next to a populated <error> reports the error's message
	// rather than nothing at all.
	consider := func(results []xmlResult, status Status) {
		for _, result := range results {
			previous := testCase.Status
			testCase.Status = worseOf(testCase.Status, status)
			if testCase.Status != previous || testCase.Message == "" {
				if text := result.text(); text != "" {
					testCase.Message = truncateMessage(text, messageBytes)
				}
			}
		}
	}
	consider(parsed.Skipped, StatusSkipped)
	consider(parsed.Failures, StatusFailed)
	consider(parsed.Errors, StatusErrored)

	return testCase
}

// truncateMessage bounds one message and the report's total message budget.
func truncateMessage(text string, total *int) string {
	if *total >= MaxTotalMessageBytes {
		return ""
	}
	if len(text) > MaxMessageBytes {
		text = text[:MaxMessageBytes]
	}
	*total += len(text)
	return text
}

// parseDurationMs converts a JUnit `time` attribute - seconds, as a decimal -
// into milliseconds. An unparseable or absent value is simply no duration: a
// report that misreports a time is not a report we should refuse.
func parseDurationMs(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return 0
	}
	return int64(math.Round(seconds * 1000))
}

// declaredSummary reads the counter attributes off a <testsuites> or <testsuite>.
func declaredSummary(element xml.StartElement) Summary {
	summary := Summary{
		Tests:      parseCount(attr(element, "tests")),
		Failed:     parseCount(attr(element, "failures")),
		Errored:    parseCount(attr(element, "errors")),
		Skipped:    parseCount(attr(element, "skipped")),
		DurationMs: parseDurationMs(attr(element, "time")),
	}

	// JUnit has no "passed" attribute: a report states its total and what went
	// wrong, and the passes are whatever is left. Deriving it matters because a
	// report that names no test cases is believed on its counters instead - so
	// leaving this zero made every minPassed requirement fail against a report
	// that in fact satisfied it, and reported "0 passed" for it everywhere.
	//
	// Clamped at zero: the counters come from a tool and need not add up.
	if passed := summary.Tests - summary.Failed - summary.Errored - summary.Skipped; passed > 0 {
		summary.Passed = passed
	}
	return summary
}

// addDeclared accumulates one suite's declared counters into the total.
func addDeclared(total *Summary, next Summary) {
	total.Tests += next.Tests
	total.Passed += next.Passed
	total.Failed += next.Failed
	total.Errored += next.Errored
	total.Skipped += next.Skipped
	total.DurationMs += next.DurationMs
}

func parseCount(value string) int32 {
	count, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil || count < 0 {
		return 0
	}
	return int32(count)
}

// attr finds an attribute by local name, case-insensitively.
func attr(element xml.StartElement, name string) string {
	for _, candidate := range element.Attr {
		if strings.EqualFold(candidate.Name.Local, name) {
			return candidate.Value
		}
	}
	return ""
}
