package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/k8sclient"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/parser"
	"github.com/kubeshop/testkube/contrib/executor/jmeterd/pkg/slaves"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

type JMeterMode string

const (
	jmeterModeStandalone        JMeterMode = "standalone"
	jmeterModeDistributed       JMeterMode = "distributed"
	globalJMeterParamPrefix                = "-G"
	standaloneJMeterParamPrefix            = "-J"
	jmxExtension                           = "jmx"
)

// JMeterDRunner runner
type JMeterDRunner struct {
	Params  envs.Params
	Scraper scraper.Scraper
	fs      filesystem.FileSystem
}

// GetType returns runner type
func (r *JMeterDRunner) GetType() runner.Type {
	return runner.TypeMain
}

func NewRunner(ctx context.Context, params envs.Params) (*JMeterDRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &JMeterDRunner{
		Params: params,
		fs:     filesystem.NewOSFileSystem(),
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *JMeterDRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
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

	slavesEnvVariables := slaves.ExtractSlaveEnvVariables(envManager.Variables)
	// creating slaves provided in SLAVES_COUNT env variable
	slavesCount, err := slaves.GetSlavesCount(slavesEnvVariables)
	if err != nil {
		output.PrintLogf("%s Failed to get slaves count %s", ui.IconCross, err)
		return result, errors.Wrap(err, "error getting slaves count")
	}
	mode := jmeterModeStandalone
	if slavesCount > 0 {
		mode = jmeterModeDistributed
	}

	for key, value := range envManager.Variables {
		if err := os.Setenv(key, value.Value); err != nil {
			output.PrintLogf("%s Failed to set env variable %s", ui.IconWarning, key)
		}
	}

	output.PrintLogf("%s Running in %s mode", ui.IconTruck, mode)

	testPath, workingDir, testFile, err := getTestPathAndWorkingDir(r.fs, &execution, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	parentTestFolder := filepath.Dir(testPath)
	// Set env plugin env variable to set custom plugin directory
	// with this path custom plugin will be copied to jmeter's plugin directory
	err = os.Setenv("JMETER_PARENT_TEST_FOLDER", parentTestFolder)
	if err != nil {
		output.PrintLogf("%s Failed to set parent test folder directory %s", ui.IconWarning, parentTestFolder)
	}
	// Add user plugins folder in slaves env variables
	slavesEnvVariables["JMETER_PARENT_TEST_FOLDER"] = testkube.NewBasicVariable("JMETER_PARENT_TEST_FOLDER", parentTestFolder)

	runPath := r.Params.DataDir
	if workingDir != "" {
		runPath = workingDir
	}

	outputDir := filepath.Join(runPath, "output")
	err = os.Setenv("OUTPUT_DIR", outputDir)
	if err != nil {
		output.PrintLogf("%s Failed to set output directory %s", ui.IconWarning, outputDir)
	}
	slavesEnvVariables["OUTPUT_DIR"] = testkube.NewBasicVariable("OUTPUT_DIR", outputDir)

	// recreate output directory with wide permissions so JMeter can create report files
	if err = os.Mkdir(outputDir, 0777); err != nil {
		return *result.Err(errors.Wrapf(err, "error creating directory %s", outputDir)), nil
	}

	jtlPath := filepath.Join(outputDir, "report.jtl")
	reportPath := filepath.Join(outputDir, "report")
	jmeterLogPath := filepath.Join(outputDir, "jmeter.log")
	args := execution.Args
	hasJunit, hasReport, args := prepareArgs(args, testPath, jtlPath, reportPath, jmeterLogPath)

	if mode == jmeterModeDistributed {
		clientSet, err := k8sclient.ConnectToK8s()
		if err != nil {
			return result, err
		}

		slaveMeta, cleanup, err := initSlaves(ctx, clientSet, execution, r.Params, slavesCount, slavesEnvVariables, parentTestFolder, testFile)
		if err != nil {
			return result, err
		}
		defer cleanup()
		args = append(args, fmt.Sprintf("-R %v", slaveMeta.ToIPString()))
	}

	args = injectAndExpandEnvVars(args, nil)
	output.PrintLogf("%s Using arguments: %v", ui.IconWorld, envManager.ObfuscateStringSlice(args))

	entryPoint := getEntryPoint()
	for i := range execution.Command {
		if execution.Command[i] == "<entryPoint>" {
			execution.Command[i] = entryPoint
			break
		}
	}

	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	// run JMeter inside repo directory ignore execution error in case of failed test
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	out, err := executor.Run(runPath, command, envManager, args...)
	if err != nil {
		return *result.Err(errors.Errorf("jmeter run error: %v", err)), nil
	}
	out = envManager.ObfuscateSecrets(out)

	var executionResult testkube.ExecutionResult
	var jtlCheckError error
	if hasJunit && hasReport {
		executionResult, err = processJTLReport(r.fs, jtlPath, out)
		if err != nil {
			jtlCheckError = errors.Wrap(err, "error processing jtl report")
			if errors.Is(err, parser.ErrEmptyReport) {
				errMsg := "JTL report is empty which probably indicates an issue with the test. Check the jmeter.log for more info."
				output.PrintLogf("%s %s", ui.IconCross, errMsg)
				jtlCheckError = errors.New(errMsg)
			}
		}
	} else {
		executionResult = parser.MakeSuccessExecution(out)
	}

	postRunScriptErr := runPostRunScriptIfEnabled(execution, r.Params.WorkingDir)

	scrapeErr := runScraperIfEnabled(ctx, r.Params.ScrapperEnabled, r.Scraper, []string{outputDir}, execution)

	if jtlCheckError != nil || postRunScriptErr != nil || scrapeErr != nil {
		return *result.WithErrors(jtlCheckError, postRunScriptErr, scrapeErr), nil
	}

	return executionResult, nil
}

func initSlaves(
	ctx context.Context,
	clientSet kubernetes.Interface,
	execution testkube.Execution,
	params envs.Params,
	slavesCount int,
	slavesEnvVariables map[string]testkube.Variable,
	parentTestFolder, testFile string,
) (slaveMeta slaves.SlaveMeta, cleanupFunc func() error, err error) {
	slavesEnvVariables["DATA_CONFIG"] = testkube.NewBasicVariable("DATA_CONFIG", parentTestFolder)
	slavesEnvVariables["JMETER_SCRIPT"] = testkube.NewBasicVariable("JMETER_SCRIPT", testFile)
	envVars := slaves.GetRunnerEnvVariables()
	for key, value := range envVars {
		slavesEnvVariables[key] = testkube.NewBasicVariable(key, value)
	}

	slavesConfigs := executor.SlavesConfigs{}
	if err := json.Unmarshal([]byte(params.SlavesConfigs), &slavesConfigs); err != nil {
		return nil, nil, errors.Wrap(err, "error unmarshalling slaves configs")
	}

	slaveClient := slaves.NewClient(clientSet, execution, slavesConfigs, envVars, slavesEnvVariables)
	slaveMeta, err = slaveClient.CreateSlaves(ctx, slavesCount)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	cleanupFunc = func() error {
		return slaveClient.DeleteSlaves(ctx, slaveMeta)
	}
	return slaveMeta, cleanupFunc, nil

}

func prepareArgs(args []string, path, jtlPath, reportPath, jmeterLogPath string) (hasJunit, hasReport bool, result []string) {
	counters := make(map[string]int)
	duplicates := make(map[string]string)
	for _, arg := range args {
		counters[arg] += 1
		if counters[arg] > 1 {
			switch arg {
			case "-t":
				duplicates["<runPath>"] = arg
			case "-l":
				duplicates["<jtlFile>"] = arg
			case "-o":
				duplicates["<reportFile>"] = arg
			case "-j":
				duplicates["<logFile>"] = arg
			}
		}
	}

	for i := len(args) - 1; i >= 0; i-- {
		if arg, ok := duplicates[args[i]]; ok {
			args = append(args[:i], args[i+1:]...)
			if i > 0 {
				i--
				if args[i] == arg {
					args = append(args[:i], args[i+1:]...)
				}
			}
		}
	}

	for i, arg := range args {
		switch arg {
		case "<runPath>":
			args[i] = path
		case "<jtlFile>":
			args[i] = jtlPath
		case "<reportFile>":
			args[i] = reportPath
			hasReport = true
		case "<logFile>":
			args[i] = jmeterLogPath
		case "-l":
			hasJunit = true
		}
	}
	return hasJunit, hasReport, args
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

func processJTLReport(fs filesystem.FileSystem, jtlPath string, resultOutput []byte) (testkube.ExecutionResult, error) {
	var executionResult testkube.ExecutionResult
	output.PrintLogf("%s Getting report %s", ui.IconFile, jtlPath)
	f, err := fs.OpenFileRO(jtlPath)
	if err != nil {
		return testkube.ExecutionResult{}, errors.Wrap(err, "error getting jtl report")
	}
	defer f.Close()

	executionResult, err = parser.ParseJTLReport(f, resultOutput)
	if err != nil {
		return testkube.ExecutionResult{}, err
	}

	return executionResult, nil
}

func runPostRunScriptIfEnabled(execution testkube.Execution, workingDir string) (err error) {
	executePostRunScript := execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping
	if executePostRunScript {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconTruck))

		if err = agent.RunScript(execution.PostRunScript, workingDir); err != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, err)
		}

		output.PrintLogf("%s Post run script execution finished!", ui.IconCheckMark)
	}
	return
}

func runScraperIfEnabled(ctx context.Context, enabled bool, scraper scraper.Scraper, dirs []string, execution testkube.Execution) (err error) {
	if enabled {
		directories := dirs
		var masks []string
		if execution.ArtifactRequest != nil {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
			masks = execution.ArtifactRequest.Masks
		}

		output.PrintLogf("%s Running scraper to scrape for the following directories: %v with masks: %v", ui.IconTruck, directories, masks)
		if err = scraper.Scrape(ctx, directories, masks, execution); err != nil {
			output.PrintLogf("%s Failed to scrape artifacts %s", ui.IconWarning, err)
		}
		output.PrintLogf("%s Tests artifacts scrapped successfully!", ui.IconCheckMark)
	}
	return
}

var _ runner.Runner = &JMeterDRunner{}
