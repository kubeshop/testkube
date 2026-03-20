package init_test

import (
	"os"
	"path/filepath"
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
