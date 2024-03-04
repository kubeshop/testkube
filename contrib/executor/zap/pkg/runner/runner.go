package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

// ZapRunner runs ZAP tests
type ZapRunner struct {
	Params  envs.Params
	ZapHome string
	Scraper scraper.Scraper
}

var _ runner.Runner = &ZapRunner{}

// NewRunner creates a new ZapRunner
func NewRunner(ctx context.Context, params envs.Params) (*ZapRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &ZapRunner{
		Params:  params,
		ZapHome: os.Getenv("ZAP_HOME"),
	}

	output.PrintLogf("%s Preparing scraper", ui.IconTruck)
	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Run executes the test and returns the test results
func (r *ZapRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)

	testFile, _, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		return result, err
	}

	var zapConfig string
	workingDir := r.Params.DataDir
	if fileInfo.IsDir() {
		// assume the ZAP config YAML has been passed as test argument
		zapConfig = filepath.Join(testFile, execution.Args[len(execution.Args)-1])
	} else {
		// use the given file config as ZAP config YAML
		zapConfig = testFile
	}

	// determine the ZAP script
	output.PrintLogf("%s Processing test type", ui.IconWorld)
	scanType := strings.Split(execution.TestType, "/")[1]
	for i := range execution.Command {
		if execution.Command[i] == "<pythonScriptPath>" {
			execution.Command[i] = zapScript(scanType)
		}
	}
	output.PrintLogf("%s Using command: %s", ui.IconCheckMark, strings.Join(execution.Command, " "))

	output.PrintLogf("%s Preparing reports folder", ui.IconFile)
	reportFolder := filepath.Join(r.Params.DataDir, "reports")
	err = os.Mkdir(reportFolder, 0700)
	if err != nil {
		return *result.WithErrors(err), nil
	}
	reportFile := filepath.Join(reportFolder, fmt.Sprintf("%s-report.html", execution.TestName))

	output.PrintLogf("%s Building arguments", ui.IconWorld)
	output.PrintLogf("%s Reading options from file", ui.IconWorld)
	options := Options{}
	err = options.UnmarshalYAML(zapConfig)
	if err != nil {
		return *result.WithErrors(err), nil
	}

	output.PrintLogf("%s Preparing variables", ui.IconWorld)
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	output.PrintLogf("%s Variables are prepared", ui.IconCheckMark)

	args := zapArgs(scanType, options, reportFile)
	output.PrintLogf("%s Reading execution arguments", ui.IconWorld)
	args = MergeArgs(args, reportFile, execution)
	output.PrintLogf("%s Arguments are ready: %s", ui.IconCheckMark, envManager.ObfuscateStringSlice(args))

	// when using file based ZAP parameters it expects a /zap/wrk directory
	// we simply symlink the directory
	os.Symlink(workingDir, filepath.Join(r.ZapHome, "wrk"))

	output.PrintLogf("%s Running ZAP test", ui.IconMicroscope)
	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	logs, err := executor.Run(r.ZapHome, command, envManager, args...)
	logs = envManager.ObfuscateSecrets(logs)

	output.PrintLogf("%s Calculating results", ui.IconMicroscope)
	if err == nil {
		result.Status = testkube.ExecutionStatusPassed
	} else {
		result.Status = testkube.ExecutionStatusFailed
		result.ErrorMessage = err.Error()
		if strings.Contains(result.ErrorMessage, "exit status 1") || strings.Contains(result.ErrorMessage, "exit status 2") {
			result.ErrorMessage = "security issues found during scan"
		} else {
			// ZAP was unable to run at all, wrong args?
			return result, nil
		}
	}

	result.Output = string(logs)
	result.OutputType = "text/plain"

	// prepare step results based on output
	result.Steps = []testkube.ExecutionStepResult{}
	lines := strings.Split(result.Output, "\n")
	for _, line := range lines {
		if strings.Index(line, "PASS") == 0 || strings.Index(line, "INFO") == 0 {
			result.Steps = append(result.Steps, testkube.ExecutionStepResult{
				Name: stepName(line),
				// always success
				Status: string(testkube.PASSED_ExecutionStatus),
			})
		} else if strings.Index(line, "WARN") == 0 {
			result.Steps = append(result.Steps, testkube.ExecutionStepResult{
				Name: stepName(line),
				// depends on the options if WARN will fail or not
				Status: warnStatus(scanType, options),
			})
		} else if strings.Index(line, "FAIL") == 0 {
			result.Steps = append(result.Steps, testkube.ExecutionStepResult{
				Name: stepName(line),
				// always error
				Status: string(testkube.FAILED_ExecutionStatus),
			})
		}
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	if r.Params.ScrapperEnabled {
		directories := []string{reportFolder}
		var masks []string
		if execution.ArtifactRequest != nil {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
			masks = execution.ArtifactRequest.Masks
		}

		output.PrintLogf("%s Scraping directories: %v with masks: %v", ui.IconCabinet, directories, masks)

		if err := r.Scraper.Scrape(ctx, directories, masks, execution); err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts from ZAP executor")
		}
	}

	if rerr != nil {
		return *result.Err(rerr), nil
	}

	return result, err
}

// GetType returns runner type
func (r *ZapRunner) GetType() runner.Type {
	return runner.TypeMain
}

const API = "api"
const BASELINE = "baseline"
const FULL = "full"

func zapScript(scanType string) string {
	switch {
	case scanType == BASELINE:
		return "./zap-baseline.py"
	default:
		return fmt.Sprintf("./zap-%s-scan.py", scanType)
	}
}

func zapArgs(scanType string, options Options, reportFile string) (args []string) {
	switch {
	case scanType == API:
		args = options.ToApiScanArgs(reportFile)
	case scanType == BASELINE:
		args = options.ToBaselineScanArgs(reportFile)
	case scanType == FULL:
		args = options.ToFullScanArgs(reportFile)
	}
	return args
}

func stepName(line string) string {
	return strings.TrimSpace(strings.SplitAfter(line, ":")[1])
}

func warnStatus(scanType string, options Options) string {
	var fail bool

	switch {
	case scanType == API:
		fail = options.API.FailOnWarn
	case scanType == BASELINE:
		fail = options.Baseline.FailOnWarn
	case scanType == FULL:
		fail = options.Full.FailOnWarn
	}

	if fail {
		return string(testkube.FAILED_ExecutionStatus)
	} else {
		return string(testkube.PASSED_ExecutionStatus)
	}
}

// MergeArgs merges the arguments read from file with the arguments read from the execution
func MergeArgs(fileArgs []string, reportFile string, execution testkube.Execution) []string {
	output.PrintLogf("%s Merging file arguments with execution arguments", ui.IconWorld)

	args := execution.Args
	for i := range args {
		if args[i] == "<fileArgs>" {
			newArgs := make([]string, len(args)+len(fileArgs)-1)
			copy(newArgs, args[:i])
			copy(newArgs[i:], fileArgs)
			copy(newArgs[i+len(fileArgs):], args[i+1:])
			args = newArgs
			break
		}
	}

	for i := range args {
		args[i] = os.ExpandEnv(args[i])
	}

	return args
}
