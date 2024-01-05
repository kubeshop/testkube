package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/executor/content"
	"github.com/kubeshop/testkube/pkg/executor/env"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/executor/scraper/factory"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunner(ctx context.Context, params envs.Params) (*TracetestRunner, error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing Runner", ui.IconTruck))

	scraper, err := factory.TryGetScrapper(ctx, params)
	if err != nil {
		return nil, err
	}

	return &TracetestRunner{
		Params:        params,
		Scraper:       scraper,
		coreExecutor:  &tracetestCoreExecutor{},
		cloudExecutor: &tracetestCloudExecutor{},
	}, nil
}

type TracetestRunner struct {
	Params        envs.Params
	Scraper       scraper.Scraper
	coreExecutor  TracetestCLIExecutor
	cloudExecutor TracetestCLIExecutor
}

type TracetestCLIExecutor interface {
	RequiredEnvVars() []string
	HasEnvVarsDefined(*env.Manager) bool
	Execute(*env.Manager, testkube.Execution, string) (model.Result, error)
}

func (r *TracetestRunner) Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	if r.Scraper != nil {
		defer r.Scraper.Close()
	}

	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing test run", ui.IconTruck))

	// Get execution content file path
	testFilePath, _, err := content.GetPathAndWorkingDir(execution.Content, r.Params.DataDir)
	if err != nil {
		outputPkg.PrintLogf("%s Failed to resolve absolute directory for %s, using the path directly", ui.IconWarning, r.Params.DataDir)
	}

	envManager := env.NewManagerWithVars(execution.Variables)
	envManager.GetReferenceVars(envManager.Variables)

	// Get a CLI test executor
	cliExecutor, err := r.getCLIExecutor(envManager)
	if err != nil {
		outputPkg.PrintLogf("%s Failed to get a Tracetest CLI executor %s", ui.IconWarning, err)
		return testkube.ExecutionResult{}, fmt.Errorf("failed to get a Tracetest CLI executor %s", err)
	}

	// Run CLI executor
	cliExecutionResult, err := cliExecutor.Execute(envManager, execution, testFilePath)
	if err != nil {
		result = cliExecutionResult.ToFailedExecutionResult(err)
	} else {
		result = cliExecutionResult.ToSuccessfulExecutionResult()
	}

	var rerr error
	if execution.PostRunScript != "" && execution.ExecutePostRunScriptBeforeScraping {
		outputPkg.PrintLog(fmt.Sprintf("%s Running post run script...", ui.IconCheckMark))

		if rerr = agent.RunScript(execution.PostRunScript, r.Params.WorkingDir); rerr != nil {
			outputPkg.PrintLogf("%s Failed to execute post run script %s", ui.IconWarning, rerr)
		}
	}

	// scrape artifacts first even if there are errors above
	if r.Params.ScrapperEnabled && execution.ArtifactRequest != nil && len(execution.ArtifactRequest.Dirs) != 0 {
		outputPkg.PrintLogf("Scraping directories: %v with masks: %v", execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks)

		if err := r.Scraper.Scrape(ctx, execution.ArtifactRequest.Dirs, execution.ArtifactRequest.Masks, execution); err != nil {
			return testkube.ExecutionResult{}, fmt.Errorf("could not scrape tracetest directories: %w", err)
		}
	}

	if rerr != nil {
		return testkube.ExecutionResult{}, rerr
	}

	return result, nil
}

// GetType returns runner type
func (r *TracetestRunner) GetType() runner.Type {
	return runner.TypeMain
}

func (r *TracetestRunner) getCLIExecutor(envManager *env.Manager) (TracetestCLIExecutor, error) {
	if r.cloudExecutor.HasEnvVarsDefined(envManager) {
		return r.cloudExecutor, nil
	}

	if r.coreExecutor.HasEnvVarsDefined(envManager) {
		return r.coreExecutor, nil
	}

	outputPkg.PrintLogf("%s [TracetestRunner]: Could not find variables to run the test with Tracetest or Tracetest Cloud.", ui.IconCross)
	outputPkg.PrintLogf("%s [TracetestRunner]: Please define the [%s] variables to run a test with Tracetest", ui.IconCross, strings.Join(r.cloudExecutor.RequiredEnvVars(), ", "))
	outputPkg.PrintLogf("%s [TracetestRunner]: Or define the [%s] variables to run a test with Tracetest Core", ui.IconCross, strings.Join(r.coreExecutor.RequiredEnvVars(), ", "))
	return nil, fmt.Errorf("could not find variables to run the test with Tracetest or Tracetest Cloud")
}

// Get variable from EnvManager
func getVariable(envManager *env.Manager, variableName string) (string, error) {
	return getVariableWithWarning(envManager, variableName, true)
}

func getOptionalVariable(envManager *env.Manager, variableName string) (string, error) {
	return getVariableWithWarning(envManager, variableName, false)
}

func getVariableWithWarning(envManager *env.Manager, variableName string, required bool) (string, error) {
	v, ok := envManager.Variables[variableName]

	warningMessage := fmt.Sprintf("%s [TracetestRunner]: %s variable was not found", ui.IconCross, variableName)
	if !required {
		warningMessage = fmt.Sprintf("[TracetestRunner]: %s variable was not found, assuming empty value", variableName)
	}

	if !ok {
		outputPkg.PrintLog(warningMessage)
		return "", fmt.Errorf(variableName + " variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}
