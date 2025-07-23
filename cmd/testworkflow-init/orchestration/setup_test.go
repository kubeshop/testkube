package orchestration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	// Read the test data from files
	rawConfig, err := os.ReadFile("testdata/config.json")
	require.NoError(t, err)
	rawSignature, err := os.ReadFile("testdata/signature.json")
	require.NoError(t, err)
	rawSpec, err := os.ReadFile("testdata/spec.json")
	require.NoError(t, err)

	// Set the environment variables and defer their cleanup
	assert.NoError(t, os.Setenv("_03_TKI_C", string(rawConfig)))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_03_TKI_C"))
	})
	assert.NoError(t, os.Setenv("_03_TKI_G", string(rawSignature)))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_03_TKI_G"))
	})
	assert.NoError(t, os.Setenv("_01_TKI_I", string(rawSpec)))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_01_TKI_I"))
	})
	assert.NoError(t, os.Setenv("_04_TKI_R_R_C", "128m"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_04_TKI_R_R_C"))
	})
	assert.NoError(t, os.Setenv("_04_TKI_R_R_M", "1024"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_04_TKI_R_R_M"))
	})
	assert.NoError(t, os.Setenv("_04_TKI_R_L_C", "256m"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_04_TKI_R_L_C"))
	})
	assert.NoError(t, os.Setenv("_04_TKI_R_L_M", "2048"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("_04_TKI_R_L_M"))
	})

	// Create a new setup instance
	setup := newSetup()
	setup.initialize()
	setup.UseBaseEnv()

	// Validate the data gets loaded correctly
	config := setup.GetInternalConfig()
	assert.Equal(t, "k6-sample", config.Workflow.Name)
	signature := setup.GetSignature()
	assert.Len(t, signature, 2)
	resources := setup.GetContainerResources()
	assert.Equal(t, "1024", resources.Requests.Memory)
	assert.Equal(t, "128m", resources.Requests.CPU)
	assert.Equal(t, "2048", resources.Limits.Memory)
	assert.Equal(t, "256m", resources.Limits.CPU)
}

func TestSetupInitialize_TableDriven(t *testing.T) {
	tests := []struct {
		name              string
		envVars           map[string]string
		expectedEnvGroups map[string]map[string]string
	}{
		{
			name:    "validate expected env groups",
			envVars: map[string]string{"_00_TKI_N": "node1", "_04_TKI_R_R_C": "123m"},
			expectedEnvGroups: map[string]map[string]string{
				"00": {"TKI_N": "node1"},
				"04": {"TKI_R_R_C": "123m"},
			},
		},
	}

	// Now we do sub-tests to verify each entry
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for envName, envValue := range tc.envVars {
				assert.NoError(t, os.Setenv(envName, envValue))
			}
			t.Cleanup(func() {
				for envName := range tc.envVars {
					assert.NoError(t, os.Unsetenv(envName))
				}
			})

			setup := newSetup()
			setup.initialize()

			assert.Equal(t, tc.expectedEnvGroups, setup.envGroups)
		})
	}
}
