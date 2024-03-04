package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	kubepug "github.com/kubepug/kubepug/pkg/results"

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

func NewRunner(ctx context.Context, params envs.Params) (*KubepugRunner, error) {
	output.PrintLogf("%s Preparing test runner", ui.IconTruck)

	var err error
	r := &KubepugRunner{
		params: params,
	}

	r.Scraper, err = factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// KubepugRunner runs kubepug against cluster
type KubepugRunner struct {
	params  envs.Params
	Scraper scraper.Scraper
}

var _ runner.Runner = &KubepugRunner{}

// Run runs the kubepug executable and parses it's output to be Testkube-compatible
func (r *KubepugRunner) Run(ctx context.Context, execution testkube.Execution) (testkube.ExecutionResult, error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}
	output.PrintLogf("%s Preparing for test run", ui.IconTruck)

	path, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.params.DataDir)
	if err != nil {
		output.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.params.DataDir)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return testkube.ExecutionResult{}, err
	}

	if !fileInfo.IsDir() {
		output.PrintLogf("%s Using single file: %v", ui.IconFile, execution)
	}

	if fileInfo.IsDir() {
		output.PrintLogf("%s Using dir: %v", ui.IconFile, execution)
	}

	args, err := buildArgs(execution.Args, path)
	if err != nil {
		output.PrintLogf("%s Could not build up parameters: %s", ui.IconCross, err.Error())
		return testkube.ExecutionResult{}, fmt.Errorf("could not build up parameters: %w", err)
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)
	output.PrintLogf("%s Running kubepug with arguments: %v", ui.IconWorld, envManager.ObfuscateStringSlice(args))

	runPath := workingDir
	command, args := executor.MergeCommandAndArgs(execution.Command, args)
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	out, err := executor.Run(runPath, command, envManager, args...)
	out = envManager.ObfuscateSecrets(out)
	if err != nil {
		output.PrintLogf("%s Could not execute kubepug: %s", ui.IconCross, err.Error())
		return testkube.ExecutionResult{}, fmt.Errorf("could not execute kubepug: %w", err)
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		output.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.params.WorkingDir); rerr != nil {
			output.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		output.PrintLogf("Scraping directories: %v with masks: %v", execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks, execution); err != nil {
			return testkube.ExecutionResult{}, fmt.Errorf("could not scrape kubepug directories: %w", err)
		}
	}

	var kubepugResult kubepug.Result
	err = json.Unmarshal(out, &kubepugResult)
	if err != nil {
		output.PrintLogf("%s could not unmarshal kubepug execution result: %s", ui.IconCross, err.Error())
		return testkube.ExecutionResult{}, fmt.Errorf("could not unmarshal kubepug execution result: %w", err)
	}

	if rerr != nil {
		return testkube.ExecutionResult{}, rerr
	}

	deprecatedAPIstep := createDeprecatedAPIsStep(kubepugResult)
	deletedAPIstep := createDeletedAPIsStep(kubepugResult)
	return testkube.ExecutionResult{
		Status: getResultStatus(kubepugResult),
		Output: string(out),
		Steps: []testkube.ExecutionStepResult{
			deprecatedAPIstep,
			deletedAPIstep,
		},
	}, nil
}

// createDeprecatedAPIsStep checks the kubepug output for deprecated APIs and converts them to Testkube step result
func createDeprecatedAPIsStep(r kubepug.Result) testkube.ExecutionStepResult {
	step := testkube.ExecutionStepResult{
		Name: "Deprecated APIs",
	}

	if len(r.DeprecatedAPIs) == 0 {
		step.Status = "passed"
		output.PrintLog(fmt.Sprintf("%s No deprecated APIs found", ui.IconCheckMark))
		return step
	}

	step.Status = "failed"
	output.PrintLog(fmt.Sprintf("%s Found deprecated APIs: %v", ui.IconCross, r.DeletedAPIs))
	for _, api := range r.DeprecatedAPIs {
		step.AssertionResults = append(step.AssertionResults, testkube.AssertionResult{
			Name:         api.Kind,
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("Deprecated API:\n %v", api),
		})
	}

	return step
}

// createDeletedAPISstep checks the kubepug output for deleted APIs and converts them to Testkube step result
func createDeletedAPIsStep(r kubepug.Result) testkube.ExecutionStepResult {
	step := testkube.ExecutionStepResult{
		Name: "Deleted APIs",
	}

	if len(r.DeletedAPIs) == 0 {
		step.Status = "passed"
		output.PrintLogf("%s No deleted APIs found", ui.IconCheckMark)
		return step
	}

	step.Status = "failed"
	output.PrintLogf("%s Found deleted APIs: %v", ui.IconCross, r.DeletedAPIs)
	for _, api := range r.DeletedAPIs {
		step.AssertionResults = append(step.AssertionResults, testkube.AssertionResult{
			Name:         api.Kind,
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("Deleted API:\n %v", api),
		})
	}

	return step
}

// getResultStatus calculates the final result status
func getResultStatus(r kubepug.Result) *testkube.ExecutionStatus {
	if len(r.DeletedAPIs) == 0 && len(r.DeprecatedAPIs) == 0 {
		return testkube.ExecutionStatusPassed
	}
	return testkube.ExecutionStatusFailed
}

// buildArgs builds up the arguments for
func buildArgs(args []string, inputPath string) ([]string, error) {
	for i := range args {
		if args[i] == "<runPath>" {
			args[i] = inputPath
		}

		args[i] = os.ExpandEnv(args[i])
	}
	return args, nil
}

// GetType returns runner type
func (r *KubepugRunner) GetType() runner.Type {
	return runner.TypeMain
}
