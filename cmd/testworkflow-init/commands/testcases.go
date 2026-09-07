package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/kubeshop/testkube/pkg/testresults"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// applyTestCases decides the step's verdict from the report the tool just
// produced, replacing the bare "exit code is zero" reading.
//
// It returns the verdict and a one-line summary for the step's error message,
// which reaches the execution record through ExecutionResult.Details. The
// summary is returned whether the step passed or failed: muted must never mean
// silent, so the numbers are always reported.
//
// This runs after the command and before `negative` is applied - the policy
// decides what "failed" means, and negative inverts that.
func applyTestCases(policy *lite.ActionTestCases, workingDir string, exitCode int) (bool, string) {
	report, found, err := readTestReport(policy.ReportPaths, workingDir)
	if err != nil {
		return false, fmt.Sprintf("could not read the test report: %s", err)
	}

	if !found {
		where := strings.Join(policy.ReportPaths, ", ")
		switch policy.OnMissing {
		case "ignore":
			return exitCode == 0, ""
		case "warn":
			return exitCode == 0, fmt.Sprintf("no test report found at %s", where)
		default:
			// Failing by default is the safety catch: it stops a mute policy
			// masking a step that crashed before it could write a report.
			return false, fmt.Sprintf("no test report found at %s, and report.onMissing is fail", where)
		}
	}

	verdict, err := testresults.Evaluate(report, testCasesPolicy(policy), testresults.Input{
		ExitCode: exitCode,
		// Narrowing is not wired up yet, so every run measures the whole suite
		// and the pass requirement always applies.
		Narrowed: false,
	})
	if err != nil {
		return false, fmt.Sprintf("could not evaluate the test report: %s", err)
	}

	details := verdict.Describe()
	if unused := verdict.UnusedMutePatterns; len(unused) > 0 {
		// A mute pattern matching nothing is dead quarantine config. Saying so
		// is what stops mute lists outliving the bugs they were written for.
		details += fmt.Sprintf(". Mute patterns matching nothing: %s", strings.Join(unused, ", "))
	}
	return verdict.Success, details
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
