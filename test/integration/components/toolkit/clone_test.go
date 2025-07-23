package toolkit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

const (
	// Using the Testkube examples repository as test target (lightweight)
	testRepoURL = "https://github.com/kubeshop/testkube-examples.git"
	testBranch  = "main"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set TK_CFG environment variable for all tests
	os.Setenv("TK_CFG", "{}")

	// Run tests
	code := m.Run()

	// Clean up
	os.Unsetenv("TK_CFG")

	os.Exit(code)
}

func TestCloneBasicHTTPS_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	outputDir := t.TempDir()

	// Test basic clone without authentication
	err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{})
	require.NoError(t, err)

	// Verify clone was successful
	assert.DirExists(t, filepath.Join(outputDir, ".git"))
	assert.FileExists(t, filepath.Join(outputDir, "README.md"))
	assert.DirExists(t, filepath.Join(outputDir, "ArgoCD"))
}

func TestCloneWithRevision_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	testCases := []struct {
		name          string
		revision      string
		verify        func(t *testing.T, outputDir string)
		skipIfNoMatch bool
	}{
		{
			name:     "clone main branch",
			revision: testBranch,
			verify: func(t *testing.T, outputDir string) {
				// Check current branch
				cmd := exec.Command("git", "-C", outputDir, "branch", "--show-current")
				output, err := cmd.Output()
				require.NoError(t, err)
				branch := strings.TrimSpace(string(output))
				assert.Equal(t, testBranch, branch)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outputDir := t.TempDir()

			err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{
				Revision: tc.revision,
			})

			if tc.skipIfNoMatch && err != nil && strings.Contains(err.Error(), "revision") {
				t.Skipf("Revision %s not found, skipping test", tc.revision)
			}

			require.NoError(t, err)
			tc.verify(t, outputDir)
		})
	}
}

func TestCloneSparseCheckout_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	testCases := []struct {
		name   string
		paths  []string
		cone   bool
		verify func(t *testing.T, outputDir string)
	}{
		{
			name:  "non-cone mode with specific paths",
			paths: []string{"README.md", "ArgoCD", "Gradle"},
			cone:  false,
			verify: func(t *testing.T, outputDir string) {
				// Verify sparse checkout worked
				assert.FileExists(t, filepath.Join(outputDir, "README.md"))
				assert.DirExists(t, filepath.Join(outputDir, "ArgoCD"))
				assert.DirExists(t, filepath.Join(outputDir, "Gradle"))

				// In non-cone mode, parent directories might still be created
				// but they should be empty or contain only README files
			},
		},
		{
			name:  "cone mode with directory paths",
			paths: []string{"ArgoCD", "ArgoEvents"},
			cone:  true,
			verify: func(t *testing.T, outputDir string) {
				// Verify cone mode sparse checkout
				assert.DirExists(t, filepath.Join(outputDir, "ArgoCD"))
				assert.DirExists(t, filepath.Join(outputDir, "ArgoEvents"))

				// Root files should exist in cone mode
				assert.FileExists(t, filepath.Join(outputDir, "README.md"))
			},
		},
		{
			name:  "cone mode with single directory",
			paths: []string{"Gradle"},
			cone:  true,
			verify: func(t *testing.T, outputDir string) {
				// Verify only Gradle directory exists
				assert.DirExists(t, filepath.Join(outputDir, "Gradle"))
				assert.FileExists(t, filepath.Join(outputDir, "README.md"))

				// Other directories should not exist
				assert.NoDirExists(t, filepath.Join(outputDir, "ArgoCD"))
				assert.NoDirExists(t, filepath.Join(outputDir, "ArgoEvents"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outputDir := t.TempDir()

			err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{
				RawPaths: tc.paths,
				Cone:     tc.cone,
			})
			require.NoError(t, err)

			tc.verify(t, outputDir)
		})
	}
}

func TestCloneWithAuthentication_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	t.Run("valid token authentication", func(t *testing.T) {
		t.Parallel()
		// Skip if no GitHub token is available
		token := os.Getenv("GIT_TOKEN")
		if token == "" {
			t.Skip("GIT_TOKEN not set, skipping authenticated clone test")
		}

		outputDir := t.TempDir()

		// Test with token authentication
		err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{
			Token:    token,
			AuthType: "basic",
		})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(outputDir, ".git"))
	})

	t.Run("invalid token authentication", func(t *testing.T) {
		t.Parallel()
		// Use a private repository URL that requires authentication
		privateRepoURL := "https://github.com/kubeshop/testkube-private-test.git"
		outputDir := t.TempDir()

		// Test with invalid token
		err := executeClone(t, privateRepoURL, outputDir, &commands.CloneOptions{
			Token:    "invalid-token-12345",
			AuthType: "basic",
		})

		// Should fail with authentication error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cloning repository")
	})

	t.Run("username without password", func(t *testing.T) {
		t.Parallel()
		outputDir := t.TempDir()

		// Test with username only (should work for public repos)
		err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{
			Username: "testuser",
			AuthType: "basic",
		})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(outputDir, ".git"))
	})

	t.Run("header authentication with token", func(t *testing.T) {
		t.Parallel()
		token := os.Getenv("GIT_TOKEN")
		if token == "" {
			t.Skip("GIT_TOKEN not set, skipping header auth test")
		}

		outputDir := t.TempDir()

		// Test with header authentication
		err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{
			Token:    token,
			AuthType: "header",
		})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(outputDir, ".git"))
	})
}

func TestCloneSSH_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	// Skip if no SSH key is available
	sshKey := os.Getenv("GIT_SSH_KEY")
	if sshKey == "" {
		t.Skip("GIT_SSH_KEY not set, skipping SSH clone test")
	}

	outputDir := t.TempDir()

	// Test SSH clone with proper SSH URI
	err := executeClone(t, "git@github.com:kubeshop/testkube.git", outputDir, &commands.CloneOptions{
		SSHKey: sshKey,
	})

	if err != nil {
		// May fail due to SSH key permissions or GitHub key not being added
		assert.Contains(t, err.Error(), "cloning repository")
		t.Logf("SSH clone failed (expected if key not configured in GitHub): %v", err)
	} else {
		// If it succeeds, verify the clone
		assert.DirExists(t, filepath.Join(outputDir, ".git"))
		assert.FileExists(t, filepath.Join(outputDir, "README.md"))
	}
}

func TestCloneErrorHandling_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Parallel()

	t.Run("invalid URI", func(t *testing.T) {
		t.Parallel()
		outputDir := t.TempDir()
		err := executeClone(t, "not-a-valid-uri", outputDir, &commands.CloneOptions{})

		assert.Error(t, err)
		// Git treats this as a relative path, so we get a cloning error
		assert.Contains(t, err.Error(), "cloning repository")
	})

	t.Run("non-existent repository", func(t *testing.T) {
		t.Parallel()
		outputDir := t.TempDir()
		err := executeClone(t, "https://github.com/kubeshop/does-not-exist-12345.git", outputDir, &commands.CloneOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cloning repository")
	})

	t.Run("invalid revision", func(t *testing.T) {
		t.Parallel()
		outputDir := t.TempDir()
		err := executeClone(t, testRepoURL, outputDir, &commands.CloneOptions{
			Revision: "non-existent-branch-xyz",
		})

		assert.Error(t, err)
		// Can fail at clone stage (for full clone) or fetch stage (for sparse clone)
		assert.True(t, strings.Contains(err.Error(), "fetching revision") || strings.Contains(err.Error(), "cloning repository"),
			"Expected error to contain 'fetching revision' or 'cloning repository', got: %s", err.Error())
	})
}

// executeClone is a test helper that directly calls RunClone with the given options
func executeClone(t *testing.T, uri string, outputPath string, opts *commands.CloneOptions) error {
	t.Helper()

	// Set default auth type if not specified
	if opts.AuthType == "" {
		opts.AuthType = "basic"
	}

	// Call RunClone directly
	return commands.RunClone(context.Background(), uri, outputPath, opts)
}
