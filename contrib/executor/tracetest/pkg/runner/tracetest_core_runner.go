package runner

import (
	"strings"

	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/env"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	TRACETEST_ENDPOINT_VAR        = "TRACETEST_ENDPOINT"
	TRACETEST_OUTPUT_ENDPOINT_VAR = "TRACETEST_OUTPUT_ENDPOINT"
)

type tracetestCoreExecutor struct{}

var _ TracetestCLIExecutor = (*tracetestCoreExecutor)(nil)

func (e *tracetestCoreExecutor) RequiredEnvVars() []string {
	return []string{TRACETEST_ENDPOINT_VAR}
}

func (e *tracetestCoreExecutor) HasEnvVarsDefined(envManager *env.Manager) bool {
	_, hasEndpointVar := envManager.Variables[TRACETEST_ENDPOINT_VAR]
	return hasEndpointVar
}

func (e *tracetestCoreExecutor) Execute(envManager *env.Manager, execution testkube.Execution, testFilePath string) (model.Result, error) {
	// Get TRACETEST_ENDPOINT from execution variables
	tracetestEndpoint, err := getVariable(envManager, TRACETEST_ENDPOINT_VAR)
	if err != nil {
		return model.Result{}, err
	}

	// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
	tracetestOutputEndpoint, _ := getOptionalVariable(envManager, TRACETEST_OUTPUT_ENDPOINT_VAR)

	// Prepare args for test run command
	// (since each strategy implementation has its own set of params, we are explicitly ignoring the execution.Args field)
	args := []string{
		"run", "test", "--server-url", tracetestEndpoint, "--file", testFilePath, "--output", "pretty",
	}

	output.PrintLogf("%s Using arguments: %v", ui.IconWorld, envManager.ObfuscateStringSlice(args))

	command, args := executor.MergeCommandAndArgs(execution.Command, args)

	// Run tracetest test from test file
	output.PrintLogf("%s Test run command %s %s", ui.IconRocket, command, strings.Join(envManager.ObfuscateStringSlice(args), " "))
	output, err := executor.Run("", command, envManager, args...)

	result := model.Result{
		Output:         string(output),
		ServerEndpoint: tracetestEndpoint,
		OutputEndpoint: tracetestOutputEndpoint,
	}

	return result, err
}
