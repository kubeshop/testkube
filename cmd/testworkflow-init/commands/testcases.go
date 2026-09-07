package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testresults"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// Environment variables carrying the selection, for a shell that would rather
// not write an expression.
const (
	EnvSelectedTests      = "TK_SELECTED_TESTS"
	EnvSelectedTestsCount = "TK_SELECTED_TESTS_COUNT"
)

// testCasesSelection is the resolved selection for one run of a step.
type testCasesSelection struct {
	// Entries are what the tool should be invoked with, already projected.
	// Always non-nil in intent: empty means "run everything".
	Entries []string
	// Narrowed is whether anything was actually selected. It gates the pass
	// requirement, because a threshold over a subset means nothing.
	Narrowed bool
	// File is where the entries were written, empty when write was not asked
	// for or there was nothing to write.
	File string
}

// resolveTestCaseSelection reads the report a previous run left behind and works
// out which test cases this run should be narrowed to.
//
// It runs *before* the command, which is what makes a narrowing retry work
// without any bookkeeping: for a tool that overwrites its report, the file on
// disk holds the previous attempt's results exactly when this reads it, and
// this attempt's results by the time the verdict reads it.
//
// Finding no report is not an error - the first attempt has not produced one -
// and neither is selecting nothing. The caller's `empty` policy decides.
func resolveTestCaseSelection(policy *lite.ActionTestCases, workingDir string) (*testCasesSelection, error) {
	selection := &testCasesSelection{}
	if policy.Select == nil {
		return selection, nil
	}

	report, found, err := readTestReport(policy.ReportPaths, workingDir)
	if err != nil {
		return nil, fmt.Errorf("reading the previous report: %w", err)
	}

	if found {
		statuses, err := testresults.ParseStatuses(policy.Select.Status)
		if err != nil {
			return nil, err
		}
		entries, err := testresults.Selection{
			Statuses:     statuses,
			Filter:       testresults.Selector{Include: policy.Select.Include, Exclude: policy.Select.Exclude},
			Mute:         testresults.Selector{Include: policy.MuteInclude, Exclude: policy.MuteExclude},
			IncludeMuted: policy.Select.IncludeMuted,
			Cases:        policy.Select.Cases,
			As:           policy.Select.As,
			Collapse:     policy.Select.Collapse,
		}.Resolve(report)
		if err != nil {
			return nil, err
		}
		selection.Entries = entries
	} else if len(policy.Select.Cases) > 0 {
		// An explicit list does not need a report to have existed.
		selection.Entries = append(selection.Entries, policy.Select.Cases...)
	}

	selection.Narrowed = len(selection.Entries) > 0

	if selection.Narrowed && policy.Select.WritePath != "" {
		path, err := writeSelection(selection.Entries, policy.Select.WritePath, policy.Select.WriteSep, workingDir)
		if err != nil {
			return nil, err
		}
		selection.File = path
	}

	return selection, nil
}

// Machine exposes the selection to the step's command.
//
// `testCases.selected` is always a list, empty when nothing was selected. That
// is the property that lets one `args:` line serve both a full run and a
// narrowed one: an empty list expands into no arguments at all, so the user
// writes no conditional.
func (s *testCasesSelection) Machine() expressions.Machine {
	entries := make([]interface{}, len(s.Entries))
	for i := range s.Entries {
		entries[i] = s.Entries[i]
	}
	return expressions.NewMachine().
		Register("testCases.selected", entries).
		Register("testCases.selected.count", len(s.Entries)).
		Register("testCases.selecting", s.Narrowed).
		Register("testCases.selectedFile", s.File)
}

// Export puts the selection in the environment, so a shell can use it without
// touching the expression language.
func (s *testCasesSelection) Export() error {
	if err := os.Setenv(EnvSelectedTests, s.File); err != nil {
		return err
	}
	return os.Setenv(EnvSelectedTestsCount, strconv.Itoa(len(s.Entries)))
}

// writeSelection writes the entries to a file, one per separator, and returns
// the path the step should read.
//
// This is the overflow path: a selection of thousands of test cases would not
// survive being passed as arguments, and some runners want an arguments file
// anyway.
func writeSelection(entries []string, target, separator, workingDir string) (string, error) {
	if separator == "" {
		separator = "\n"
	}

	full := target
	if !strings.HasPrefix(target, "/") {
		base := workingDir
		if base == "" {
			base = "/"
		}
		full = path.Join(base, target)
	}

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("creating the directory for %s: %w", target, err)
	}
	content := strings.Join(entries, separator) + separator
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil { //nolint:gosec // the tool that reads it may run as another user
		return "", fmt.Errorf("writing the selection to %s: %w", target, err)
	}
	return full, nil
}

// applyEmptySelectionPolicy decides what an empty selection means, and reports
// whether the step is already finished as a result.
//
// The default runs everything, which is what makes `from: self` work: the first
// attempt has no report to narrow against and must run the whole suite. A
// dedicated re-run step usually wants skip instead, since "nothing failed" is
// not a reason to run the suite again.
func applyEmptySelectionPolicy(step *data.StepData, policy string) (finished bool) {
	switch policy {
	case "skip":
		step.SetStatus(constants.StepStatusSkipped)
		orchestration.FinishExecution(step, constants.ExecutionResult{
			Details:   "no test cases were selected, so there was nothing to run",
			Iteration: int(step.Iteration),
		})
		return true
	case "fail":
		step.SetStatus(constants.StepStatusFailed).SetExitCode(1)
		orchestration.FinishExecution(step, constants.ExecutionResult{
			ExitCode:  1,
			Details:   "no test cases were selected, and select.empty is fail",
			Iteration: int(step.Iteration),
		})
		return true
	default:
		return false
	}
}

// testCasesOutcome is what the policy decided, in the three shapes the rest of
// the step needs it: the verdict, a line for a person, and the counters for the
// execution record.
type testCasesOutcome struct {
	Success bool
	Details string
	Results *testkube.TestWorkflowStepTestResults
}

// applyTestCases decides the step's verdict from the report the tool just
// produced, replacing the bare "exit code is zero" reading.
//
// The summary and counters are produced whether the step passed or failed:
// muted must never mean silent, so the numbers are always reported.
//
// This runs after the command and before `negative` is applied - the policy
// decides what "failed" means, and negative inverts that.
func applyTestCases(policy *lite.ActionTestCases, workingDir string, exitCode int, narrowed bool) testCasesOutcome {
	report, found, err := readTestReport(policy.ReportPaths, workingDir)
	if err != nil {
		return testCasesOutcome{Details: fmt.Sprintf("could not read the test report: %s", err)}
	}

	if !found {
		where := strings.Join(policy.ReportPaths, ", ")
		switch policy.OnMissing {
		case "ignore":
			return testCasesOutcome{Success: exitCode == 0}
		case "warn":
			return testCasesOutcome{
				Success: exitCode == 0,
				Details: fmt.Sprintf("no test report found at %s", where),
			}
		default:
			// Failing by default is the safety catch: it stops a mute policy
			// masking a step that crashed before it could write a report.
			return testCasesOutcome{
				Details: fmt.Sprintf("no test report found at %s, and report.onMissing is fail", where),
			}
		}
	}

	verdict, err := testresults.Evaluate(report, testCasesPolicy(policy), testresults.Input{
		ExitCode: exitCode,
		// A narrowed run measured a subset, so the pass requirement is not
		// evaluated against it - a threshold over a subset means nothing.
		Narrowed: narrowed,
	})
	if err != nil {
		return testCasesOutcome{Details: fmt.Sprintf("could not evaluate the test report: %s", err)}
	}

	details := verdict.Describe()
	if unused := verdict.UnusedMutePatterns; len(unused) > 0 {
		// A mute pattern matching nothing is dead quarantine config. Saying so
		// is what stops mute lists outliving the bugs they were written for.
		details += fmt.Sprintf(". Mute patterns matching nothing: %s", strings.Join(unused, ", "))
	}

	return testCasesOutcome{
		Success: verdict.Success,
		Details: details,
		Results: stepTestResults(verdict),
	}
}

// stepTestResults converts the verdict into the execution record's shape.
func stepTestResults(verdict testresults.Verdict) *testkube.TestWorkflowStepTestResults {
	return &testkube.TestWorkflowStepTestResults{
		Tests:                verdict.Summary.Tests,
		Passed:               verdict.Summary.Passed,
		Failed:               verdict.Summary.Failed,
		Errored:              verdict.Summary.Errored,
		Skipped:              verdict.Summary.Skipped,
		Muted:                int32(len(verdict.Muted)),
		Unexpected:           int32(len(verdict.Unexpected)),
		Tolerated:            verdict.Tolerated,
		RequirementApplied:   verdict.ToleranceApplied,
		IdentitiesIncomplete: verdict.IdentitiesIncomplete,
		Unrepresented:        verdict.Unrepresented,
		UnusedMutePatterns:   verdict.UnusedMutePatterns,
	}
}

// testCasesPolicy converts the action payload into the evaluator's policy.
func testCasesPolicy(policy *lite.ActionTestCases) testresults.Policy {
	converted := testresults.Policy{
		Mute:    testresults.Selector{Include: policy.MuteInclude, Exclude: policy.MuteExclude},
		Enforce: testresults.Enforce(policy.Enforce),
	}
	if policy.Tolerate != nil {
		converted.Tolerate = &testresults.Tolerance{
			MaxFailed:        policy.Tolerate.MaxFailed,
			MaxFailedPercent: policy.Tolerate.MaxFailedPercent,
			MinPassed:        policy.Tolerate.MinPassed,
			MinPassedPercent: policy.Tolerate.MinPassedPercent,
		}
	}
	return converted
}

// readTestReport reads every report the patterns match, merged into one.
//
// found is false when the patterns matched nothing, which the caller treats
// according to report.onMissing rather than as an error - a report that is
// legitimately absent is a policy question, not a failure to read.
func readTestReport(patterns []string, workingDir string) (testresults.Report, bool, error) {
	files, err := matchReportFiles(patterns, workingDir)
	if err != nil {
		return testresults.Report{}, false, err
	}
	if len(files) == 0 {
		return testresults.Report{}, false, nil
	}

	reports := make([]testresults.Report, 0, len(files))
	for _, file := range files {
		handle, err := os.Open(file)
		if err != nil {
			return testresults.Report{}, false, fmt.Errorf("opening %s: %w", file, err)
		}
		report, err := testresults.Parse(handle)
		handle.Close()
		if err != nil {
			return testresults.Report{}, false, fmt.Errorf("parsing %s: %w", file, err)
		}
		reports = append(reports, report)
	}
	return testresults.Merge(reports...), true, nil
}

// matchReportFiles expands the patterns into concrete files, sorted so that a
// merged report is built in a stable order regardless of directory iteration.
func matchReportFiles(patterns []string, workingDir string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		root, relative := splitPatternRoot(pattern, workingDir)
		matches, err := doublestar.Glob(os.DirFS(root), relative)
		if err != nil {
			return nil, fmt.Errorf("bad report path %q: %w", pattern, err)
		}
		for _, match := range matches {
			full := path.Join(root, match)
			info, err := fs.Stat(os.DirFS(root), match)
			if err != nil || info.IsDir() {
				continue
			}
			if !seen[full] {
				seen[full] = true
				files = append(files, full)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

// splitPatternRoot decides what the pattern is relative to. An absolute path is
// taken as written; anything else hangs off the container's working directory,
// which is where the tool wrote its report.
func splitPatternRoot(pattern, workingDir string) (root, relative string) {
	pattern = strings.TrimSpace(pattern)
	if strings.HasPrefix(pattern, "/") {
		return "/", strings.TrimPrefix(pattern, "/")
	}
	if workingDir == "" {
		workingDir = "/"
	}
	return workingDir, pattern
}
