package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	contentPkg "github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

// CurlRunner is used to run curl commands.
type CurlRunner struct {
	Params  envs.Params
	Log     *zap.SugaredLogger
	Scraper scraper.Scraper
}

var _ runner.Runner = &CurlRunner{}

func NewCurlRunner(ctx context.Context, params envs.Params) (*CurlRunner, error) {
	outputPkg.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &CurlRunner{
		Log:    log.DefaultLogger,
		Params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *CurlRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

	outputPkg.PrintLogf("%s Preparing for test run", ui.IconTruck)
	var runnerInput CurlRunnerInput

	path, workingDir, err := contentPkg.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		outputPkg.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		scriptName := execution.Args[len(execution.Args)-1]
		if workingDir != "" {
			path = ""
			if execution.Content != nil && execution.Content.Repository != nil {
				scriptName = filepath.Join(execution.Content.Repository.Path, scriptName)
			}
		}

		execution.Args = execution.Args[:len(execution.Args)-1]
		output.PrintLogf("%s It is a directory test - trying to find file from the last executor argument %s in directory %s", ui.IconWorld, scriptName, path)

		// sanity checking for test script
		scriptFile := filepath.Join(path, workingDir, scriptName)
		fileInfo, errFile := os.Stat(scriptFile)
		if errors.Is(errFile, os.ErrNotExist) || fileInfo.IsDir() {
			output.PrintLogf("%s Could not find file %s in the directory, error: %s", ui.IconCross, scriptName, errFile)
			return *result.Err(errors.Errorf("could not find file %s in the directory: %v", scriptName, errFile)), nil
		}
		path = scriptFile
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(content, &runnerInput)
	if err != nil {
		return result, err
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	variables := testkube.VariablesToMap(envManager.Variables)

	outputPkg.PrintLogf("%s Filling in the input templates", ui.IconKey)
	err = runnerInput.FillTemplates(variables)
	if err != nil {
		outputPkg.PrintLogf("%s Failed to fill in the input templates: %s", ui.IconCross, err.Error())
		r.Log.Errorf("Error occured when resolving input templates %s", err)
		return *result.Err(err), nil
	}
	outputPkg.PrintLogf("%s Successfully filled the input templates", ui.IconCheckMark)

	command := ""
	var args []string
	if len(execution.Command) != 0 {
		command = execution.Command[0]
		args = execution.Command[1:]
	}

	if len(runnerInput.Command) != 0 {
		command = runnerInput.Command[0]
		args = runnerInput.Command[1:]
	}

	if command != "curl" {
		outputPkg.PrintLogf("%s you can run only `curl` commands with this executor but passed: `%s`", ui.IconCross, command)
		return result, errors.Errorf("you can run only `curl` commands with this executor but passed: `%s`", command)
	}

	args = append(args, execution.Args...)
	for i := range args {
		args[i] = os.ExpandEnv(args[i])
	}

	runPath := workingDir
	outputPkg.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	output, err := executor.Run(runPath, command, envManager, args...)
	output = envManager.ObfuscateSecrets(output)

	if err != nil {
		r.Log.Errorf("Error occured when running a command %s", err)
		return *result.Err(err), nil
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		outputPkg.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			outputPkg.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		outputPkg.PrintLogf("Scraping directories: %v with masks: %v", execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks, execution); err != nil {
			return *result.WithErrors(err), nil
		}
	}

	outputString := string(output)
	result.Output = outputString
	responseStatus, err := getResponseCode(outputString)
	if err != nil {
		outputPkg.PrintLogf("%s Test run failed: %s", ui.IconCross, err.Error())
		return *result.Err(err), nil
	}

	expectedStatus, err := strconv.Atoi(runnerInput.ExpectedStatus)
	if err != nil {
		outputPkg.PrintLogf("%s Test run failed: cannot process expected status: %s", ui.IconCross, err.Error())
		return *result.Err(errors.Errorf("cannot process expected status %s", runnerInput.ExpectedStatus)), nil
	}

	if responseStatus != expectedStatus {
		outputPkg.PrintLogf("%s Test run failed: response status don't match: expected %d got %d", ui.IconCross, expectedStatus, responseStatus)
		return *result.Err(errors.Errorf("response status don't match expected %d got %d", expectedStatus, responseStatus)), nil
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		outputPkg.PrintLogf("%s Test run failed: response doesn't contain body: %s", ui.IconCross, runnerInput.ExpectedBody)
		return *result.Err(errors.Errorf("response doesn't contain body: %s", runnerInput.ExpectedBody)), nil
	}

	if rerr != nil {
		return *result.Err(rerr), nil
	}

	outputPkg.PrintLogf("%s Test run succeeded", ui.IconCheckMark)

	return testkube.ExecutionResult{
		Status: testkube.ExecutionStatusPassed,
		Output: outputString,
	}, nil
}

func getResponseCode(curlOutput string) (int, error) {
	re := regexp.MustCompile(`\A\S*\s(\d+)`)
	matches := re.FindStringSubmatch(curlOutput)
	if len(matches) == 0 {
		return -1, errors.Errorf("could not find a response status in the command output")
	}
	return strconv.Atoi(matches[1])
}

// GetType returns runner type
func (r *CurlRunner) GetType() runner.Type {
	return runner.TypeMain
}
