package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kubepug "github.com/rikatz/kubepug/pkg/results"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/ui"
)

type Params struct {
	DataDir string // RUNNER_DATADIR
}

func NewRunner() *KubepugRunner {
	output.PrintLog(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))

	output.PrintLog(fmt.Sprintf("%s Reading environment variables...", ui.IconWorld))
	params := Params{
		DataDir: os.Getenv("RUNNER_DATADIR"),
	}
	output.PrintLog(fmt.Sprintf("%s Environment variables read successfully", ui.IconCheckMark))
	output.PrintLog(fmt.Sprintf("RUNNER_DATADIR=\"%s\"", params.DataDir))

	return &KubepugRunner{
		Fetcher: content.NewFetcher(""),
		params:  params,
	}
}

// KubepugRunner runs kubepug against cluster
type KubepugRunner struct {
	Fetcher content.ContentFetcher
	params  Params
}

// Run runs the kubepug executable and parses it's output to be Testkube-compatible
func (r *KubepugRunner) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
	output.PrintLog(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))

	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return testkube.ExecutionResult{}, fmt.Errorf("could not get content: %w", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return testkube.ExecutionResult{}, err
	}

	if !fileInfo.IsDir() {
		output.PrintLog(fmt.Sprintf("%s Using single file: %v", ui.IconFile, execution))
	}

	if fileInfo.IsDir() {
		output.PrintLog(fmt.Sprintf("%s Using dir: %v", ui.IconFile, execution))
	}

	args, err := buildArgs(execution.Args, path)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Could not build up parameters: %s", ui.IconCross, err.Error()))
		return testkube.ExecutionResult{}, fmt.Errorf("could not build up parameters: %w", err)
	}

	output.PrintLog(fmt.Sprintf("%s Running kubepug with arguments: %v", ui.IconWorld, args))
	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	runPath := ""
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	out, err := executor.Run(runPath, "kubepug", envManager, args...)
	out = envManager.ObfuscateSecrets(out)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Could not execute kubepug: %s", ui.IconCross, err.Error()))
		return testkube.ExecutionResult{}, fmt.Errorf("could not execute kubepug: %w", err)
	}

	var kubepugResult kubepug.Result
	err = json.Unmarshal(out, &kubepugResult)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s could not unmarshal kubepug execution result: %s", ui.IconCross, err.Error()))
		return testkube.ExecutionResult{}, fmt.Errorf("could not unmarshal kubepug execution result: %w", err)
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
			Name:         api.Name,
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
		output.PrintLog(fmt.Sprintf("%s No deleted APIs found", ui.IconCheckMark))
		return step
	}

	step.Status = "failed"
	output.PrintLog(fmt.Sprintf("%s Found deleted APIs: %v", ui.IconCross, r.DeletedAPIs))
	for _, api := range r.DeletedAPIs {
		step.AssertionResults = append(step.AssertionResults, testkube.AssertionResult{
			Name:         api.Name,
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
	for _, a := range args {
		if strings.Contains(a, "--format") {
			return []string{}, fmt.Errorf("the Testkube Kubepug executor does not accept the \"--format\" parameter: %s", a)
		}
		if strings.Contains(a, "--input-file") {
			return []string{}, fmt.Errorf("the Testkube Kubepug executor does not accept the \"--input-file\" parameter: %s", a)
		}
	}
	return append(args, "--format=json", "--input-file", inputPath), nil
}

// GetType returns runner type
func (r *KubepugRunner) GetType() runner.Type {
	return runner.TypeMain
}
