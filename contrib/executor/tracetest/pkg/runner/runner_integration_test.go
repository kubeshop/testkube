package runner_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/command"
	"github.com/kubeshop/testkube/contrib/executor/tracetest/pkg/runner"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func startTracetestEnvironment(t *testing.T, envType string) {
	currentDir := getExecutingDir()
	fullDockerComposeFilePath := fmt.Sprintf("%s/../testing/docker-compose-tracetest-%s.yaml", currentDir, envType)

	result, err := command.Run("docker", "compose", "--file", fullDockerComposeFilePath, "up", "--detach")
	require.NoErrorf(t, err, "error when starting tracetest core env: %s", string(result))

	// pokeshop takes some time to warm up, so we need to wait a little bit before doing tests
	time.Sleep(10 * time.Second)
}

func stopTracetestEnvironment(t *testing.T, envType string) {
	currentDir := getExecutingDir()
	fullDockerComposeFilePath := fmt.Sprintf("%s/../testing/docker-compose-tracetest-%s.yaml", currentDir, envType)

	result, err := command.Run("docker", "compose", "--file", fullDockerComposeFilePath, "rm", "--force", "--volumes", "--stop")
	require.NoErrorf(t, err, "error when stopping tracetest core env: %s", string(result))
}

func TestRunFiles_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()

	t.Run("Run tracetest core with no variables", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "*")
		require.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := runner.NewRunner(context.Background(), params)
		require.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "tracetest/test"
		execution.Command = []string{
			"tracetest",
		}
		execution.Args = []string{}
		writeTestContent(t, tempDir, "../testing/tracetest-test-script.yaml")

		// when
		_, err = runner.Run(ctx, *execution)

		// then
		require.ErrorContains(t, err, "failed to get a Tracetest CLI executor")
	})

	t.Run("Run tracetest core with simple script", func(t *testing.T) {
		envType := "core"

		startTracetestEnvironment(t, envType)
		defer stopTracetestEnvironment(t, envType)

		tempDir, err := os.MkdirTemp("", "*")
		require.NoErrorf(t, err, "failed to create temp dir: %v", err)
		defer os.RemoveAll(tempDir)

		params := envs.Params{DataDir: tempDir}
		runner, err := runner.NewRunner(context.Background(), params)
		require.NoError(t, err)

		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "tracetest/test"
		execution.Command = []string{
			"tracetest",
		}
		execution.Variables = map[string]testkube.Variable{
			"TRACETEST_ENDPOINT": {
				Name:  "TRACETEST_ENDPOINT",
				Value: "http://localhost:11633",
				Type_: testkube.VariableTypeBasic,
			},
		}
		execution.Args = []string{}
		writeTestContent(t, tempDir, "../testing/tracetest-test-script.yaml")

		// when
		result, err := runner.Run(ctx, *execution)

		// then
		require.NoError(t, err)
		require.Equal(t, testkube.ExecutionStatusPassed, result.Status)
	})

	// t.Run("Run tracetest cloud with simple script", func(t *testing.T) {
	// 	envType := "cloud"

	// 	os.Setenv("TRACETEST_API_KEY", "some_api_key")

	// 	startTracetestEnvironment(t, envType)
	// 	defer stopTracetestEnvironment(t, envType)

	// 	tempDir, err := os.MkdirTemp("", "*")
	// 	require.NoErrorf(t, err, "failed to create temp dir: %v", err)
	// 	defer os.RemoveAll(tempDir)

	// 	params := envs.Params{DataDir: tempDir}
	// 	runner, err := runner.NewRunner(context.Background(), params)
	// 	require.NoError(t, err)

	// 	execution := testkube.NewQueuedExecution()
	// 	execution.Content = testkube.NewStringTestContent("")
	// 	execution.TestType = "tracetest/test"
	// 	execution.Command = []string{
	// 		"tracetest",
	// 	}
	// 	execution.Variables = map[string]testkube.Variable{
	// 		"TRACETEST_TOKEN": {
	// 			Name:  "TRACETEST_TOKEN",
	// 			Value: "some_token",
	// 			Type_: testkube.VariableTypeBasic,
	// 		},
	// 		"TRACETEST_ORGANIZATION": {
	// 			Name:  "TRACETEST_ORGANIZATION",
	// 			Value: "some_token",
	// 			Type_: testkube.VariableTypeBasic,
	// 		},
	// 		"TRACETEST_ENVIRONMENT": {
	// 			Name:  "TRACETEST_ENVIRONMENT",
	// 			Value: "some_token",
	// 			Type_: testkube.VariableTypeBasic,
	// 		},
	// 	}
	// 	execution.Args = []string{}
	// 	writeTestContent(t, tempDir, "../testing/tracetest-test-script.yaml")

	// 	// when
	// 	result, err := runner.Run(ctx, *execution)

	// 	// then
	// 	require.NoError(t, err)
	// 	require.Equal(t, testkube.ExecutionStatusPassed, result.Status)
	// })
}

func writeTestContent(t *testing.T, dir string, testScriptPath string) {
	currentDir := getExecutingDir()
	fullTestScriptPath := fmt.Sprintf("%s/%s", currentDir, testScriptPath)

	testScript, err := os.ReadFile(fullTestScriptPath)
	if err != nil {
		assert.FailNow(t, "Unable to read tracetest test script")
	}

	err = os.WriteFile(filepath.Join(dir, "test-content"), testScript, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write tracetest runner test content file")
	}
}

func getExecutingDir() string {
	_, filename, _, _ := runtime.Caller(0) // get file of the getExecutingDir caller
	return path.Dir(filename)
}
