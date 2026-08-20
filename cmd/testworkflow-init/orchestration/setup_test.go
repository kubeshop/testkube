package orchestration

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/executiondata"
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

func TestGetSensitiveWords(t *testing.T) {
	t.Run("keeps a short word added at runtime", func(t *testing.T) {
		// This set is the only thing deciding whether a step output may be published, so
		// dropping a word for being short would publish the credential it stands for
		// verbatim into the execution record - and mask it nowhere in the logs either.
		// Production sets the minimum to 4, which must not reach these words.
		setup := newSetup()
		setup.SetSensitiveWordMinimumLength(4)
		setup.AddSensitiveWords("a", "abc", "s3cr3t-value")

		words := setup.GetSensitiveWords()
		assert.Contains(t, words, "s3cr3t-value")
		assert.Contains(t, words, "abc", "a three-character credential is still a credential")
		assert.Contains(t, words, "a")
	})

	t.Run("a short output value is withheld from the record", func(t *testing.T) {
		// The end of the path the previous case guards: a short credential written to the
		// outputs directory must not leave the workflow.
		setup := newSetup()
		setup.SetSensitiveWordMinimumLength(4)
		setup.AddSensitiveWords("ab")

		publishable, withheld := data.PartitionSensitiveOutputs(
			map[string]string{"token": "ab", "count": "42"}, setup.GetSensitiveWords())
		assert.Equal(t, map[string]string{"count": "42"}, publishable)
		assert.Equal(t, []string{"token"}, withheld)
	})
}

// A computed environment variable is resolved and installed by UseEnv, which no later
// guard inspects - the command guard only sees arguments. So a withheld output reaching
// a variable has to stop the step here, rather than arrive at the tool as its value.
func TestUseEnv_WithheldOutput(t *testing.T) {
	// UseEnv replaces the process environment, so put it back for the tests after this one.
	original := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, item := range original {
			key, value, _ := strings.Cut(item, "=")
			assert.NoError(t, os.Setenv(key, value))
		}
	})

	marker := executiondata.WithheldMarker("producer", "token")

	t.Run("a computed variable resolving to a marker stops the step", func(t *testing.T) {
		require.NoError(t, os.Setenv("_01C_TOKEN", marker))
		require.NoError(t, os.Setenv("_01C_GREETING", "hello"))

		setup := newSetup()
		setup.initialize()

		err := setup.UseEnv("01")
		require.Error(t, err)
		assert.ErrorContains(t, err, `the "TOKEN" environment variable`)
		assert.ErrorContains(t, err, "was not published outside the workflow that produced it")
		assert.ErrorContains(t, err, marker)
		assert.Empty(t, os.Getenv("TOKEN"), "the marker must not reach the environment")
	})

	t.Run("a plain variable holding a marker stops the step too", func(t *testing.T) {
		// A step that spawns workers resolves their specification itself, so the marker
		// reaches the worker as a literal rather than as something to compute.
		os.Clearenv()
		require.NoError(t, os.Setenv("_02_TOKEN", marker))

		setup := newSetup()
		setup.initialize()

		err := setup.UseEnv("02")
		require.Error(t, err)
		assert.ErrorContains(t, err, `the "TOKEN" environment variable`)
		assert.Empty(t, os.Getenv("TOKEN"), "the marker must not reach the environment")
	})
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
