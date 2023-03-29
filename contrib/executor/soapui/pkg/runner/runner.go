package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/ui"
)

const FailureMessage string = "finished with status [FAILED]"

// NewRunner creates a new SoapUIRunner
func NewRunner() (*SoapUIRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Artillery runner variables: %w", err)
	}

	return &SoapUIRunner{
		SoapUIExecPath: "/usr/local/SmartBear/EntryPoint.sh",
		SoapUILogsPath: "/home/soapui/.soapuios/logs",
		Scraper: scraper.NewMinioScraper(
			params.Endpoint,
			params.AccessKeyID,
			params.SecretAccessKey,
			params.Region,
			params.Token,
			params.Bucket,
			params.Ssl,
		),
		DataDir: params.DataDir,
	}, nil
}

// SoapUIRunner runs SoapUI tests
type SoapUIRunner struct {
	SoapUIExecPath string
	SoapUILogsPath string
	Scraper        scraper.Scraper
	DataDir        string
}

// Run executes the test and returns the test results
func (r *SoapUIRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))

	testFile := ""
	if execution.Content != nil {
		if execution.Content.Type_ == string(testkube.TestContentTypeString) ||
			execution.Content.Type_ == string(testkube.TestContentTypeFileURI) {
			testFile = filepath.Join(r.DataDir, "test-content")
		}

		if execution.Content.Type_ == string(testkube.TestContentTypeGitFile) ||
			execution.Content.Type_ == string(testkube.TestContentTypeGitDir) ||
			execution.Content.Type_ == string(testkube.TestContentTypeGit) {
			testFile = filepath.Join(r.DataDir, "repo")
			if execution.Content.Repository != nil {
				testFile = filepath.Join(testFile, execution.Content.Repository.Path)
			}
		}
	}

	setUpEnvironment(execution.Args, testFile)

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		return testkube.ExecutionResult{}, errors.New("SoapUI executor only tests one project per execution, a directory of projects was given")
	}

	output.PrintLog(fmt.Sprintf("%s Running SoapUI tests", ui.IconMicroscope))
	result = r.runSoapUI(&execution)

	if err = r.Scraper.Scrape(execution.Id, []string{r.SoapUILogsPath}); err != nil {
		return result, fmt.Errorf("failed getting artifacts: %w", err)
	}

	return result, nil
}

// setUpEnvironment sets up the COMMAND_LINE environment variable to
// contain the incoming arguments and to point to the test file path
func setUpEnvironment(args []string, testFilePath string) {
	args = append(args, testFilePath)
	os.Setenv("COMMAND_LINE", strings.Join(args, " "))
}

// runSoapUI runs the SoapUI executable and returns the output
func (r *SoapUIRunner) runSoapUI(execution *testkube.Execution) testkube.ExecutionResult {

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	runPath := ""
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	output, err := executor.Run(runPath, "/bin/sh", envManager, r.SoapUIExecPath)
	output = envManager.ObfuscateSecrets(output)
	if err != nil {
		return testkube.ExecutionResult{
			Status:       testkube.ExecutionStatusFailed,
			ErrorMessage: err.Error(),
		}
	}
	if strings.Contains(string(output), FailureMessage) {
		return testkube.ExecutionResult{
			Status:       testkube.ExecutionStatusFailed,
			ErrorMessage: FailureMessage,
			Output:       string(output),
		}
	}

	return testkube.ExecutionResult{
		Status: testkube.ExecutionStatusPassed,
		Output: string(output),
	}
}

// GetType returns runner type
func (r *SoapUIRunner) GetType() runner.Type {
	return runner.TypeMain
}
