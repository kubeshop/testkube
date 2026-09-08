package commands

import (
	"context"
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
	"github.com/kubeshop/testkube/pkg/executiondata"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testresults"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// Environment variables carrying the selection, for a shell that would rather
// not write an expression.
const (
	// EnvSelectedTests carries the selected entries, one per line.
	EnvSelectedTests = "TK_SELECTED_TESTS"
	// EnvSelectedTestsFile carries the path select.write produced, empty when
	// it was not asked for. Kept apart from EnvSelectedTests, which used to
	// hold this and so was empty for every selection that wanted no file.
	EnvSelectedTestsFile  = "TK_SELECTED_TESTS_FILE"
	EnvSelectedTestsCount = "TK_SELECTED_TESTS_COUNT"
)

// testCasesSelection is the resolved selection for one run of a step.
type testCasesSelection struct {
	// Entries are what the tool should be invoked with, already projected.
	// Always non-nil in intent: empty means "run everything".
	Entries []string
	// Addresses are the canonical ids behind Entries, kept so the verdict can
	// check that the cases this run was narrowed to actually ran. Empty for a
	// selection that came from an explicit list rather than a report, which has
	// no address to check against.
	Addresses []string
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
func resolveTestCaseSelection(ctx context.Context, policy *lite.ActionTestCases, workingDir string, rerun *testworkflowconfig.RerunConfig, source reportSource) (*testCasesSelection, error) {
	selection := &testCasesSelection{}
	if policy.Select == nil {
		return selection, nil
	}

	// The selection reads its own paths when given some, which is how a step
	// re-runs what an *earlier* step failed: the verdict still judges this
	// step's report, while the selection draws from the other one. Defaulting
	// to the step's own report is `from: self` - the narrowing retry.
	sourcePaths := policy.Select.Paths
	if len(sourcePaths) == 0 {
		sourcePaths = policy.ReportPaths
	}

	report, found, err := readSelectionSource(ctx, policy, sourcePaths, workingDir, rerun, source)
	if err != nil {
		return nil, err
	}

	if found {
		statuses, err := testresults.ParseStatuses(policy.Select.Status)
		if err != nil {
			return nil, err
		}
		resolver := testresults.Selection{
			Statuses:     statuses,
			Filter:       testresults.Selector{Include: policy.Select.Include, Exclude: policy.Select.Exclude},
			Mute:         testresults.Selector{Include: policy.MuteInclude, Exclude: policy.MuteExclude},
			IncludeMuted: policy.Select.IncludeMuted,
			Cases:        policy.Select.Cases,
			As:           policy.Select.As,
			Collapse:     policy.Select.Collapse,
		}
		entries, err := resolver.Resolve(report)
		if err != nil {
			return nil, err
		}
		selection.Entries = entries

		// Kept alongside the projected entries so the verdict can tell a
		// narrowed run that passed from one that tested nothing.
		addresses, err := resolver.Addresses(report)
		if err != nil {
			return nil, err
		}
		selection.Addresses = addresses
	} else if len(policy.Select.Cases) > 0 {
		// An explicit list does not need a report to have existed.
		selection.Entries = append(selection.Entries, policy.Select.Cases...)
	}

	// A selection may also arrive with the execution, from an API or CLI caller
	// who named the test cases rather than the workflow declaring them. It joins
	// verbatim for the same reason select.cases does: the caller has already
	// written the names in the shape their tool wants.
	selection.Entries = append(selection.Entries, rerunTestCases(rerun)...)

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

// readSelectionSource reads the previous results the selection draws from, which
// is either this pod's own file system or another execution's artifacts.
//
// The local read comes first for `self`, and a step left on `self` falls back to
// the execution's rerun policy when it finds nothing. That fallback is what makes
// `testkube rerun --only-failed` work on a workflow written for a narrowing
// retry: the first attempt has no report of its own, so it takes the failures of
// the execution being rerun, and every later attempt prefers the report it just
// wrote. The two orderings can never conflict, so neither has to be configured.
func readSelectionSource(
	ctx context.Context,
	policy *lite.ActionTestCases,
	sourcePaths []string,
	workingDir string,
	rerun *testworkflowconfig.RerunConfig,
	source reportSource,
) (testresults.Report, bool, error) {
	from := policy.Select.From
	if from != "" && from != selectFromSelf {
		report, found, err := source.readRemoteReport(ctx, from, policy.Select.Paths)
		if err != nil {
			return testresults.Report{}, false, fmt.Errorf("reading the report of %q: %w", from, err)
		}
		return report, found, nil
	}

	report, _, found, err := readTestReport(sourcePaths, workingDir)
	if err != nil {
		return testresults.Report{}, false, fmt.Errorf("reading the previous report: %w", err)
	}
	if found {
		return report, true, nil
	}

	if !seedsFromRerun(rerun) {
		return report, false, nil
	}
	// The rerun names an execution but not where its report is, so the step's own
	// report paths are the only thing that can: the same workflow produced both.
	remote, found, err := source.readRemoteReport(ctx, executiondata.RerunRef, policy.ReportPaths)
	if err != nil {
		return testresults.Report{}, false, fmt.Errorf("reading the report of the execution being rerun: %w", err)
	}
	return remote, found, nil
}

// selectFromSelf is the default source: this execution, read off the disk.
const selectFromSelf = "self"

// seedsFromRerun reports whether the execution was narrowed to the failures of
// another one, which is what --only-failed asks for.
//
// An explicit test case list does not seed anything, even alongside onlyFailed:
// it already *is* the selection, named by the caller in their tool's own shape,
// so there is nothing to derive from a report and no reason to fetch one.
func seedsFromRerun(rerun *testworkflowconfig.RerunConfig) bool {
	return rerun != nil && rerun.OnlyFailed && rerun.ExecutionId != "" && len(rerun.TestCases) == 0
}

// rerunTestCases are the test cases the execution itself was narrowed to, named
// by whoever scheduled it rather than by the workflow.
//
// These join verbatim, without a report: the caller has already written them in
// the shape their tool wants. The other half of the rerun policy - narrowing to
// whatever failed - is served by readSelectionSource.
func rerunTestCases(rerun *testworkflowconfig.RerunConfig) []string {
	if rerun == nil {
		return nil
	}
	return rerun.TestCases
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
//
// The entries are newline-separated rather than delimited by anything else: a
// test case name may contain a space or a comma - parameterized names routinely
// do - so any other separator would split names in half. A shell reads it with
// `while read`, or `xargs -d '\n'`.
func (s *testCasesSelection) Export() error {
	if err := os.Setenv(EnvSelectedTests, strings.Join(s.Entries, "\n")); err != nil {
		return err
	}
	if err := os.Setenv(EnvSelectedTestsFile, s.File); err != nil {
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
func applyTestCases(ref string, policy *lite.ActionTestCases, workingDir string, exitCode int, selection *testCasesSelection) testCasesOutcome {
	report, files, found, err := readTestReport(policy.ReportPaths, workingDir)
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
		Narrowed: selection.Narrowed,
		Selected: selection.Addresses,
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
	if missing := verdict.SelectedMissing; len(missing) > 0 {
		// Naming a few beats the count alone: the usual cause is the runner
		// spelling a test differently than the report does, which you can only
		// see by reading one.
		details += fmt.Sprintf(". Selected but absent from the report: %s", summarizeList(missing, maxNamedMissing))
	}

	// Hand the verdict to the container that will upload these reports. It can
	// read which cases failed out of the XML, but not which failures this step
	// declared acceptable. Losing the handoff costs the muted counts on the
	// uploaded report, not the step's status, which is already decided.
	if err := recordVerdict(ref, files, verdict); err != nil {
		fmt.Printf("warn: could not record the test case verdict for the report upload: %s\n", err)
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
// readTestReport parses the reports the patterns name, returning their merge
// along with the files it read. The file list is what lets the verdict be handed
// to the container that uploads those same files.
func readTestReport(patterns []string, workingDir string) (testresults.Report, []string, bool, error) {
	files, err := matchReportFiles(patterns, workingDir)
	if err != nil {
		return testresults.Report{}, nil, false, err
	}
	if len(files) == 0 {
		return testresults.Report{}, nil, false, nil
	}

	reports := make([]testresults.Report, 0, len(files))
	for _, file := range files {
		handle, err := os.Open(file)
		if err != nil {
			return testresults.Report{}, nil, false, fmt.Errorf("opening %s: %w", file, err)
		}
		report, err := testresults.Parse(handle)
		handle.Close()
		if err != nil {
			return testresults.Report{}, nil, false, fmt.Errorf("parsing %s: %w", file, err)
		}
		reports = append(reports, report)
	}
	return testresults.Merge(reports...), files, true, nil
}

// recordVerdict leaves the muted decision where the artifacts container can find
// it, keyed by the report files it was read from.
func recordVerdict(ref string, files []string, verdict testresults.Verdict) error {
	muted := make([]string, 0, len(verdict.Muted))
	for _, testCase := range verdict.Muted {
		muted = append(muted, testCase.ID())
	}

	absolute := make([]string, 0, len(files))
	for _, file := range files {
		// The uploader resolves its own paths against the artifacts root, so both
		// sides have to be absolute for the join to land.
		path, err := filepath.Abs(file)
		if err != nil {
			path = file
		}
		absolute = append(absolute, path)
	}

	return testresults.WriteVerdict(ref, testresults.ReportVerdict{
		Reports:    absolute,
		Muted:      muted,
		Unexpected: int32(len(verdict.Unexpected)),
		Tolerated:  verdict.Tolerated,
	})
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

// maxNamedMissing caps how many absent test cases the step log names. A
// selection can be thousands of entries and every one of them can be missing;
// the point of the line is to show what the mismatch looks like.
const maxNamedMissing = 10

// summarizeList renders at most limit entries, saying how many were left out.
func summarizeList(entries []string, limit int) string {
	if len(entries) <= limit {
		return strings.Join(entries, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(entries[:limit], ", "), len(entries)-limit)
}
