package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
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

	testFile, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		scriptName := execution.Args[len(execution.Args)-1]
		if workingDir != "" {
			testFile = ""
			if execution.Content != nil && execution.Content.Repository != nil {
				scriptName = filepath.Join(execution.Content.Repository.Path, scriptName)
			}
		}

		execution.Args = execution.Args[:len(execution.Args)-1]
		output.PrintLogf("%s It is a directory test - trying to find file from the last executor argument %s in directory %s", ui.IconWorld, scriptName, testFile)

		// sanity checking for test script
		scriptFile := filepath.Join(testFile, workingDir, scriptName)
		fileInfo, errFile := os.Stat(scriptFile)
		if errors.Is(errFile, os.ErrNotExist) || fileInfo.IsDir() {
			output.PrintLogf("%s Could not find file %s in the directory, error: %s", ui.IconCross, scriptName, errFile)
			return *result.Err(errors.Errorf("could not find file %s in the directory: %v", scriptName, errFile)), nil
		}
		testFile = scriptFile
	}

	setUpEnvironment(execution.Args, testFile)

	output.PrintLogf("%s Running SoapUI tests", ui.IconMicroscope)
	result = r.runSoapUI(&execution, workingDir)

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	if r.Params.ScrapperEnabled {
		directories := []string{r.SoapUILogsPath}
		var masks []string
		if execution.ArtifactRequest != nil {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
			masks = execution.ArtifactRequest.Masks
		}

		output.PrintLogf("Scraping directories: %v with masks: %v", directories, masks)

		if err := r.Scraper.Scrape(ctx, directories, masks, execution); err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts from SoapUI executor")
		}
	}

	if rerr != nil {
		return *result.Err(rerr), nil
	}

	return result, nil
}

// setUpEnvironment sets up the COMMAND_LINE environment variable to
// contain the incoming arguments and to point to the test file path
func setUpEnvironment(args []string, testFilePath string) {
	for i := range args {
		if args[i] == "<runPath>" {
			args[i] = testFilePath
		}

		args[i] = os.ExpandEnv(args[i])
	}
	os.Setenv("COMMAND_LINE", strings.Join(args, " "))
}

// runSoapUI runs the SoapUI executable and returns the output
func (r *SoapUIRunner) runSoapUI(execution *testkube.Execution, workingDir string) testkube.ExecutionResult {

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	runPath := workingDir
	command, args := executor.MergeCommandAndArgs(execution.Command, nil)
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, strings.Join(execution.Command, " "),
		strings.Join(envManager.ObfuscateStringSlice(execution.Args), " "))
	output, err := executor.Run(runPath, command, envManager, args...)
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
