package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/test/tktw/framework"
)

// TestToolkitBasicCommands tests the basic toolkit functionality
func TestToolkitBasicCommands(t *testing.T) {
	t.Run("help command", func(t *testing.T) {
		cleanup := framework.SetupTestEnvironment()
		t.Cleanup(cleanup)

		fw := framework.NewToolkitFramework().
			WithCommand("--help").
			WithTimeout(10 * time.Second)

		err := fw.Setup(t)
		require.NoError(t, err)
		t.Cleanup(func() { fw.Cleanup(t) })

		ctx := context.Background()
		err = fw.Run(ctx)
		require.NoError(t, err)

		fw.AssertSuccess(t)
		fw.AssertOutputContains(t, "Orchestrating Testkube workflows")
		fw.AssertOutputContains(t, "Available Commands")
	})

	t.Run("missing configuration", func(t *testing.T) {
		// Don't setup environment - should cause failure
		fw := framework.NewToolkitFramework().
			WithCommand("execute").
			WithArgs("--", "echo", "test").
			WithTimeout(5 * time.Second)

		err := fw.Setup(t)
		require.NoError(t, err)
		t.Cleanup(func() { fw.Cleanup(t) })

		ctx := context.Background()
		err = fw.Run(ctx)

		// Execute command requires configuration to work properly
		require.Error(t, err, "Toolkit process should fail with missing configuration")
		require.NotEqual(t, 0, fw.GetProcess().ExitCode, "Should exit with non-zero code")
	})
}

// TestToolkitWithCustomImages tests toolkit with different container images
func TestToolkitWithCustomImages(t *testing.T) {
	testCases := []struct {
		name  string
		image string
	}{
		{"curl image", "curlimages/curl:latest"},
		{"cypress image", "cypress/included:latest"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := framework.SetupTestEnvironment()
			t.Cleanup(cleanup)

			// Set container image
			require.NoError(t, os.Setenv("CONTAINER_IMAGE", tc.image))
			t.Cleanup(func() { os.Unsetenv("CONTAINER_IMAGE") })

			fw := framework.NewToolkitFramework().
				WithCommand("--help").
				WithTimeout(10 * time.Second)

			err := fw.Setup(t)
			require.NoError(t, err)
			t.Cleanup(func() { fw.Cleanup(t) })

			ctx := context.Background()
			err = fw.Run(ctx)
			require.NoError(t, err, "Toolkit process should work with %s", tc.image)

			fw.AssertSuccess(t)
		})
	}
}

// TestToolkitResourceMetrics tests resource usage tracking
func TestToolkitResourceMetrics(t *testing.T) {
	cleanup := framework.SetupResourceTestEnvironment()
	t.Cleanup(cleanup)

	fw := framework.NewToolkitFramework().
		WithCommand("--help").
		WithTimeout(10 * time.Second)

	err := fw.Setup(t)
	require.NoError(t, err)
	t.Cleanup(func() { fw.Cleanup(t) })

	ctx := context.Background()
	err = fw.Run(ctx)
	require.NoError(t, err)

	fw.AssertSuccess(t)

	// Verify process metrics
	process := fw.GetProcess()
	assert.Greater(t, process.Duration, 10*time.Millisecond, "Process should run for at least 10ms")
	assert.Less(t, process.Duration, 5*time.Second, "Process should complete within 5 seconds")
}
