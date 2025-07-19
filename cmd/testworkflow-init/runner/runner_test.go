package runner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/internal/test/framework"
)

const (
	// testErrorExitCode is the exit code used for testing error scenarios
	testErrorExitCode = 42
)

func TestInitProcess(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		cleanup := framework.SetupTestEnvironment()
		t.Cleanup(cleanup)

		fw := framework.NewInitTestFramework()
		err := fw.Setup(t)
		require.NoError(t, err)
		t.Cleanup(func() { fw.Cleanup(t) })

		ctx := context.Background()
		err = fw.Run(ctx)
		require.NoError(t, err)

		// Verify process completed successfully
		process := fw.GetProcess()
		assert.Equal(t, 0, process.ExitCode)
		assert.Greater(t, process.Duration, time.Duration(0), "process should have measurable duration")
		// Init process should be fast, but avoid flaky timing assertions

		// Verify directory structure
		tempDir := fw.GetTempDir().Path()
		assert.DirExists(t, filepath.Join(tempDir, ".tktw", "bin"))
		assert.DirExists(t, filepath.Join(tempDir, "tmp"))

		// Verify metrics directory structure
		assert.DirExists(t, filepath.Join(tempDir, "data", ".testkube", "internal", "metrics"))
		// Verify step-specific metrics directory
		metricsStepDir := filepath.Join(tempDir, "data", ".testkube", "internal", "metrics", framework.DefaultStepRef)
		assert.DirExists(t, metricsStepDir, "metrics directory for step should exist")

		// Verify shell binary was copied
		shPath := filepath.Join(tempDir, ".tktw", "bin", "sh")
		assert.FileExists(t, shPath)

		// Verify state file exists with correct permissions
		statePath := filepath.Join(tempDir, ".tktw", "state")
		assert.FileExists(t, statePath)

		info, err := os.Stat(statePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0777), info.Mode().Perm(), "state file should have 0777 permissions")
	})

	t.Run("handles invalid group index", func(t *testing.T) {
		cleanup := framework.SetupTestEnvironment()
		t.Cleanup(cleanup)

		fw := framework.NewInitTestFramework()
		err := fw.Setup(t)
		require.NoError(t, err)
		t.Cleanup(func() { fw.Cleanup(t) })

		ctx := context.Background()
		err = fw.RunGroup(ctx, 99)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown actions group")

		process := fw.GetProcess()
		assert.Equal(t, 1, process.ExitCode)
	})

	t.Run("custom resource limits", func(t *testing.T) {
		opts := framework.WorkflowOptions{
			StepRef: "resource-test",
		}
		opts.Resources.Requests.CPU = "10m"
		opts.Resources.Requests.Memory = "32Mi"
		opts.Resources.Limits.CPU = "50m"
		opts.Resources.Limits.Memory = "64Mi"

		cleanup := framework.SetupTestEnvironmentWithOptions(opts)
		t.Cleanup(cleanup)

		fw := framework.NewInitTestFramework()
		err := fw.Setup(t)
		require.NoError(t, err)
		t.Cleanup(func() { fw.Cleanup(t) })

		ctx := context.Background()
		err = fw.Run(ctx)
		require.NoError(t, err)

		// Read and verify the state file contains our custom resources
		statePath := filepath.Join(fw.GetTempDir().Path(), ".tktw", "state")
		stateData, err := os.ReadFile(statePath)
		require.NoError(t, err)

		var state map[string]interface{}
		err = json.Unmarshal(stateData, &state)
		require.NoError(t, err)

		resources := state["R"].(map[string]interface{})
		requests := resources["r"].(map[string]interface{})
		limits := resources["l"].(map[string]interface{})

		assert.Equal(t, "10m", requests["c"])
		assert.Equal(t, "32Mi", requests["m"])
		assert.Equal(t, "50m", limits["c"])
		assert.Equal(t, "64Mi", limits["m"])
	})

	t.Run("error handling workflow", func(t *testing.T) {
		// Create a workflow that fails by using a command that exits with error
		builder := framework.NewActionBuilder()
		builder.AddInitGroup("fail-test")
		builder.AddExecutionGroup("fail-test", []string{"/bin/sh"}, []string{"-c", "echo 'This test will fail' && exit " + fmt.Sprintf("%d", testErrorExitCode)})

		actions, err := builder.Build()
		require.NoError(t, err)

		cleanup := framework.SetupTestEnvironmentWithOptions(framework.WorkflowOptions{
			Actions:   actions,
			StepRef:   "fail-test",
			Signature: `[{"ref": "fail-test", "name": "fail-test", "category": "Error handling test"}]`,
		})
		t.Cleanup(cleanup)

		fw := framework.NewInitTestFramework()
		err = fw.Setup(t)
		require.NoError(t, err)
		t.Cleanup(func() { fw.Cleanup(t) })

		ctx := context.Background()
		err = fw.Run(ctx)
		require.NoError(t, err)

		// Run the group - init framework runs successfully
		err = fw.RunGroup(ctx, 1)
		require.NoError(t, err)

		// The command inside should have exited with our error code
		// Check the state file to verify the step failed
		statePath := filepath.Join(fw.GetTempDir().Path(), ".tktw", "state")
		stateData, err := os.ReadFile(statePath)
		require.NoError(t, err)

		var state map[string]interface{}
		err = json.Unmarshal(stateData, &state)
		require.NoError(t, err)

		// Check the step status
		steps := state["S"].(map[string]interface{})
		failStep := steps["fail-test"].(map[string]interface{})
		assert.Equal(t, "failed", failStep["s"], "step should have failed status")
	})
}

func TestInitStateContent(t *testing.T) {
	cleanup := framework.SetupTestEnvironment()
	t.Cleanup(cleanup)

	fw := framework.NewInitTestFramework()
	err := fw.Setup(t)
	require.NoError(t, err)
	t.Cleanup(func() { fw.Cleanup(t) })

	ctx := context.Background()
	err = fw.Run(ctx)
	require.NoError(t, err)

	// Read and parse state file
	statePath := filepath.Join(fw.GetTempDir().Path(), ".tktw", "state")
	stateData, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state map[string]interface{}
	err = json.Unmarshal(stateData, &state)
	require.NoError(t, err, "state file should contain valid JSON")

	t.Run("has required top-level fields", func(t *testing.T) {
		assert.Contains(t, state, "a", "state should have actions field 'a'")
		assert.Contains(t, state, "C", "state should have config field 'C'")
		assert.Contains(t, state, "S", "state should have steps field 'S'")
		assert.Contains(t, state, "R", "state should have resources field 'R'")
	})

	t.Run("actions structure is valid", func(t *testing.T) {
		actions, ok := state["a"].([]interface{})
		require.True(t, ok, "actions 'a' should be an array")
		require.GreaterOrEqual(t, len(actions), 1, "should have at least one action group")

		// Verify group 0 contains setup action
		group0, ok := actions[0].([]interface{})
		require.True(t, ok, "group 0 should be an array")

		var foundSetup bool
		for _, action := range group0 {
			if actionMap, ok := action.(map[string]interface{}); ok {
				if _, hasSetup := actionMap["_"]; hasSetup {
					foundSetup = true
					break
				}
			}
		}
		assert.True(t, foundSetup, "group 0 should contain a setup action")
	})

	t.Run("current reference is set", func(t *testing.T) {
		currentRef, ok := state["c"].(string)
		assert.True(t, ok, "current reference 'c' should be a string")
		assert.Equal(t, "root", currentRef, "initial reference should be 'root'")
	})

	t.Run("steps tracking includes init", func(t *testing.T) {
		steps, ok := state["S"].(map[string]interface{})
		require.True(t, ok, "steps 'S' should be a map")

		initStep, ok := steps["tktw-init"].(map[string]interface{})
		require.True(t, ok, "should have 'tktw-init' step")

		assert.Equal(t, "tktw-init", initStep["_"], "step ref should be 'tktw-init'")
		assert.Equal(t, "passed", initStep["s"], "step status should be 'passed'")
		assert.Equal(t, "passed", initStep["c"], "step condition should be 'passed'")
	})

	t.Run("resources are configured", func(t *testing.T) {
		resources, ok := state["R"].(map[string]interface{})
		require.True(t, ok, "resources 'R' should be a map")

		// Check requests
		if requests, hasRequests := resources["r"].(map[string]interface{}); hasRequests {
			assert.Contains(t, requests, "c", "should have CPU request")
			assert.Contains(t, requests, "m", "should have memory request")
		}

		// Check limits
		if limits, hasLimits := resources["l"].(map[string]interface{}); hasLimits {
			assert.Contains(t, limits, "c", "should have CPU limit")
			assert.Contains(t, limits, "m", "should have memory limit")
		}
	})

	t.Run("outputs field behavior", func(t *testing.T) {
		// The outputs field may or may not exist initially
		if outputs, exists := state["o"]; exists {
			outputMap, ok := outputs.(map[string]interface{})
			assert.True(t, ok, "outputs 'o' should be a map when present")
			assert.Empty(t, outputMap, "initial outputs should be empty")
		}
	})
}

func TestInitMetrics(t *testing.T) {
	cleanup := framework.SetupTestEnvironment()
	t.Cleanup(cleanup)

	fw := framework.NewInitTestFramework()
	err := fw.Setup(t)
	require.NoError(t, err)
	t.Cleanup(func() { fw.Cleanup(t) })

	// Run the init process (group 0) first
	ctx := context.Background()
	err = fw.Run(ctx)
	require.NoError(t, err)

	// Now run group 1 which contains the actual test that runs for ~5 seconds
	err = fw.RunGroup(ctx, 1)
	require.NoError(t, err)

	t.Run("metrics directory structure exists", func(t *testing.T) {
		// Verify metrics directory structure
		metricsPath := filepath.Join(fw.GetTempDir().Path(), "data", ".testkube", "internal", "metrics")
		assert.DirExists(t, metricsPath, "metrics root directory should exist")

		// Check for any metrics directories (could be for different steps)
		entries, err := os.ReadDir(metricsPath)
		require.NoError(t, err)
		assert.Greater(t, len(entries), 0, "should have at least one step metrics directory")
	})

	t.Run("metrics files are written by background collector", func(t *testing.T) {
		// The metrics recorder runs in a goroutine and collects metrics every second
		// Check the actual location where metrics are written: /.tktw/metrics/{stepRef}
		metricsPath := filepath.Join(fw.GetTempDir().Path(), ".tktw", "metrics", framework.DefaultStepRef)

		// The directory should exist
		assert.DirExists(t, metricsPath, "step-specific metrics directory should exist")

		// List all metrics files
		entries, err := os.ReadDir(metricsPath)
		require.NoError(t, err, "should be able to read metrics directory")

		// Log what we found
		t.Logf("Found %d entries in metrics directory %s", len(entries), metricsPath)

		metricsFileCount := 0
		totalMetricsSize := int64(0)

		for _, entry := range entries {
			if !entry.IsDir() {
				metricsFileCount++
				info, err := entry.Info()
				if err == nil {
					totalMetricsSize += info.Size()
				}

				// Read and validate the metrics file
				filePath := filepath.Join(metricsPath, entry.Name())
				data, err := os.ReadFile(filePath)
				if err == nil {
					t.Logf("  - %s: %d bytes", entry.Name(), len(data))

					// Basic validation of metrics content
					content := string(data)
					assert.NotEmpty(t, content, "metrics file should not be empty")

					// Print the full metrics file content for verification
					t.Logf("Full metrics file content:\n%s", content)

					// Validate that it contains expected metrics
					// The file should be in InfluxDB line protocol format with CPU, memory, disk metrics
					assert.Contains(t, content, "workflow=", "should contain workflow metadata")
					assert.Contains(t, content, "step.ref=", "should contain step reference")

					// Check for actual metrics data (CPU, memory, disk, etc.)
					assertContainsMetricsData(t, content)
				}
			}
		}

		// The process ran for expected duration, collecting metrics every second
		// We should have at least a few metrics files
		assert.Greater(t, metricsFileCount, 0, "should have at least one metrics file")
		assert.Greater(t, totalMetricsSize, int64(0), "metrics files should contain data")
	})
}

// assertContainsMetricsData verifies that the metrics content contains actual metric data
func assertContainsMetricsData(t *testing.T, content string) {
	metricTypes := []string{"cpu", "memory", "disk", "network"}
	hasMetricsData := false

	for _, line := range strings.Split(content, "\n") {
		for _, metricType := range metricTypes {
			if strings.Contains(line, metricType) {
				hasMetricsData = true
				break
			}
		}
		if hasMetricsData {
			break
		}
	}

	assert.True(t, hasMetricsData, "should contain actual metrics data (cpu/memory/disk/network)")
}
