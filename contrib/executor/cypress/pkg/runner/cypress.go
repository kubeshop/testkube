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

func NewCypressRunner(ctx context.Context, dependency string, params envs.Params) (*CypressRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &CypressRunner{
		Params:     params,
		dependency: dependency,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// CypressRunner - implements runner interface used in worker to start test execution
type CypressRunner struct {
	Params     envs.Params
	Scraper    scraper.Scraper
	dependency string
}

func (r *CypressRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)
	// make some validation
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	output.PrintLogf("%s Checking test content from %s...", ui.IconBox, execution.Content.Type_)

	runPath, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	projectPath := runPath
	if workingDir != "" {
		runPath = workingDir
	}

	fileInfo, err := os.Stat(projectPath)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		output.PrintLogf("%s Using file...", ui.IconTruck)

		// TODO add cypress project structure
		// TODO checkout this repo with `skeleton` path
		// TODO overwrite skeleton/cypress/integration/test.js
		//      file with execution content git file
		output.PrintLogf("%s Passing Cypress test as single file not implemented yet", ui.IconCross)
		return result, errors.Errorf("passing cypress test as single file not implemented yet")
	}

	output.PrintLogf("%s Test content checked", ui.IconCheckMark)

	out, err := r.installModule(runPath)
	if err != nil {
		return result, errors.Errorf("cypress module install error: %v\n\n%s", err, out)
	}

	// handle project local Cypress version install (`Cypress` app)
	command, args := executor.MergeCommandAndArgs(execution.Command, []string{"install"})
	out, err = executor.Run(runPath, command, nil, args...)
	if err != nil {
		return result, errors.Errorf("cypress binary install error: %v\n\n%s", err, out)
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	envVars := make([]string, 0, len(envManager.Variables))
	for _, value := range envManager.Variables {
		if !value.IsSecret() {
			output.PrintLogf("%s=%s", value.Name, value.Value)
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", value.Name, value.Value))
	}

	junitReportDir := filepath.Join(projectPath, "results")
	junitReportPath := filepath.Join(projectPath, "results/junit-[hash].xml")

	var project string
	if workingDir != "" {
		project = projectPath
	}

	// append args from execution
	args = execution.Args
	hasJunit := false
	hasReporter := false
	for i := len(args) - 1; i >= 0; i-- {
		if project == "" && (args[i] == "--project" || args[i] == "<projectPath>") {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<projectPath>" {
			args[i] = project
		}

		if args[i] == "junit" {
			hasJunit = true
		}

		if args[i] == "--reporter" {
			hasReporter = true
		}

		if strings.Contains(args[i], "<reportFile>") {
			args[i] = strings.ReplaceAll(args[i], "<reportFile>", junitReportPath)
		}

		if args[i] == "<envVars>" {
			args[i] = strings.Join(envVars, ",")
		}

		args[i] = os.ExpandEnv(args[i])
	}

	// run cypress inside repo directory ignore execution error in case of failed test
	command, args = executor.MergeCommandAndArgs(execution.Command, args)
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	out, err = executor.Run(runPath, command, envManager, args...)
	out = envManager.ObfuscateSecrets(out)

	var suites []junit.Suite
	var serr error
	if hasJunit && hasReporter {
		suites, serr = junit.IngestDir(junitReportDir)
		result = MapJunitToExecutionResults(out, suites)
	} else {
		result = makeSuccessExecution(out)
	}

	output.PrintLogf("%s Mapped Junit to Execution Results...", ui.IconCheckMark)

	if steps := result.FailedSteps(); len(steps) > 0 {
		output.PrintLogf("Test Failed steps")
		for _, s := range steps {
			errorMessage := ""
			for _, a := range s.AssertionResults {
				errorMessage += a.ErrorMessage
			}
			output.PrintLog("step: " + s.Name + " error: " + errorMessage)
		}
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
			junitReportDir,
			filepath.Join(projectPath, "cypress/videos"),
			filepath.Join(projectPath, "cypress/screenshots"),
		}

		var masks []string
		if execution.ArtifactRequest != nil {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
			masks = execution.ArtifactRequest.Masks
		}

		output.PrintLogf("Scraping directories: %v", directories)

		if err := r.Scraper.Scrape(ctx, directories, masks, execution); err != nil {
			return *result.WithErrors(err), nil
		}
	}

	return *result.WithErrors(err, serr, rerr), nil
}

func (r *CypressRunner) installModule(runPath string) (out []byte, err error) {
	if _, err = os.Stat(filepath.Join(runPath, "package.json")); err == nil {
		// be gentle to different cypress versions, run from local npm deps
		if r.dependency == "npm" {
			out, err = executor.Run(runPath, "npm", nil, "install")
			if err != nil {
				return nil, errors.Errorf("npm install error: %v\n\n%s", err, out)
			}
		}

		if r.dependency == "yarn" {
			out, err = executor.Run(runPath, "yarn", nil, "install")
			if err != nil {
				return nil, errors.Errorf("yarn install error: %v\n\n%s", err, out)
			}
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if r.dependency == "npm" {
			out, err = executor.Run(runPath, "npm", nil, "init", "--yes")
			if err != nil {
				return nil, errors.Errorf("npm init error: %v\n\n%s", err, out)
			}

			out, err = executor.Run(runPath, "npm", nil, "install", "cypress", "--save-dev")
			if err != nil {
				return nil, errors.Errorf("npm install cypress error: %v\n\n%s", err, out)
			}
		}

		if r.dependency == "yarn" {
			out, err = executor.Run(runPath, "yarn", nil, "init", "--yes")
			if err != nil {
				return nil, errors.Errorf("yarn init error: %v\n\n%s", err, out)
			}

			out, err = executor.Run(runPath, "yarn", nil, "add", "cypress", "--dev")
			if err != nil {
				return nil, errors.Errorf("yarn add cypress error: %v\n\n%s", err, out)
			}
		}
	} else {
		output.PrintLogf("%s failed checking package.json file: %s", ui.IconCross, err.Error())
		return nil, errors.Errorf("checking package.json file: %v", err)
	}
	return
}

// Validate checks if Execution has valid data in context of Cypress executor.
// Cypress executor runs currently only based on Cypress project
func (r *CypressRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		output.PrintLogf("%s Invalid input: can't find any content to run in execution data", ui.IconCross)
		return errors.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	if execution.Content.Repository == nil {
		output.PrintLogf("%s Cypress executor handles only repository based tests, but repository is nil", ui.IconCross)
		return errors.Errorf("cypress executor handles only repository based tests, but repository is nil")
	}

	if execution.Content.Repository.Branch == "" && execution.Content.Repository.Commit == "" {
		output.PrintLogf("%s can't find branch or commit in params, repo:%+v", ui.IconCross, execution.Content.Repository)
		return errors.Errorf("can't find branch or commit in params, repo:%+v", execution.Content.Repository)
	}

	return nil
}

func makeSuccessExecution(out []byte) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status

	return result
}

func MapJunitToExecutionResults(out []byte, suites []junit.Suite) (result testkube.ExecutionResult) {
	result = makeSuccessExecution(out)

	for _, suite := range suites {
		for _, test := range suite.Tests {

			result.Steps = append(
				result.Steps,
				testkube.ExecutionStepResult{
					Name:     fmt.Sprintf("%s - %s", suite.Name, test.Name),
					Duration: test.Duration.String(),
					Status:   MapStatus(test.Status),
				})
		}

		// TODO parse sub suites recursively

	}

	return result
}

func MapStatus(in junit.Status) (out string) {
	switch string(in) {
	case "passed":
		return string(testkube.PASSED_ExecutionStatus)
	case "pending":
		return string(testkube.QUEUED_ExecutionStatus)
	case "skipped":
		return string(testkube.SKIPPED_ExecutionStatus)
	default:
		return string(testkube.FAILED_ExecutionStatus)
	}
}

// GetType returns runner type
func (r *CypressRunner) GetType() runner.Type {
	return runner.TypeMain
}
