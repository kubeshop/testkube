package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewArtilleryRunner creates a new Testkube test runner for Artillery tests
func NewArtilleryRunner(ctx context.Context, params envs.Params) (*ArtilleryRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &ArtilleryRunner{
		Params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// ArtilleryRunner ...
type ArtilleryRunner struct {
	Params  envs.Params
	Scraper scraper.Scraper
}

var _ runner.Runner = &ArtilleryRunner{}

// Run ...
func (r *ArtilleryRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)
	// make some validation
	err = r.Validate(execution)
	if err != nil {
		return result, err
	}

	path, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	testDir, _ := filepath.Split(path)
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	var envFile string
	if len(envManager.Variables) != 0 {
		envFile, err = CreateEnvFile(envManager.Variables)
		if err != nil {
			return result, err
		}

		defer os.Remove(envFile)
		output.PrintEvent("created dotenv file", envFile)
	}

	// artillery test result output file
	testReportFile := filepath.Join(testDir, "test-report.json")

	args := execution.Args
	hasJunit := false
	hasReport := false
	for i := len(args) - 1; i >= 0; i-- {
		if envFile == "" && (args[i] == "--dotenv" || args[i] == "<envFile>") {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<envFile>" {
			args[i] = envFile
		}

		if args[i] == "<reportFile>" {
			args[i] = testReportFile
			hasReport = true
		}

		if args[i] == "<runPath>" {
			args[i] = path
		}

		if args[i] == "-o" {
			hasJunit = true
		}

		args[i] = os.ExpandEnv(args[i])
	}

	runPath := testDir
	if workingDir != "" {
		runPath = workingDir
	}

	// run executor
	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	out, runerr := executor.Run(runPath, command, envManager, args...)

	out = envManager.ObfuscateSecrets(out)

	if hasJunit && hasReport {
		var artilleryResult ArtilleryExecutionResult
		artilleryResult, err = r.GetArtilleryExecutionResult(testReportFile, out)
		if err != nil {
			return *result.WithErrors(runerr, errors.Errorf("failed to get test execution results")), err
		}

		result = MapTestSummaryToResults(artilleryResult)
	} else {
		result = makeSuccessExecution(out)
	}

	output.PrintLog(fmt.Sprintf("%s Mapped test summary to Execution Results...", ui.IconCheckMark))

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	if r.Params.ScrapperEnabled {
		directories := []string{
			testReportFile,
		}
		var masks []string
		if execution.ArtifactRequest != nil {
			directories = append(directories, execution.ArtifactRequest.Dirs...)
			masks = execution.ArtifactRequest.Masks
		}

		err = r.Scraper.Scrape(ctx, directories, masks, execution)
		if err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts for Artillery executor")
		}
	}

	// return ExecutionResult
	return *result.WithErrors(err, rerr), nil
}

// GetType returns runner type
func (r *ArtilleryRunner) GetType() runner.Type {
	return runner.TypeMain
}

func CreateEnvFile(vars map[string]testkube.Variable) (string, error) {
	var envVars []byte
	for _, v := range vars {
		envVars = append(envVars, []byte(fmt.Sprintf("%s=%s\n", v.Name, v.Value))...)
	}
	envFile, err := os.CreateTemp("/tmp", "")
	if err != nil {
		return "", errors.Errorf("could not create dotenv file: %v", err)
	}
	defer envFile.Close()

	if _, err := envFile.Write(envVars); err != nil {
		return "", errors.Errorf("could not write dotenv file: %v", err)
	}

	return envFile.Name(), nil
}
