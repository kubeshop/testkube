package init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runner"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestStepOutputsAndResults_Integration(t *testing.T) {
	test.IntegrationTest(t)

	testDir := t.TempDir()
	internalPath := filepath.Join(testDir, ".tktw")
	require.NoError(t, os.MkdirAll(internalPath, 0755))

	termLogPath := filepath.Join(testDir, "termination.log")
	require.NoError(t, os.WriteFile(termLogPath, []byte{}, 0666))

	outputsDir := filepath.Join(testDir, "testkube", "outputs")

	// Step 1 writes outputs to the outputs directory
	step1Script := filepath.Join(testDir, "step1.sh")
	require.NoError(t, os.WriteFile(step1Script, []byte("#!/bin/sh\nset -e\nmkdir -p "+outputsDir+"\necho secret_token_123 > "+outputsDir+"/token\necho 42 > "+outputsDir+"/count\n"), 0755))

	// Step 2 just runs successfully
	step2Script := filepath.Join(testDir, "step2.sh")
	require.NoError(t, os.WriteFile(step2Script, []byte("#!/bin/sh\necho ok\n"), 0755))

	// Action format: S=Start(string), E=End(string), d=Declare, e=Execute, c=Container
	actions := [][]map[string]any{
		// Group 0 - Setup
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		// Group 1 - Step that writes outputs
		{
			{"d": map[string]any{"c": "true", "r": "step1", "i": "generate"}},
			{"S": "step1"},
			{"c": map[string]any{
				"r": "step1",
				"c": map[string]any{
					"command": []string{step1Script},
				},
			}},
			{"e": map[string]any{"r": "step1", "t": true}},
			{"E": "step1"},
		},
		// Group 2 - Step that should see step 1's outputs in state
		{
			{"d": map[string]any{"c": "true", "r": "step2", "i": "use_data", "p": []string{"step1"}}},
			{"S": "step2"},
			{"c": map[string]any{
				"r": "step2",
				"c": map[string]any{
					"command": []string{step2Script},
				},
			}},
			{"e": map[string]any{"r": "step2", "t": true}},
			{"E": "step2"},
		},
	}

	setupEnvWithActions(t, testDir, actions)
	updateConstants(testDir)

	// Override dirs for testing (production paths are read-only in test env)
	data.SetOutputsDir(outputsDir)
	data.SetStepResultsBase(filepath.Join(testDir, "data", ".steps"))

	statePath := filepath.Join(testDir, ".tktw", "state")

	// Run Group 0 - Setup
	t.Run("Setup", func(t *testing.T) {
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode, err := runner.RunInit(0)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode)
	})

	// Run Group 1 - Step that writes outputs
	t.Run("GenerateOutputs", func(t *testing.T) {
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode, err := runner.RunInit(1)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode)

		// Verify outputs persisted to state file
		state := loadStateFromPath(t, statePath)
		assert.Equal(t, "secret_token_123", state.Output["step.generate.token"], "token output should be in state")
		assert.Equal(t, "42", state.Output["step.generate.count"], "count output should be in state")

		// Verify step ID was stored
		if step1, ok := state.Steps["step1"].(map[string]any); ok {
			assert.Equal(t, "generate", step1["I"], "step ID should be stored")
		}
	})

	// Run Group 2 - Outputs from step 1 should survive state reload
	t.Run("OutputsSurviveStateReload", func(t *testing.T) {
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode, err := runner.RunInit(2)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode)

		// Verify step 1 outputs still in state after group 2
		state := loadStateFromPath(t, statePath)
		assert.Equal(t, "secret_token_123", state.Output["step.generate.token"], "step 1 outputs should persist across groups")
		assert.Equal(t, "42", state.Output["step.generate.count"])
	})
}

func TestStepResultExpressions_Integration(t *testing.T) {
	test.IntegrationTest(t)

	testDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(testDir, ".tktw"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "termination.log"), []byte{}, 0666))

	failScript := filepath.Join(testDir, "fail.sh")
	require.NoError(t, os.WriteFile(failScript, []byte("#!/bin/sh\nexit 3\n"), 0755))

	// The second step gets the result of the first step as arguments, so the file holds what the expressions resolved to.
	resultPath := filepath.Join(testDir, "result.txt")
	readScript := filepath.Join(testDir, "read.sh")
	require.NoError(t, os.WriteFile(readScript, []byte("#!/bin/sh\necho \"$1 $2\" > "+resultPath+"\n"), 0755))

	actions := [][]map[string]any{
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		{
			{"d": map[string]any{"c": "true", "r": "step1", "i": "failing"}},
			{"S": "step1"},
			{"c": map[string]any{"r": "step1", "c": map[string]any{"command": []string{failScript}}}},
			{"e": map[string]any{"r": "step1"}},
			{"E": "step1"},
		},
		{
			{"d": map[string]any{"c": "true", "r": "step2", "i": "read_result", "p": []string{"step1"}}},
			{"S": "step2"},
			{"c": map[string]any{"r": "step2", "c": map[string]any{
				"command": []string{readScript, "{{step.failing.exitCode}}", "{{step.failing.status}}"},
			}}},
			{"e": map[string]any{"r": "step2"}},
			{"E": "step2"},
		},
	}

	setupEnvWithActions(t, testDir, actions)
	updateConstants(testDir)
	// The test overrides the directories, because the production paths are read-only.
	data.SetOutputsDir(filepath.Join(testDir, "testkube", "outputs"))
	data.SetStepResultsBase(filepath.Join(testDir, "data", ".steps"))

	for i := range actions {
		initializeOrchestration(t)
		_, err := runner.RunInit(i)
		cleanupOrchestration(t)
		require.NoError(t, err)
	}

	result, err := os.ReadFile(resultPath)
	require.NoError(t, err, "the second step did not run")
	assert.Equal(t, "3 failed", strings.TrimSpace(string(result)))
}

func TestStepConditionReadsTimedOutStatus_Integration(t *testing.T) {
	test.IntegrationTest(t)

	testDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(testDir, ".tktw"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "termination.log"), []byte{}, 0666))

	sleepScript := filepath.Join(testDir, "sleep.sh")
	require.NoError(t, os.WriteFile(sleepScript, []byte("#!/bin/sh\nsleep 30\n"), 0755))

	ranPath := filepath.Join(testDir, "ran.txt")
	markScript := filepath.Join(testDir, "mark.sh")
	require.NoError(t, os.WriteFile(markScript, []byte("#!/bin/sh\necho ran > "+ranPath+"\n"), 0755))

	actions := [][]map[string]any{
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		{
			{"d": map[string]any{"c": "true", "r": "step1", "i": "slow", "o": true}},
			{"S": "step1"},
			{"t": map[string]any{"r": "step1", "t": "1s"}},
			{"c": map[string]any{"r": "step1", "c": map[string]any{"command": []string{sleepScript}}}},
			{"e": map[string]any{"r": "step1"}},
			{"E": "step1"},
		},
		{
			{"d": map[string]any{"c": `step.slow.status == "timeout"`, "r": "step2", "i": "after_timeout", "p": []string{"step1"}}},
			{"S": "step2"},
			{"c": map[string]any{"r": "step2", "c": map[string]any{"command": []string{markScript}}}},
			{"e": map[string]any{"r": "step2"}},
			{"E": "step2"},
		},
	}

	setupEnvWithActions(t, testDir, actions)
	updateConstants(testDir)
	data.SetOutputsDir(filepath.Join(testDir, "testkube", "outputs"))
	data.SetStepResultsBase(filepath.Join(testDir, "data", ".steps"))

	for i := range actions {
		initializeOrchestration(t)
		_, err := runner.RunInit(i)
		cleanupOrchestration(t)
		require.NoError(t, err, "the condition must resolve against the step that timed out")
	}

	_, err := os.Stat(ranPath)
	assert.NoError(t, err, "the step whose condition read the timed out status did not run")
}
