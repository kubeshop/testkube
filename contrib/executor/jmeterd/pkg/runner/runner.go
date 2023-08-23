package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/jmeter_env"
	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/parser"
	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/slaves"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
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
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintEvent(
		fmt.Sprintf("%s Running with config", ui.IconTruck),
		"scraperEnabled", r.Params.ScrapperEnabled,
		"dataDir", r.Params.DataDir,
		"SSL", r.Params.Ssl,
		"endpoint", r.Params.Endpoint,
	)

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	path, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
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

		output.PrintLog(fmt.Sprintf("execution arg before %s", execution.Args))
		execution.Args = execution.Args[:len(execution.Args)-1]
		output.PrintLog(fmt.Sprintf("execution arg afrer %s", execution.Args))
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

	slavesEnvVariables := jmeter_env.ExtractSlaveEnvVariables(envManager.Variables)
	// compose parameters passed to JMeter with -J
	params := make([]string, 0, len(envManager.Variables))
	for _, value := range envManager.Variables {
		
		if value.Name == jmeter_env.MasterOverrideJvmArgs || value.Name == jmeter_env.MasterAdditionalJvmArgs {
			//Skip JVM ARGS to be appended in the command
			continue
		}
		params = append(params, fmt.Sprintf("-J%s=%s", value.Name, value.Value))

	}

	runPath := r.Params.DataDir
	if workingDir != "" {
		runPath = workingDir
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
	args := execution.Args
	for i := range args {
		if args[i] == "<runPath>" {
			args[i] = path
		}

		if args[i] == "<jtlFile>" {
			args[i] = jtlPath
		}

		if args[i] == "<reportFile>" {
			args[i] = reportPath
		}

		if args[i] == "<logFile>" {
			args[i] = jmeterLogPath
		}
	}

	sl, err := slaves.NewClient(execution, r.Params, slavesEnvVariables)
	if err != nil {
		return *result.WithErrors(errors.Errorf("Getting error while creating slaves client: %v", err)), nil
	}

	//creating slaves provided in SLAVES_COUNT env variable
	slavesNameIpMap, err := sl.CreateSlaves()
	if err != nil {
		return *result.WithErrors(errors.Errorf("Getting error while creating slaves nodes: %v", err)), nil
	}
	defer sl.DeleteSlaves(slavesNameIpMap)

	args = append(args, fmt.Sprintf("-R %v", GetSlavesIpString(slavesNameIpMap)))

	for i := range args {
		if args[i] == "<envVars>" {
			newArgs := make([]string, len(args)+len(params)-1)
			copy(newArgs, args[:i])
			copy(newArgs[i:], params)
			copy(newArgs[i+len(params):], args[i+1:])
			args = newArgs
			break
		}
	}

	output.PrintLogf("%s Using arguments: %v", ui.IconWorld, args)

	entryPoint := getEntryPoint()
	for i := range execution.Command {
		if execution.Command[i] == "<entryPoint>" {
			execution.Command[i] = entryPoint
		}
	}

	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	// run JMeter inside repo directory ignore execution error in case of failed test
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(args, " "))
	out, err := executor.Run(runPath, command, envManager, args...)
	if err != nil {
		return *result.WithErrors(errors.Errorf("jmeter run error: %v", err)), nil
	}
	out = envManager.ObfuscateSecrets(out)

	output.PrintLogf("%s Getting report %s", ui.IconFile, jtlPath)
	f, err := os.Open(jtlPath)
	if err != nil {
		return *result.WithErrors(errors.Errorf("getting jtl report error: %v", err)), nil
	}

	results, err := parser.ParseCSV(f)
	f.Close()

	var executionResult testkube.ExecutionResult
	if err != nil {
		data, err := os.ReadFile(jtlPath)
		if err != nil {
			return *result.WithErrors(errors.Errorf("getting jtl report error: %v", err)), nil
		}

		testResults, err := parser.ParseXML(data)
		if err != nil {
			return *result.WithErrors(errors.Errorf("parsing jtl report error: %v", err)), nil
		}

		executionResult = MapTestResultsToExecutionResults(out, testResults)
	} else {
		executionResult = MapResultsToExecutionResults(out, results)
	}

	output.PrintLogf("%s Mapped JMeter results to Execution Results...", ui.IconCheckMark)

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled {
		directories := []string{
			outputDir,
		}
		if execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
		}

		output.PrintLogf("Scraping directories: %v", directories)
		if err := r.Scraper.Scrape(ctx, directories, execution); err != nil {
			return *executionResult.Err(err), errors.Wrap(err, "error scraping artifacts for JMeter executor")
		}
	}

	return executionResult, nil
}

func getEntryPoint() (entrypoint string) {
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
				Status:   MapResultStatus(r),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: MapResultStatus(r),
				}},
			})
	}

	return result
}

func MapTestResultsToExecutionResults(out []byte, results parser.TestResults) (result testkube.ExecutionResult) {
	result.Status = testkube.ExecutionStatusPassed

	result.Output = string(out)
	result.OutputType = "text/plain"

	samples := append(results.HTTPSamples, results.Samples...)
	for _, r := range samples {
		if !r.Success {
			result.Status = testkube.ExecutionStatusFailed
			if r.AssertionResult != nil {
				result.ErrorMessage = r.AssertionResult.FailureMessage
			}
		}

		result.Steps = append(
			result.Steps,
			testkube.ExecutionStepResult{
				Name:     r.Label,
				Duration: fmt.Sprintf("%dms", r.Time),
				Status:   MapTestResultStatus(r.Success),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: MapTestResultStatus(r.Success),
				}},
			})
	}

	return result
}

func MapResultStatus(result parser.Result) string {
	if result.Success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}

func MapTestResultStatus(success bool) string {
	if success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}

// GetType returns runner type
func (r *JMeterRunner) GetType() runner.Type {
	return runner.TypeMain
}

func GetSlavesIpString(podNameIpMap map[string]string) string {
	podIps := []string{}
	for _, ip := range podNameIpMap {
		podIps = append(podIps, ip)
	}
	return strings.Join(podIps, ",")
}
