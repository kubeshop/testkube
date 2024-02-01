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
	jmeterModeStandalone  JMeterMode = "standalone"
	jmeterModeDistributed JMeterMode = "distributed"
	jmxExtension                     = "jmx"
	jmeterTestFileFlag               = "-t"
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

	runPath := workingDir

	outputDir := ""
	if envVar, ok := envManager.Variables["RUNNER_ARTIFACTS_DIR"]; ok {
		outputDir = envVar.Value
	}

	if outputDir == "" {
		outputDir = filepath.Join(runPath, "output")
		err = os.Setenv("RUNNER_ARTIFACTS_DIR", outputDir)
		if err != nil {
			output.PrintLogf("%s Failed to set output directory %s", ui.IconWarning, outputDir)
		}
	}

	// recreate output directory with wide permissions so JMeter can create report files
	if err = os.Mkdir(outputDir, 0777); err != nil {
		return *result.Err(errors.Wrapf(err, "error creating directory %s", outputDir)), nil
	}

	jtlPath := filepath.Join(outputDir, "report.jtl")
	reportPath := filepath.Join(outputDir, "report")
	jmeterLogPath := filepath.Join(outputDir, "jmeter.log")
	args := mergeDuplicatedArgs(removeDuplicatedArgs(execution.Args))
	hasJunit, hasReport := prepareArgs(args, testPath, jtlPath, reportPath, jmeterLogPath)

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

	// TODO: this is a workaround, the check should be ideally performed in the getTestPathAndWorkingDir function
	if err := checkIfTestFileExists(r.fs, args, workingDir); err != nil {
		output.PrintLogf("%s  Error validating test file exists: %v", ui.IconCross, err.Error())
		return result, errors.WithStack(err)
	}

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
	if hasJunit && hasReport {
		executionResult, err = processJTLReport(r.fs, jtlPath, out)
		if err != nil {
			if errors.Is(err, parser.ErrEmptyReport) {
				output.PrintLogf("%s JTL report is empty which probably indicates an issue with the test. Check the jmeter.log for more info.", ui.IconCross)
			}
			return *result.Err(errors.Wrap(err, "error processing jtl report")), nil
		}
	} else {
		executionResult = parser.MakeSuccessExecution(out)
	}

	output.PrintLogf("%s Mapped JMeter results to Execution Results...", ui.IconCheckMark)

	postRunScriptErr := runPostRunScriptIfEnabled(execution, r.Params.WorkingDir)

	scrapeErr := runScraperIfEnabled(ctx, r.Params.ScrapperEnabled, r.Scraper, []string{outputDir}, execution)

	if postRunScriptErr != nil || scrapeErr != nil {
		return *result.WithErrors(postRunScriptErr, scrapeErr), nil
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

func checkIfTestFileExists(fs filesystem.FileSystem, args []string, workingDir string) error {
	if len(args) == 0 {
		return errors.New("no arguments provided")
	}
	testParamValue, err := getParamValue(args, jmeterTestFileFlag)
	if err != nil {
		return errors.Wrapf(err, "error extracting value for %s flag", jmeterTestFileFlag)
	}
	if !filepath.IsAbs(testParamValue) {
		testParamValue = filepath.Join(workingDir, testParamValue)
	}
	info, err := fs.Stat(testParamValue)
	if err != nil {
		return errors.WithStack(err)
	}
	if info.IsDir() {
		return errors.Errorf("test file %s is a directory", testParamValue)
	}

	return nil
}

func removeDuplicatedArgs(args []string) []string {
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

	return args
}

func mergeDuplicatedArgs(args []string) []string {
	// -n is mandatory regardless of args mode
	args = append(args, "-n")
	allowed := map[string]int{
		"-e": 0,
		"-n": 0,
	}

	for i := len(args) - 1; i >= 0; i-- {
		if counter, ok := allowed[args[i]]; ok {
			allowed[args[i]]++
			if counter == 0 {
				continue
			}

			args = append(args[:i], args[i+1:]...)
		}
	}

	return args
}

func prepareArgs(args []string, path, jtlPath, reportPath, jmeterLogPath string) (hasJunit, hasReport bool) {
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
	return hasJunit, hasReport
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
			if len(execution.ArtifactRequest.Dirs) != 0 {
				directories = execution.ArtifactRequest.Dirs
			}

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
