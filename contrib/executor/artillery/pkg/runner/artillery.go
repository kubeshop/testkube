package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewArtilleryRunner creates a new Testkube test runner for Artillery tests
func NewArtilleryRunner(ctx context.Context, params envs.Params) (*ArtilleryRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &ArtilleryRunner{
		Fetcher: content.NewFetcher(""),
		Params:  params,
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
	Fetcher content.ContentFetcher
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
	if r.Params.GitUsername != "" || r.Params.GitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = r.Params.GitUsername
			execution.Content.Repository.Token = r.Params.GitToken
		}
	}

	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return result, errors.Errorf("could not fetch test content: %v", err)
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
	for i := len(args); i >= 0; i-- {
		if envFile == "" && (args[i] == "dotenv" || args[i] == "<envFile>") {
			args = append(args[:i], args[i+1:]...)
			continue
		}

		if args[i] == "<envFile>" {
			args[i] = envFile
		}

		if args[i] == "<reportFile>" {
			args[i] = testReportFile
		}

		if args[i] == "<runPath>" {
			args[i] = path
		}
	}

	runPath := testDir
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	// run executor
	command := strings.Join(execution.Command, " ")
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(args, " "))
	out, rerr := executor.Run(runPath, command, envManager, args...)

	out = envManager.ObfuscateSecrets(out)

	var artilleryResult ArtilleryExecutionResult
	artilleryResult, err = r.GetArtilleryExecutionResult(testReportFile, out)
	if err != nil {
		return *result.WithErrors(rerr, errors.Errorf("failed to get test execution results")), err
	}

	result = MapTestSummaryToResults(artilleryResult)
	output.PrintLog(fmt.Sprintf("%s Mapped test summary to Execution Results...", ui.IconCheckMark))

	if r.Params.ScrapperEnabled {
		directories := []string{
			testReportFile,
		}
		err = r.Scraper.Scrape(ctx, directories, execution)
		if err != nil {
			return *result.Err(err), errors.Wrap(err, "error scraping artifacts for Artillery executor")
		}
	}

	// return ExecutionResult
	return *result.WithErrors(err), nil
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
