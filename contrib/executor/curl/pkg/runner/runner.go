package runner

import (
	"context"
	"encoding/json"
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
	contentPkg "github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

const CurlAdditionalFlags = "-is"

// CurlRunner is used to run curl commands.
type CurlRunner struct {
	Params  envs.Params
	Fetcher contentPkg.ContentFetcher
	Log     *zap.SugaredLogger
}

var _ runner.Runner = &CurlRunner{}

func NewCurlRunner() (*CurlRunner, error) {
	outputPkg.PrintLogf("%s Preparing test runner", ui.IconTruck)
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, errors.Errorf("could not initialize cURL runner variables: %v", err)
	}

	return &CurlRunner{
		Log:     log.DefaultLogger,
		Params:  params,
		Fetcher: contentPkg.NewFetcher(""),
	}, nil
}

func (r *CurlRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	outputPkg.PrintLogf("%s Preparing for test run", ui.IconTruck)
	var runnerInput CurlRunnerInput
	if r.Params.GitUsername != "" || r.Params.GitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = r.Params.GitUsername
			execution.Content.Repository.Token = r.Params.GitToken
		}
	}

	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return result, err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		return result, testkube.ErrTestContentTypeNotFile
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

	command := runnerInput.Command[0]
	if command != "curl" {
		outputPkg.PrintLogf("%s you can run only `curl` commands with this executor but passed: `%s`", ui.IconCross, command)
		return result, errors.Errorf("you can run only `curl` commands with this executor but passed: `%s`", command)
	}

	runnerInput.Command[0] = CurlAdditionalFlags

	args := runnerInput.Command
	args = append(args, execution.Args...)

	runPath := ""
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	output, err := executor.Run(runPath, command, envManager, args...)
	output = envManager.ObfuscateSecrets(output)

	if err != nil {
		r.Log.Errorf("Error occured when running a command %s", err)
		return *result.Err(err), nil
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
		outputPkg.PrintLogf("%s Test run failed: cannot process expected status: %s", ui.IconCross, err.Error())
		return *result.Err(errors.Errorf("response status don't match expected %d got %d", expectedStatus, responseStatus)), nil
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		outputPkg.PrintLogf("%s Test run failed: response doesn't contain body: %s", ui.IconCross, runnerInput.ExpectedBody)
		return *result.Err(errors.Errorf("response doesn't contain body: %s", runnerInput.ExpectedBody)), nil
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
