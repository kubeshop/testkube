package runner

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/command"
	"github.com/kubeshop/testkube-executor-tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

const TRACETEST_ENDPOINT_VAR = "TRACETEST_ENDPOINT"
const TRACETEST_OUTPUT_ENDPOINT_VAR = "TRACETEST_OUTPUT_ENDPOINT"

func NewRunner() (*TracetestRunner, error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing Runner", ui.IconTruck))

	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Testkube variables: %w", err)
	}

	return &TracetestRunner{
		Fetcher: content.NewFetcher(""),
		Params:  params,
	}, nil
}

type TracetestRunner struct {
	Fetcher content.ContentFetcher
	Params  envs.Params
}

func (r *TracetestRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing test run", ui.IconTruck))

	envManager := secret.NewEnvManagerWithVars(execution.Variables)
	envManager.GetVars(envManager.Variables)

	// Get TRACETEST_ENDPOINT from execution variables
	te, err := getTracetestEndpointFromVars(envManager)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: TRACETEST_ENDPOINT variable was not found", ui.IconCross))
		return result, err
	}

	// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
	toe, _ := getTracetestOutputEndpointFromVars(envManager)

	// Configure Tracetest CLI
	// err = configureTracetestCLI(te)
	// if err != nil {
	//	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Error when configuring the Tracetest CLI", ui.IconCross))
	//	return result, err
	// }

	// Get execution content file path
	path, err := getContentPath(r.Params.DataDir, execution.Content, r.Fetcher)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Error fetching the content file", ui.IconCross))
		return result, err
	}

	// Prepare args for test run command
	args := []string{
		"test", "run", "--server-url", te, "--definition", path, "--wait-for-result", "--output", "pretty",
	}
	// Pass additional execution arguments to tracetest
	args = append(args, execution.Args...)

	// Run tracetest test from definition file
	output, err := executor.Run("", "tracetest", envManager, args...)
	runResult := model.Result{Output: string(output), ServerEndpoint: te, OutputEndpoint: toe}

	if err != nil {
		result.ErrorMessage = runResult.GetOutput()
		result.Output = runResult.GetOutput()
		result.Status = testkube.ExecutionStatusFailed
		return result, nil
	}

	result.Output = runResult.GetOutput()
	result.Status = runResult.GetStatus()
	return result, nil
}

// GetType returns runner type
func (r *TracetestRunner) GetType() runner.Type {
	return runner.TypeMain
}

// Get TRACETEST_ENDPOINT from execution variables
func getTracetestEndpointFromVars(envManager *secret.EnvManager) (string, error) {
	v, ok := envManager.Variables[TRACETEST_ENDPOINT_VAR]
	if !ok {
		return "", fmt.Errorf("TRACETEST_ENDPOINT variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}

// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
func getTracetestOutputEndpointFromVars(envManager *secret.EnvManager) (string, error) {
	v, ok := envManager.Variables[TRACETEST_OUTPUT_ENDPOINT_VAR]
	if !ok {
		return "", fmt.Errorf("TRACETEST_OUTPUT_ENDPOINT variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}

// Configure Tracetest CLI
func configureTracetestCLI(endpoint string) error {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Configuring Tracetest CLI with endpoint %s", ui.IconTruck, endpoint))
	_, err := command.Run("tracetest", "configure", "--endpoint", endpoint, "--analytics=false")
	if err != nil {
		return err
	}

	// Get Tracetest Version
	output, err := command.Run("tracetest", "version")
	if err == nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Tracetest CLI version, %s ", ui.IconCheckMark, string(output)))
	}

	return err
}

// Get execution content file path
func getContentPath(dataDir string, content *testkube.TestContent, fetcher content.ContentFetcher) (string, error) {
	// Check that the data dir exists
	_, err := os.Stat(dataDir)
	if errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	// Fetch execution content to file
	path, err := fetcher.Fetch(content)
	if err != nil {
		return "", err
	}

	if !content.IsFile() {
		return "", testkube.ErrTestContentTypeNotFile
	}

	return path, nil
}
