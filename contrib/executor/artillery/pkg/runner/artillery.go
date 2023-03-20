package runner

import (
	"fmt"
	"os"
	"path/filepath"

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
func NewArtilleryRunner() (*ArtilleryRunner, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))
	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Artillery runner variables: %w", err)
	}

	return &ArtilleryRunner{
		Fetcher: content.NewFetcher(""),
		Params:  params,
		Scraper: scraper.NewMinioScraper(
			params.Endpoint,
			params.AccessKeyID,
			params.SecretAccessKey,
			params.Location,
			params.Token,
			params.Bucket,
			params.Ssl,
		),
	}, err
}

// ArtilleryRunner ...
type ArtilleryRunner struct {
	Params  envs.Params
	Fetcher content.ContentFetcher
	Scraper scraper.Scraper
}

// Run ...
func (r *ArtilleryRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))
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
		return result, fmt.Errorf("could not fetch test content: %w", err)
	}

	testDir, _ := filepath.Split(path)
	args := []string{"run", path}
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	if len(envManager.Variables) != 0 {
		envFile, err := CreateEnvFile(envManager.Variables)
		if err != nil {
			return result, err
		}

		defer os.Remove(envFile)
		args = append(args, "--dotenv", envFile)
		output.PrintEvent("created dotenv file", envFile)
	}

	// artillery test result output file
	testReportFile := filepath.Join(testDir, "test-report.json")

	// append args from execution
	args = append(args, "-o", testReportFile)

	args = append(args, execution.Args...)

	runPath := testDir
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	// run executor
	out, rerr := executor.Run(runPath, "artillery", envManager, args...)

	out = envManager.ObfuscateSecrets(out)

	var artilleryResult ArtilleryExecutionResult
	artilleryResult, err = r.GetArtilleryExecutionResult(testReportFile, out)
	if err != nil {
		return *result.WithErrors(rerr, fmt.Errorf("failed to get test execution results")), err
	}

	result = MapTestSummaryToResults(artilleryResult)
	output.PrintLog(fmt.Sprintf("%s Mapped test summary to Execution Results...", ui.IconCheckMark))

	if r.Params.ScrapperEnabled && r.Scraper != nil {
		artifacts := []string{
			testReportFile,
		}
		err = r.Scraper.Scrape(execution.Id, artifacts)
		if err != nil {
			return *result.WithErrors(fmt.Errorf("scrape artifacts error: %w", err)), nil
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
	envVars := []byte{}
	for _, v := range vars {
		envVars = append(envVars, []byte(fmt.Sprintf("%s=%s\n", v.Name, v.Value))...)
	}
	envFile, err := os.CreateTemp("/tmp", "")
	if err != nil {
		return "", fmt.Errorf("could not create dotenv file: %w", err)
	}
	defer envFile.Close()

	if _, err := envFile.Write(envVars); err != nil {
		return "", fmt.Errorf("could not write dotenv file: %w", err)
	}

	return envFile.Name(), nil
}
