package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshdk/go-junit"
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

var ginkgoDefaultParams = InitializeGinkgoParams()

func NewGinkgoRunner(ctx context.Context, params envs.Params) (*GinkgoRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &GinkgoRunner{
		Params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

type GinkgoRunner struct {
	Params  envs.Params
	Scraper scraper.Scraper
}

var _ runner.Runner = &GinkgoRunner{}

func (r *GinkgoRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	// use `execution.Variables` for variables passed from Test/Execution
	// variables of type "secret" will be automatically decoded
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	path, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		output.PrintLogf("%s passing ginkgo test as single file not implemented yet", ui.IconCross)
		return result, errors.Errorf("passing ginkgo test as single file not implemented yet")
	}

	// Set up ginkgo params
	ginkgoParams := FindGinkgoParams(&execution, ginkgoDefaultParams)

	runPath := path
	if workingDir != "" {
		runPath = workingDir
		path = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)
	}

	reportFile := "report.xml"
	if ginkgoParams["GinkgoJunitReport"] != "" {
		values := strings.Split(ginkgoParams["GinkgoJunitReport"], " ")
		if len(values) > 1 {
			reportFile = values[1]
		}
	}

	// Set up ginkgo potential args
	ginkgoArgs, junitReport, err := BuildGinkgoArgs(envManager, ginkgoParams, path, runPath, reportFile, execution)
	if err != nil {
		return result, err
	}

	// set up reports directory
	reportsPath := filepath.Join(path, "reports")
	if _, err := os.Stat(reportsPath); os.IsNotExist(err) {
		mkdirErr := os.Mkdir(reportsPath, os.ModePerm)
		if mkdirErr != nil {
			output.PrintLogf("%s could not set up reports directory: %s", ui.IconCross, mkdirErr.Error())
			return result, mkdirErr
		}
	}

	// check Ginkgo version
	output.PrintLogf("%s Checking Ginkgo CLI version", ui.IconTruck)
	command, args := executor.MergeCommandAndArgs(execution.Command, []string{"version"})
	_, err = executor.Run(runPath, command, envManager, args...)
	if err != nil {
		output.PrintLogf("%s error checking Ginkgo CLI version: %s", ui.IconCross, err.Error())
		return result, err
	}

	// run executor here
	command, args = executor.MergeCommandAndArgs(execution.Command, ginkgoArgs)
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	out, err := executor.Run(runPath, command, envManager, args...)
	out = envManager.ObfuscateSecrets(out)

	// generate report/result
	if ginkgoParams["GinkgoJsonReport"] != "" {
		moveErr := MoveReport(runPath, reportsPath, strings.Split(ginkgoParams["GinkgoJsonReport"], " ")[1])
		if moveErr != nil {
			output.PrintLogf("%s could not move JSON report: %s", ui.IconCross, moveErr.Error())
			return result, moveErr
		}
	}

	if ginkgoParams["GinkgoTeamCityReport"] != "" {
		moveErr := MoveReport(runPath, reportsPath, strings.Split(ginkgoParams["GinkgoTeamCityReport"], " ")[1])
		if moveErr != nil {
			output.PrintLogf("%s could not move TeamCity report: %s", ui.IconCross, moveErr.Error())
			return result, moveErr
		}
	}

	var serr error
	if junitReport {
		moveErr := MoveReport(runPath, reportsPath, reportFile)
		if moveErr != nil {
			output.PrintLogf("%s could not move Junit report: %s", ui.IconCross, moveErr.Error())
		}

		var suites []junit.Suite
		suites, serr = junit.IngestFile(filepath.Join(reportsPath, reportFile))
		if serr == nil {
			result = MapJunitToExecutionResults(out, suites)
			output.PrintLogf("%s Mapped Junit to Execution Results...", ui.IconCheckMark)
		}
	} else {
		result = makeSuccessExecution(out)
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled {
		directories := []string{
			reportsPath,
		}

		var masks []string
		if execution.ArtifactRequest != nil {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
			masks = execution.ArtifactRequest.Masks
		}

		if err := r.Scraper.Scrape(ctx, directories, masks, execution); err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts for Ginkgo executor")
		}
	}

	return *result.WithErrors(err, serr, rerr), nil
}

func MoveReport(path string, reportsPath string, reportFileName string) error {
	oldpath := filepath.Join(path, reportFileName)
	newpath := filepath.Join(reportsPath, reportFileName)
	err := os.Rename(oldpath, newpath)
	if err != nil {
		return err
	}
	return nil
}

func InitializeGinkgoParams() map[string]string {
	output.PrintLogf("%s Preparing initial Ginkgo parameters", ui.IconWorld)

	ginkgoParams := make(map[string]string)
	ginkgoParams["GinkgoTestPackage"] = ""
	ginkgoParams["GinkgoRecursive"] = ""       // -r
	ginkgoParams["GinkgoParallel"] = ""        // -p
	ginkgoParams["GinkgoParallelProcs"] = ""   // --procs N
	ginkgoParams["GinkgoCompilers"] = ""       // --compilers N
	ginkgoParams["GinkgoRandomize"] = ""       // --randomize-all
	ginkgoParams["GinkgoRandomizeSuites"] = "" // --randomize-suites
	ginkgoParams["GinkgoLabelFilter"] = ""     // --label-filter QUERY
	ginkgoParams["GinkgoFocusFilter"] = ""     // --focus REGEXP
	ginkgoParams["GinkgoSkipFilter"] = ""      // --skip REGEXP
	ginkgoParams["GinkgoUntilItFails"] = ""    // --until-it-fails
	ginkgoParams["GinkgoRepeat"] = ""          // --repeat N
	ginkgoParams["GinkgoFlakeAttempts"] = ""   // --flake-attempts N
	ginkgoParams["GinkgoTimeout"] = ""         // --timeout=duration
	ginkgoParams["GinkgoSkipPackage"] = ""     // --skip-package list,of,packages
	ginkgoParams["GinkgoFailFast"] = ""        // --fail-fast
	ginkgoParams["GinkgoKeepGoing"] = ""       // --keep-going
	ginkgoParams["GinkgoFailOnPending"] = ""   // --fail-on-pending
	ginkgoParams["GinkgoCover"] = ""           // --cover
	ginkgoParams["GinkgoCoverProfile"] = ""    // --coverprofile cover.profile
	ginkgoParams["GinkgoRace"] = ""            // --race
	ginkgoParams["GinkgoTrace"] = ""           // --trace
	ginkgoParams["GinkgoJsonReport"] = ""      // --json-report report.json [will be stored in reports/filename]
	ginkgoParams["GinkgoJunitReport"] = ""     // --junit-report report.xml [will be stored in reports/filename]
	ginkgoParams["GinkgoTeamCityReport"] = ""  // --teamcity-report report.teamcity [will be stored in reports/filename]

	output.PrintLogf("%s Initial Ginkgo parameters prepared: %s", ui.IconCheckMark, ginkgoParams)
	return ginkgoParams
}

// FindGinkgoParams finds any GinkgoParams in execution.Variables
func FindGinkgoParams(execution *testkube.Execution, defaultParams map[string]string) map[string]string {
	output.PrintLogf("%s Setting Ginkgo parameters from variables", ui.IconWorld)

	var retVal = make(map[string]string)
	for k, p := range defaultParams {
		v, found := execution.Variables[k]
		if found {
			retVal[k] = v.Value
			delete(execution.Variables, k)
		} else {
			if p != "" {
				retVal[k] = p
			}
		}
	}

	output.PrintLogf("%s Ginkgo parameters from variables set: %s", ui.IconCheckMark, retVal)
	return retVal
}

func BuildGinkgoArgs(envManager *env.Manager, params map[string]string, path, runPath, reportFile string, execution testkube.Execution) ([]string, bool, error) {
	output.PrintLogf("%s Building Ginkgo arguments from params", ui.IconWorld)

	args := execution.Args
	for i := range args {
		if args[i] == "<envVars>" {
			var envVars []string
			for k, p := range params {
				if p == "" {
					continue
				}
				if k != "GinkgoTestPackage" {
					envVars = append(envVars, strings.Split(p, " ")...)
				}
			}

			newArgs := make([]string, len(args)+len(envVars)-1)
			copy(newArgs, args[:i])
			copy(newArgs[i:], envVars)
			copy(newArgs[i+len(envVars):], args[i+1:])
			args = newArgs
			break
		}
	}

	var rp string
	if params["GinkgoTestPackage"] != "" {
		if path != runPath {
			rp = filepath.Join(path, params["GinkgoTestPackage"])
		} else {
			rp = params["GinkgoTestPackage"]
		}
	} else {
		if path != runPath {
			rp = path
		}
	}

	hasJunit := false
	hasReport := false
	for i := len(args) - 1; i >= 0; i-- {
		if rp == "" && args[i] == "<runPath>" {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<runPath>" {
			args[i] = rp
		}

		if args[i] == "<reportFile>" {
			args[i] = reportFile
			hasReport = true
		}

		if args[i] == "--junit-report" {
			hasJunit = true
		}

		args[i] = os.ExpandEnv(args[i])
	}

	output.PrintLogf("%s Ginkgo arguments from params built: %s", ui.IconCheckMark, envManager.ObfuscateStringSlice(args))
	return args, hasJunit && hasReport, nil
}

// Validate checks if Execution has valid data in context of Ginkgo executor
func (r *GinkgoRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		output.PrintLogf("%s Can't find any content to run in execution data", ui.IconCross)
		return fmt.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	if execution.Content.Repository == nil {
		output.PrintLogf("%s Ginkgo executor handles only repository based tests, but repository is nil", ui.IconCross)
		return errors.New("ginkgo executor handles only repository based tests, but repository is nil")
	}

	if execution.Content.Repository.Branch == "" && execution.Content.Repository.Commit == "" {
		output.PrintLogf("%s Can't find branch or commit in params must use one or the other, repo %+v", ui.IconCross, execution.Content.Repository)
		return fmt.Errorf("can't find branch or commit in params must use one or the other, repo: %+v", execution.Content.Repository)
	}

	return nil
}

func MapJunitToExecutionResults(out []byte, suites []junit.Suite) (result testkube.ExecutionResult) {
	result = makeSuccessExecution(out)

	overallStatusFailed := false
	for _, suite := range suites {
		for _, test := range suite.Tests {
			result.Steps = append(
				result.Steps,
				testkube.ExecutionStepResult{
					Name:     fmt.Sprintf("%s - %s", suite.Name, test.Name),
					Duration: test.Duration.String(),
					Status:   MapStatus(test.Status),
				})
			if test.Status == junit.Status(testkube.FAILED_ExecutionStatus) {
				overallStatusFailed = true
			}
		}

		// TODO parse sub suites recursively

	}
	if overallStatusFailed {
		result.Status = testkube.ExecutionStatusFailed
	} else {
		result.Status = testkube.ExecutionStatusPassed
	}
	return result
}

func MapStatus(in junit.Status) (out string) {
	switch string(in) {
	case "passed":
		return string(testkube.PASSED_ExecutionStatus)
	default:
		return string(testkube.FAILED_ExecutionStatus)
	}
}

// GetType returns runner type
func (r *GinkgoRunner) GetType() runner.Type {
	return runner.TypeMain
}

func makeSuccessExecution(out []byte) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status
	result.Output = string(out)
	result.OutputType = "text/plain"

	return result
}
