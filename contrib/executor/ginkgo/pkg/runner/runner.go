package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	junit "github.com/joshdk/go-junit"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/ui"
)

var ginkgoDefaultParams = InitializeGinkgoParams()
var ginkgoBin = "ginkgo"

func NewGinkgoRunner() (*GinkgoRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Ginkgo runner variables: %w", err)
	}

	runner := &GinkgoRunner{
		Fetcher: content.NewFetcher(""),
		Scraper: scraper.NewMinioScraper(
			params.Endpoint,
			params.AccessKeyID,
			params.SecretAccessKey,
			params.Location,
			params.Region,
			params.Token,
			params.Bucket,
			params.Ssl,
		),
		Params: params,
	}

	return runner, nil
}

type GinkgoRunner struct {
	Params  envs.Params
	Fetcher content.ContentFetcher
	Scraper scraper.Scraper
}

func (r *GinkgoRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	// Set github user and token params in Content.Repository
	if r.Params.GitUsername != "" || r.Params.GitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = r.Params.GitUsername
			execution.Content.Repository.Token = r.Params.GitToken
		}
	}

	// use `execution.Variables` for variables passed from Test/Execution
	// variables of type "secret" will be automatically decoded
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return result, err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		output.PrintLog(fmt.Sprintf("%s passing ginkgo test as single file not implemented yet", ui.IconCross))
		return result, fmt.Errorf("passing ginkgo test as single file not implemented yet")
	}

	// Set up ginkgo params
	ginkgoParams := FindGinkgoParams(&execution, ginkgoDefaultParams)

	runPath := path
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
		path = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)
	}

	// Set up ginkgo potential args
	ginkgoArgs, err := BuildGinkgoArgs(ginkgoParams, path, runPath)
	if err != nil {
		return result, err
	}
	ginkgoPassThroughFlags := BuildGinkgoPassThroughFlags(execution)
	ginkgoArgsAndFlags := append(ginkgoArgs, ginkgoPassThroughFlags...)

	// set up reports directory
	reportsPath := filepath.Join(path, "reports")
	if _, err := os.Stat(reportsPath); os.IsNotExist(err) {
		mkdirErr := os.Mkdir(reportsPath, os.ModePerm)
		if mkdirErr != nil {
			output.PrintLog(fmt.Sprintf("%s could not set up reports directory: %s", ui.IconCross, mkdirErr.Error()))
			return result, mkdirErr
		}
	}

	// run executor here
	out, err := executor.Run(runPath, ginkgoBin, envManager, ginkgoArgsAndFlags...)
	out = envManager.ObfuscateSecrets(out)

	// generate report/result
	if ginkgoParams["GinkgoJsonReport"] != "" {
		moveErr := MoveReport(runPath, reportsPath, strings.Split(ginkgoParams["GinkgoJsonReport"], " ")[1])
		if moveErr != nil {
			output.PrintLog(fmt.Sprintf("%s could not move JSON report: %s", ui.IconCross, moveErr.Error()))
			return result, moveErr
		}
	}
	if ginkgoParams["GinkgoJunitReport"] != "" {
		moveErr := MoveReport(runPath, reportsPath, strings.Split(ginkgoParams["GinkgoJunitReport"], " ")[1])
		if moveErr != nil {
			output.PrintLog(fmt.Sprintf("%s could not move Junit report: %s", ui.IconCross, moveErr.Error()))
			return result, moveErr
		}
	}
	if ginkgoParams["GinkgoTeamCityReport"] != "" {
		moveErr := MoveReport(runPath, reportsPath, strings.Split(ginkgoParams["GinkgoTeamCityReport"], " ")[1])
		if moveErr != nil {
			output.PrintLog(fmt.Sprintf("%s could not move TeamCity report: %s", ui.IconCross, moveErr.Error()))
			return result, moveErr
		}
	}
	suites, serr := junit.IngestFile(filepath.Join(reportsPath, strings.Split(ginkgoParams["GinkgoJunitReport"], " ")[1]))
	result = MapJunitToExecutionResults(out, suites)
	output.PrintLog(fmt.Sprintf("%s Mapped Junit to Execution Results...", ui.IconCheckMark))

	// scrape artifacts first even if there are errors above

	if r.Params.ScrapperEnabled {
		directories := []string{
			reportsPath,
		}
		err := r.Scraper.Scrape(execution.Id, directories)
		if err != nil {
			return *result.WithErrors(fmt.Errorf("scrape artifacts error: %w", err)), nil
		}
	}

	return *result.WithErrors(err, serr), nil
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
	output.PrintLog(fmt.Sprintf("%s Preparing initial Ginkgo parameters", ui.IconWorld))

	ginkgoParams := make(map[string]string)
	ginkgoParams["GinkgoTestPackage"] = ""
	ginkgoParams["GinkgoRecursive"] = "-r"                          // -r
	ginkgoParams["GinkgoParallel"] = "-p"                           // -p
	ginkgoParams["GinkgoParallelProcs"] = ""                        // --procs N
	ginkgoParams["GinkgoCompilers"] = ""                            // --compilers N
	ginkgoParams["GinkgoRandomize"] = "--randomize-all"             // --randomize-all
	ginkgoParams["GinkgoRandomizeSuites"] = "--randomize-suites"    // --randomize-suites
	ginkgoParams["GinkgoLabelFilter"] = ""                          // --label-filter QUERY
	ginkgoParams["GinkgoFocusFilter"] = ""                          // --focus REGEXP
	ginkgoParams["GinkgoSkipFilter"] = ""                           // --skip REGEXP
	ginkgoParams["GinkgoUntilItFails"] = ""                         // --until-it-fails
	ginkgoParams["GinkgoRepeat"] = ""                               // --repeat N
	ginkgoParams["GinkgoFlakeAttempts"] = ""                        // --flake-attempts N
	ginkgoParams["GinkgoTimeout"] = ""                              // --timeout=duration
	ginkgoParams["GinkgoSkipPackage"] = ""                          // --skip-package list,of,packages
	ginkgoParams["GinkgoFailFast"] = ""                             // --fail-fast
	ginkgoParams["GinkgoKeepGoing"] = "--keep-going"                // --keep-going
	ginkgoParams["GinkgoFailOnPending"] = ""                        // --fail-on-pending
	ginkgoParams["GinkgoCover"] = ""                                // --cover
	ginkgoParams["GinkgoCoverProfile"] = ""                         // --coverprofile cover.profile
	ginkgoParams["GinkgoRace"] = ""                                 // --race
	ginkgoParams["GinkgoTrace"] = "--trace"                         // --trace
	ginkgoParams["GinkgoJsonReport"] = ""                           // --json-report report.json [will be stored in reports/filename]
	ginkgoParams["GinkgoJunitReport"] = "--junit-report report.xml" // --junit-report report.xml [will be stored in reports/filename]
	ginkgoParams["GinkgoTeamCityReport"] = ""                       // --teamcity-report report.teamcity [will be stored in reports/filename]

	output.PrintLog(fmt.Sprintf("%s Initial Ginkgo parameters prepared: %s", ui.IconCheckMark, ginkgoParams))
	return ginkgoParams
}

// Find any GinkgoParams in execution.Variables
func FindGinkgoParams(execution *testkube.Execution, defaultParams map[string]string) map[string]string {
	output.PrintLog(fmt.Sprintf("%s Setting Ginkgo parameters from variables", ui.IconWorld))

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

	output.PrintLog(fmt.Sprintf("%s Ginkgo parameters from variables set: %s", ui.IconCheckMark, retVal))
	return retVal
}

func BuildGinkgoArgs(params map[string]string, path, runPath string) ([]string, error) {
	output.PrintLog(fmt.Sprintf("%s Building Ginkgo arguments from params", ui.IconWorld))

	args := []string{}
	for k, p := range params {
		if p == "" {
			continue
		}
		if k != "GinkgoTestPackage" {
			args = append(args, strings.Split(p, " ")...)
		}
	}

	if params["GinkgoTestPackage"] != "" {
		if path != runPath {
			args = append(args, filepath.Join(path, params["GinkgoTestPackage"]))
		} else {
			args = append(args, params["GinkgoTestPackage"])
		}
	} else {
		if path != runPath {
			args = append(args, path)
		}
	}

	output.PrintLog(fmt.Sprintf("%s Ginkgo arguments from params built: %s", ui.IconCheckMark, args))
	return args, nil
}

// This should always be called after FindGinkgoParams so that it only
// acts on the "left over" Variables that are to be treated as pass through
// flags to GInkgo
func BuildGinkgoPassThroughFlags(execution testkube.Execution) []string {
	output.PrintLog(fmt.Sprintf("%s Building Ginkgo flags", ui.IconWorld))

	vars := execution.Variables
	args := execution.Args
	flags := []string{}
	for _, v := range vars {
		os.Setenv(v.Name, v.Value)
	}

	if len(args) > 0 {
		flags = append(flags, args...)
	}

	if len(flags) > 0 {
		flags = append([]string{"--"}, flags...)
	}

	output.PrintLog(fmt.Sprintf("%s Ginkgo flags built: %s", ui.IconCheckMark, flags))
	return flags
}

// Validate checks if Execution has valid data in context of Ginkgo executor
func (r *GinkgoRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		output.PrintLog(fmt.Sprintf("%s Can't find any content to run in execution data", ui.IconCross))
		return fmt.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	if execution.Content.Repository == nil {
		output.PrintLog(fmt.Sprintf("%s Ginkgo executor handles only repository based tests, but repository is nil", ui.IconCross))
		return fmt.Errorf("ginkgo executor handles only repository based tests, but repository is nil")
	}

	if execution.Content.Repository.Branch == "" && execution.Content.Repository.Commit == "" {
		output.PrintLog(fmt.Sprintf("%s Can't find branch or commit in params must use one or the other, repo %+v", ui.IconCross, execution.Content.Repository))
		return fmt.Errorf("can't find branch or commit in params must use one or the other, repo:%+v", execution.Content.Repository)
	}

	return nil
}

func MapJunitToExecutionResults(out []byte, suites []junit.Suite) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status
	result.Output = string(out)
	result.OutputType = "text/plain"
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
