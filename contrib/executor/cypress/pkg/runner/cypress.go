package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	junit "github.com/joshdk/go-junit"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCypressRunner(dependency string) (*CypressRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, errors.Errorf("could not initialize Cypress runner variables: %v", err)
	}

	r := &CypressRunner{
		Params:     params,
		dependency: dependency,
	}

	return r, nil
}

// CypressRunner - implements runner interface used in worker to start test execution
type CypressRunner struct {
	Params     envs.Params
	dependency string
}

func (r *CypressRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))
	// make some validation
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	output.PrintLog(fmt.Sprintf("%s Checking test content from %s...", ui.IconBox, execution.Content.Type_))
	// check that the datadir exists
	_, err = os.Stat(r.Params.DataDir)
	if errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	runPath := filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.Path)
	projectPath := runPath
	if execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	fileInfo, err := os.Stat(projectPath)
	if err != nil {
		return result, err
	}

	if !fileInfo.IsDir() {
		output.PrintLog(fmt.Sprintf("%s Using file...", ui.IconTruck))

		// TODO add cypress project structure
		// TODO checkout this repo with `skeleton` path
		// TODO overwrite skeleton/cypress/integration/test.js
		//      file with execution content git file
		output.PrintLog(fmt.Sprintf("%s Passing Cypress test as single file not implemented yet", ui.IconCross))
		return result, errors.Errorf("passing cypress test as single file not implemented yet")
	}

	output.PrintLog(fmt.Sprintf("%s Test content checked", ui.IconCheckMark))

	out, err := r.installModule(runPath)
	if err != nil {
		return result, errors.Errorf("cypress module install error: %v\n\n%s", err, out)
	}

	// handle project local Cypress version install (`Cypress` app)
	out, err = executor.Run(runPath, "./node_modules/cypress/bin/cypress", nil, "install")
	if err != nil {
		return result, errors.Errorf("cypress binary install error: %v\n\n%s", err, out)
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	envVars := make([]string, 0, len(envManager.Variables))
	for _, value := range envManager.Variables {
		if !value.IsSecret() {
			output.PrintLog(fmt.Sprintf("%s=%s", value.Name, value.Value))
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", value.Name, value.Value))
	}

	junitReportPath := filepath.Join(projectPath, "results/junit.xml")
	args := []string{"run", "--reporter", "junit", "--reporter-options", fmt.Sprintf("mochaFile=%s,toConsole=false", junitReportPath),
		"--env", strings.Join(envVars, ",")}

	if execution.Content.Repository.WorkingDir != "" {
		args = append(args, "--project", projectPath)
	}

	// append args from execution
	args = append(args, execution.Args...)

	// run cypress inside repo directory ignore execution error in case of failed test
	out, err = executor.Run(runPath, "./node_modules/cypress/bin/cypress", envManager, args...)
	out = envManager.ObfuscateSecrets(out)
	suites, serr := junit.IngestFile(junitReportPath)
	result = MapJunitToExecutionResults(out, suites)
	output.PrintLog(fmt.Sprintf("%s Mapped Junit to Execution Results...", ui.IconCheckMark))

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled {
		directories := []string{
			projectPath,
		}

		output.PrintLog(fmt.Sprintf("Scraping directories: %v", directories))

		if err := factory.Scrape(context.Background(), directories, execution, r.Params); err != nil {
			return *result.WithErrors(err), nil
		}
	}

	return *result.WithErrors(err, serr), nil
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
		output.PrintLog(fmt.Sprintf("%s failed checking package.json file: %s", ui.IconCross, err.Error()))
		return nil, errors.Errorf("checking package.json file: %v", err)
	}
	return
}

// Validate checks if Execution has valid data in context of Cypress executor
// Cypress executor runs currently only based on cypress project
func (r *CypressRunner) Validate(execution testkube.Execution) error {

	if execution.Content == nil {
		output.PrintLog(fmt.Sprintf("%s Invalid input: can't find any content to run in execution data", ui.IconCross))
		return errors.Errorf("can't find any content to run in execution data: %+v", execution)
	}

	if execution.Content.Repository == nil {
		output.PrintLog(fmt.Sprintf("%s Cypress executor handles only repository based tests, but repository is nil", ui.IconCross))
		return errors.Errorf("cypress executor handles only repository based tests, but repository is nil")
	}

	if execution.Content.Repository.Branch == "" && execution.Content.Repository.Commit == "" {
		output.PrintLog(fmt.Sprintf("%s can't find branch or commit in params, repo:%+v", ui.IconCross, execution.Content.Repository))
		return errors.Errorf("can't find branch or commit in params, repo:%+v", execution.Content.Repository)
	}

	return nil
}

func MapJunitToExecutionResults(out []byte, suites []junit.Suite) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status
	result.Output = string(out)
	result.OutputType = "text/plain"

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
	default:
		return string(testkube.FAILED_ExecutionStatus)
	}
}

// GetType returns runner type
func (r *CypressRunner) GetType() runner.Type {
	return runner.TypeMain
}
