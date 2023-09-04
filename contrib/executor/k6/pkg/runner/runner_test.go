package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestExecutionResult(t *testing.T) {
	t.Parallel()

	t.Run("Get default k6 execution result", func(t *testing.T) {
		t.Parallel()
		// setup
		summary, err := os.ReadFile("../../examples/k6-test-summary.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		result := finalExecutionResult(string(summary), nil)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 1)
	})

	t.Run("Get custom scenario k6 execution result", func(t *testing.T) {
		t.Parallel()
		// setup
		summary, err := os.ReadFile("../../examples/k6-test-scenarios.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		result := finalExecutionResult(string(summary), nil)
		assert.Equal(t, testkube.ExecutionStatusPassed, result.Status)
		assert.Len(t, result.Steps, 2)
	})

	t.Run("Get successful checks for k6 execution result", func(t *testing.T) {
		t.Parallel()
		// setup
		summary, err := os.ReadFile("../../examples/k6-test-scenarios.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		result := areChecksSuccessful(string(summary))
		assert.Equal(t, true, result)
	})

}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("Split scenario name", func(t *testing.T) {
		t.Parallel()

		name := splitScenarioName("default: 1 iterations for each of 1 VUs (maxDuration: 10m0s, gracefulStop: 30s)")
		assert.Equal(t, "default", name)
	})

	t.Run("Parse k6 default summary", func(t *testing.T) {
		// setup
		summary, err := os.ReadFile("../../examples/k6-test-summary.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		t.Run("Parse scenario names", func(t *testing.T) {
			t.Parallel()

			names := parseScenarioNames(string(summary))
			assert.Len(t, names, 1)
			assert.Equal(t, "default: 1 iterations for each of 1 VUs (maxDuration: 10m0s, gracefulStop: 30s)", names[0])
		})

		t.Run("Parse default scenario duration", func(t *testing.T) {
			t.Parallel()

			duration := parseScenarioDuration(string(summary), "default")
			assert.Equal(t, "00m01.0s/10m0s", duration)
		})
	})

	t.Run("Parse k6 scenario summary", func(t *testing.T) {
		t.Parallel()
		// setup
		summary, err := os.ReadFile("../../examples/k6-test-scenarios.txt")
		if err != nil {
			assert.FailNow(t, "Unable to read k6 test summary")
		}

		t.Run("Parse scenario names", func(t *testing.T) {
			t.Parallel()

			names := parseScenarioNames(string(summary))
			assert.Len(t, names, 2)
			assert.Equal(t, "testkube: 5 looping VUs for 10s (exec: testkube, gracefulStop: 30s)", names[0])
			assert.Equal(t, "monokle: 10 iterations for each of 5 VUs (maxDuration: 1m0s, exec: monokle, startTime: 5s, gracefulStop: 30s)", names[1])
		})

		t.Run("Parse teskube scenario duration", func(t *testing.T) {
			t.Parallel()

			duration := parseScenarioDuration(string(summary), "testkube")
			assert.Equal(t, "10.0s/10s", duration)
		})

		t.Run("Parse monokle scenario duration", func(t *testing.T) {
			t.Parallel()

			duration := parseScenarioDuration(string(summary), "monokle")
			assert.Equal(t, "0m10.2s/1m0s", duration)
		})
	})
}
