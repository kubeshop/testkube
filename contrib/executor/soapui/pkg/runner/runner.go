package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/ui"
)

const FailureMessage string = "finished with status [FAILED]"

// NewRunner creates a new SoapUIRunner
func NewRunner(ctx context.Context, params envs.Params) (*SoapUIRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &SoapUIRunner{
		SoapUILogsPath: "/home/soapui/.soapuios/logs",
		Params:         params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// SoapUIRunner runs SoapUI tests
type SoapUIRunner struct {
	SoapUILogsPath string
	Scraper        scraper.Scraper
	Params         envs.Params
}

var _ runner.Runner = &SoapUIRunner{}

// Run executes the test and returns the test results
func (r *SoapUIRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)

	testFile := ""
	if execution.Content != nil {
		if execution.Content.Type_ == string(testkube.TestContentTypeString) ||
			execution.Content.Type_ == string(testkube.TestContentTypeFileURI) {
			testFile = filepath.Join(r.Params.DataDir, "test-content")
		}

		if execution.Content.Type_ == string(testkube.TestContentTypeGitFile) ||
			execution.Content.Type_ == string(testkube.TestContentTypeGitDir) ||
			execution.Content.Type_ == string(testkube.TestContentTypeGit) {
			testFile = filepath.Join(r.Params.DataDir, "repo")
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

	output.PrintLogf("%s Running SoapUI tests", ui.IconMicroscope)
	result = r.runSoapUI(&execution)

	if r.Params.ScrapperEnabled {
		directories := []string{r.SoapUILogsPath}
		if execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
		}

		output.PrintLogf("Scraping directories: %v", directories)

		if err := r.Scraper.Scrape(ctx, directories, execution); err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts from SoapUI executor")
		}
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
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	command := strings.Join(execution.Command, " ")
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(execution.Args, " "))
	output, err := executor.Run(runPath, command, envManager, execution.Args...)
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
