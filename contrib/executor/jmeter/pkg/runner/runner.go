package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/executor/scraper"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/parser"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunner(ctx context.Context, params envs.Params) (*JMeterRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))

	var err error
	r := &JMeterRunner{
		Params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// JMeterRunner runner
type JMeterRunner struct {
	Params  envs.Params
	Scraper scraper.Scraper
}

var _ runner.Runner = &JMeterRunner{}

func (r *JMeterRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {

	output.PrintEvent(
		fmt.Sprintf("%s Running with config", ui.IconTruck),
		"scraperEnabled", r.Params.ScrapperEnabled,
		"dataDir", r.Params.DataDir,
		"SSL", r.Params.Ssl,
		"endpoint", r.Params.Endpoint,
	)

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	gitUsername := r.Params.GitUsername
	gitToken := r.Params.GitToken
	if gitUsername != "" || gitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = gitUsername
			execution.Content.Repository.Token = gitToken
		}
	}

	path := ""
	workingDir := ""
	basePath, err := filepath.Abs(r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
		basePath = r.Params.DataDir
	}
	if execution.Content != nil {
		isStringContentType := execution.Content.Type_ == string(testkube.TestContentTypeString)
		isFileURIContentType := execution.Content.Type_ == string(testkube.TestContentTypeFileURI)
		if isStringContentType || isFileURIContentType {
			path = filepath.Join(basePath, "test-content")
		}

		isGitFileContentType := execution.Content.Type_ == string(testkube.TestContentTypeGitFile)
		isGitDirContentType := execution.Content.Type_ == string(testkube.TestContentTypeGitDir)
		isGitContentType := execution.Content.Type_ == string(testkube.TestContentTypeGit)
		if isGitFileContentType || isGitDirContentType || isGitContentType {
			path = filepath.Join(basePath, "repo")
			if execution.Content.Repository != nil {
				path = filepath.Join(path, execution.Content.Repository.Path)
				workingDir = execution.Content.Repository.WorkingDir
			}
		}
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return result, err
	}

	if fileInfo.IsDir() {
		scriptName := execution.Args[len(execution.Args)-1]
		if workingDir != "" {
			path = filepath.Join(r.Params.DataDir, "repo")
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

	// compose parameters passed to JMeter with -J
	params := make([]string, 0, len(envManager.Variables))
	for _, value := range envManager.Variables {
		params = append(params, fmt.Sprintf("-J%s=%s", value.Name, value.Value))
	}

	runPath := basePath
	if workingDir != "" {
		runPath = filepath.Join(basePath, "repo", workingDir)
	}

	outputDir := filepath.Join(runPath, "output")
	// clean output directory it already exists, only useful for local development
	_, err = os.Stat(outputDir)
	if err == nil {
		if err = os.RemoveAll(outputDir); err != nil {
			output.PrintLogf("%s Failed to clean output directory %s", ui.IconWarning, outputDir)
		}
	}
	// recreate output directory with wide permissions so JMeter can create report files
	if err = os.Mkdir(outputDir, 0777); err != nil {
		return *result.Err(errors.Errorf("could not create directory %s: %v", runPath, err)), nil
	}

	jtlPath := filepath.Join(outputDir, "report.jtl")
	reportPath := filepath.Join(outputDir, "report")
	jmeterLogPath := filepath.Join(outputDir, "jmeter.log")
	args := []string{"-n", "-j", jmeterLogPath, "-t", path, "-l", jtlPath, "-e", "-o", reportPath}
	args = append(args, params...)

	// append args from execution
	args = append(args, execution.Args...)
	output.PrintLogf("%s Using arguments: %v", ui.IconWorld, args)

	mainCmd := getEntrypoint()
	// run JMeter inside repo directory ignore execution error in case of failed test
	out, err := executor.Run(runPath, mainCmd, envManager, args...)
	if err != nil {
		return *result.WithErrors(errors.Errorf("jmeter run error: %v", err)), nil
	}
	out = envManager.ObfuscateSecrets(out)

	output.PrintLogf("%s Getting report %s", ui.IconFile, jtlPath)
	f, err := os.Open(jtlPath)
	if err != nil {
		return *result.WithErrors(errors.Errorf("getting jtl report error: %v", err)), nil
	}

	results := parser.Parse(f)
	executionResult := MapResultsToExecutionResults(out, results)
	output.PrintLogf("%s Mapped JMeter results to Execution Results...", ui.IconCheckMark)

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled {
		directories := []string{
			outputDir,
		}

		output.PrintLogf("Scraping directories: %v", directories)

		if err := r.Scraper.Scrape(ctx, directories, execution); err != nil {
			return *executionResult.Err(err), errors.Wrap(err, "error scraping artifacts for JMeter executor")
		}
	}

	return executionResult, nil
}

func getEntrypoint() (entrypoint string) {
	if entrypoint = os.Getenv("ENTRYPOINT_CMD"); entrypoint != "" {
		return entrypoint
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return filepath.Join(wd, "scripts/entrypoint.sh")
}

func MapResultsToExecutionResults(out []byte, results parser.Results) (result testkube.ExecutionResult) {
	result.Status = testkube.ExecutionStatusPassed
	if results.HasError {
		result.Status = testkube.ExecutionStatusFailed
		result.ErrorMessage = results.LastErrorMessage
	}

	result.Output = string(out)
	result.OutputType = "text/plain"

	for _, r := range results.Results {
		result.Steps = append(
			result.Steps,
			testkube.ExecutionStepResult{
				Name:     r.Label,
				Duration: r.Duration.String(),
				Status:   MapStatus(r),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: MapStatus(r),
				}},
			})
	}

	return result
}

func MapStatus(result parser.Result) string {
	if result.Success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}

// GetType returns runner type
func (r *JMeterRunner) GetType() runner.Type {
	return runner.TypeMain
}
