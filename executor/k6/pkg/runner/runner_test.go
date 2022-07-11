package runner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestRunFiles(t *testing.T) {
	// setup
	tempDir := os.TempDir()
	os.Setenv("RUNNER_DATADIR", tempDir)

	t.Run("Run k6 with simple script", func(t *testing.T) {
		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		writeTestContent(t, tempDir, "../../examples/k6-test-script.js")

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Run k6 with arguments and simple script", func(t *testing.T) {
		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/run"
		execution.Args = []string{"--vus", "2", "--duration", "1s"}
		writeTestContent(t, tempDir, "../../examples/k6-test-script.js")

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Run k6 with ENV variables and script", func(t *testing.T) {
		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Envs = map[string]string{"TARGET_HOSTNAME": "kubeshop.github.io"}
		writeTestContent(t, tempDir, "../../examples/k6-test-environment.js")

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunAdvanced(t *testing.T) {
	// setup
	tempDir := os.TempDir()
	os.Setenv("RUNNER_DATADIR", tempDir)

	t.Run("Run k6 with scenarios", func(t *testing.T) {
		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/run"
		writeTestContent(t, tempDir, "../../examples/k6-test-scenarios.js")

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 2)
	})

	t.Run("Run k6 with checks and thresholds", func(t *testing.T) {
		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		writeTestContent(t, tempDir, "../../examples/k6-test-thresholds.js")

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, result.ErrorMessage, "some thresholds have failed")
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunDirs(t *testing.T) {
	// setup
	tempDir, _ := os.MkdirTemp("", "*")
	os.Setenv("RUNNER_DATADIR", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	os.Mkdir(repoDir, 0755)

	k6Script, err := ioutil.ReadFile("../../examples/k6-test-script.js")
	if err != nil {
		assert.FailNow(t, "Unable to read k6 test script")
	}

	err = ioutil.WriteFile(filepath.Join(repoDir, "k6-test-script.js"), k6Script, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write k6 runner test content file")
	}

	t.Run("Run k6 from directory with script argument", func(t *testing.T) {
		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/kubeshop/testkube-executor-k6.git",
				Branch: "main",
			},
		}
		execution.TestType = "k6/script"
		execution.Args = []string{"--duration", "1s", "k6-test-script.js"}

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})
}

func TestRunErrors(t *testing.T) {

	t.Run("Run k6 with no script", func(t *testing.T) {
		// setup
		os.Setenv("RUNNER_DATADIR", ".")

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "255")
	})

	t.Run("Run k6 with invalid test type script", func(t *testing.T) {
		// setup
		os.Setenv("RUNNER_DATADIR", ".")

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/unknown"

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "unsupported test type k6/unknown")
	})

	t.Run("Run k6 with invalid arguments", func(t *testing.T) {
		// setup
		os.Setenv("RUNNER_DATADIR", ".")

		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("")
		execution.TestType = "k6/script"
		execution.Args = []string{"--vues", "2", "--duration", "5"}

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "255")
	})

	t.Run("Run k6 from directory with missing script arg", func(t *testing.T) {
		// setup
		os.Setenv("RUNNER_DATADIR", ".")

		// given
		runner := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = &testkube.TestContent{
			Type_: string(testkube.TestContentTypeGitDir),
			Repository: &testkube.Repository{
				Uri:    "https://github.com/kubeshop/testkube-executor-k6.git",
				Branch: "main",
				Path:   "examples",
			},
		}
		execution.TestType = "k6/script"
		execution.Args = []string{}

		// when
		result, err := runner.Run(*execution)

		// then
		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, result.Status)
		assert.Contains(t, result.ErrorMessage, "not found")
	})
}

func TestExecutionResult(t *testing.T) {
	t.Run("Get default k6 execution result", func(t *testing.T) {
		// setup
		summary, err := ioutil.ReadFile("../../examples/k6-test-summary.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		result := finalExecutionResult(summary, nil)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Get custom scenario k6 execution result", func(t *testing.T) {
		// setup
		summary, err := ioutil.ReadFile("../../examples/k6-test-scenarios.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		result := finalExecutionResult(summary, nil)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 2)
	})
}

func TestParse(t *testing.T) {
	t.Run("Split scenario name", func(t *testing.T) {
		name := splitScenarioName("default: 1 iterations for each of 1 VUs (maxDuration: 10m0s, gracefulStop: 30s)")
		assert.Equal(t, "default", name)
	})

	t.Run("Parse k6 default summary", func(t *testing.T) {
		// setup
		summary, err := ioutil.ReadFile("../../examples/k6-test-summary.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		t.Run("Parse scenario names", func(t *testing.T) {
			names := parseScenarioNames(string(summary))
			assert.Len(t, names, 1)
			assert.Equal(t, "default: 1 iterations for each of 1 VUs (maxDuration: 10m0s, gracefulStop: 30s)", names[0])
		})

		t.Run("Parse default scenario duration", func(t *testing.T) {
			duration := parseScenarioDuration(string(summary), "default")
			assert.Equal(t, "00m01.0s/10m0s", duration)
		})
	})

	t.Run("Parse k6 scenario summary", func(t *testing.T) {
		// setup
		summary, err := ioutil.ReadFile("../../examples/k6-test-scenarios.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		t.Run("Parse scenario names", func(t *testing.T) {
			names := parseScenarioNames(string(summary))
			assert.Len(t, names, 2)
			assert.Equal(t, "testkube: 5 looping VUs for 10s (exec: testkube, gracefulStop: 30s)", names[0])
			assert.Equal(t, "monokle: 10 iterations for each of 5 VUs (maxDuration: 1m0s, exec: monokle, startTime: 5s, gracefulStop: 30s)", names[1])
		})

		t.Run("Parse teskube scenario duration", func(t *testing.T) {
			duration := parseScenarioDuration(string(summary), "testkube")
			assert.Equal(t, "10.0s/10s", duration)
		})

		t.Run("Parse monokle scenario duration", func(t *testing.T) {
			duration := parseScenarioDuration(string(summary), "monokle")
			assert.Equal(t, "0m10.2s/1m0s", duration)
		})
	})
}

func writeTestContent(t *testing.T, dir string, testScript string) {
	k6Script, err := ioutil.ReadFile(testScript)
	if err != nil {
		assert.FailNow(t, "Unable to read k6 test script")
	}

	err = ioutil.WriteFile(filepath.Join(dir, "test-content"), k6Script, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write k6 runner test content file")
	}
}
