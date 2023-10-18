package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

const TRACETEST_ENDPOINT_VAR = "TRACETEST_ENDPOINT"
const TRACETEST_OUTPUT_ENDPOINT_VAR = "TRACETEST_OUTPUT_ENDPOINT"

func NewRunner(ctx context.Context, params envs.Params) (*TracetestRunner, error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing Runner", ui.IconTruck))

	scraper, err := factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return &TracetestRunner{
		Params:  params,
		Scraper: scraper,
	}, nil
}

type TracetestRunner struct {
	Params  envs.Params
	Scraper scraper.Scraper
}

func (r *TracetestRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing test run", ui.IconTruck))

	// Get execution content file path
	path, workingDir, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		outputPkg.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	// Get TRACETEST_ENDPOINT from execution variables
	te, err := getTracetestEndpointFromVars(envManager)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: TRACETEST_ENDPOINT variable was not found: %v", ui.IconCross, err))
		return result, err
	}

	// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
	toe, err := getTracetestOutputEndpointFromVars(envManager)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: error on processing variables: %v", ui.IconCross, err))
		return result, err
	}

	// Prepare args for test run command
	args, err := buildArgs(execution.Args, te, path)
	if err != nil {
		output.PrintLogf("%s Could not build up parameters: %s", ui.IconCross, err.Error())
		return testkube.ExecutionResult{}, fmt.Errorf("could not build up parameters: %w", err)
	}
	output.PrintLogf("%s Using arguments: %v", ui.IconWorld, args)

	command, args := executor.MergeCommandAndArgs(execution.Command, args)

	// Run tracetest test from definition file
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(args, " "))
	output, err := executor.Run("", command, envManager, args...)
	runResult := model.Result{Output: string(output), ServerEndpoint: te, OutputEndpoint: toe}

	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		outputPkg.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if err = agent.RunScript(execution.PostRunScript, workingDir); err != nil {
			outputPkg.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, err)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		outputPkg.PrintLogf("Scraping directories: %v", execution.ArtifactRequest.Dirs)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution); err != nil {
			return testkube.ExecutionResult{}, fmt.Errorf("could not scrape tracetest directories: %w", err)
		}
	}

	if err != nil {
		result.ErrorMessage = runResult.GetOutput()
		result.Output = runResult.GetOutput()
		result.Status = testkube.ExecutionStatusFailed
		return result, nil
	}

	result.Output = runResult.GetOutput()
	result.Status = runResult.GetStatus()

	return result, nil
}

// GetType returns runner type
func (r *TracetestRunner) GetType() runner.Type {
	return runner.TypeMain
}

// Get TRACETEST_ENDPOINT from execution variables
func getTracetestEndpointFromVars(envManager *env.Manager) (string, error) {
	v, ok := envManager.Variables[TRACETEST_ENDPOINT_VAR]
	if !ok {
		return "", fmt.Errorf("TRACETEST_ENDPOINT variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}

// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
func getTracetestOutputEndpointFromVars(envManager *env.Manager) (string, error) {
	v, ok := envManager.Variables[TRACETEST_OUTPUT_ENDPOINT_VAR]
	if !ok {
		return "", fmt.Errorf("TRACETEST_OUTPUT_ENDPOINT variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}

// buildArgs builds up the arguments for
func buildArgs(args []string, tracetestEndpoint string, inputPath string) ([]string, error) {
	for i := range args {
		if args[i] == "<tracetestServer>" {
			args[i] = tracetestEndpoint
		}
		if args[i] == "<filePath>" {
			args[i] = inputPath
		}
	}
	return args, nil
}
