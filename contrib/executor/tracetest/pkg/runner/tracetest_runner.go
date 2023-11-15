package runner

import (
	"strings"

	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/model"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	TRACETEST_TOKEN_VAR        = "TRACETEST_TOKEN"
	TRACETEST_ORGANIZATION_VAR = "TRACETEST_ORGANIZATION"
	TRACETEST_ENVIRONMENT_VAR  = "TRACETEST_ENVIRONMENT"
	TRACETEST_CLOUD_URL        = "https://app.tracetest.io"
)

type tracetestCloudExecutor struct{}

var _ TracetestCLIExecutor = (*tracetestCloudExecutor)(nil)

func (e *tracetestCloudExecutor) RequiredEnvVars() []string {
	return []string{TRACETEST_TOKEN_VAR, TRACETEST_ORGANIZATION_VAR, TRACETEST_ENVIRONMENT_VAR}
}

func (e *tracetestCloudExecutor) HasEnvVarsDefined(envManager *env.Manager) bool {
	_, hasTokenVar := envManager.Variables[TRACETEST_TOKEN_VAR]
	_, hasOrganizationVar := envManager.Variables[TRACETEST_ORGANIZATION_VAR]
	_, hasEnvironmentVar := envManager.Variables[TRACETEST_ENVIRONMENT_VAR]

	return hasTokenVar && hasOrganizationVar && hasEnvironmentVar
}

func (e *tracetestCloudExecutor) Execute(envManager *env.Manager, execution testkube.Execution, testFilePath string) (model.Result, error) {
	tracetestToken, err := getVariable(envManager, TRACETEST_TOKEN_VAR)
	if err != nil {
		return model.Result{}, err
	}

	tracetestOrganization, err := getVariable(envManager, TRACETEST_ORGANIZATION_VAR)
	if err != nil {
		return model.Result{}, err
	}

	tracetestEnvironment, err := getVariable(envManager, TRACETEST_ENVIRONMENT_VAR)
	if err != nil {
		return model.Result{}, err
	}

	// setup config with API key
	output.PrintLogf("%s Configuring Tracetest CLI with Token", ui.IconWorld)

	configArgs := []string{
		"configure", "--token", tracetestToken, "--organization", tracetestOrganization, "--environment", tracetestEnvironment,
	}

	output.PrintLogf("%s Using arguments to configure CLI: %v", ui.IconWorld, envManager.ObfuscateStringSlice(configArgs))
	configCommand, configArgs := executor.MergeCommandAndArgs(execution.Command, configArgs)

	output.PrintLogf("%s Configure command %s %s", ui.IconRocket, configCommand, strings.Join(envManager.ObfuscateStringSlice(configArgs), " "))
	_, err = executor.Run("", configCommand, envManager, configArgs...)

	if err != nil {
		outputPkg.PrintLogf("%s Failed to configure Tracetest CLI %v", ui.IconCross, err)
		return model.Result{}, err
	}

	// Prepare args for test run command
	// (since each strategy implementation has its own set of params, we are explicitly ignoring the execution.Args field)
	runTestArgs := []string{
		"run", "test", "--file", testFilePath, "--output", "pretty",
	}

	output.PrintLogf("%s Using arguments to run test: %v", ui.IconWorld, envManager.ObfuscateStringSlice(runTestArgs))
	runTestCommand, runTestArgs := executor.MergeCommandAndArgs(execution.Command, runTestArgs)

	// Run tracetest test from definition file
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, runTestCommand, strings.Join(envManager.ObfuscateStringSlice(runTestArgs), " "))
	output, err := executor.Run("", runTestCommand, envManager, runTestArgs...)

	result := model.Result{
		Output:         string(output),
		ServerEndpoint: TRACETEST_CLOUD_URL,
		OutputEndpoint: "",
	}

	return result, err
}
