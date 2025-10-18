package commands

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGitURI(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:     "https URL unchanged",
			input:    "https://github.com/kubeshop/testkube.git",
			expected: "https://github.com/kubeshop/testkube.git",
		},
		{
			name:     "http URL unchanged",
			input:    "http://github.com/kubeshop/testkube.git",
			expected: "http://github.com/kubeshop/testkube.git",
		},
		{
			name:     "SSH format converted",
			input:    "git@github.com:kubeshop/testkube.git",
			expected: "ssh://git@github.com/kubeshop/testkube.git",
		},
		{
			name:     "SSH URL unchanged",
			input:    "ssh://git@github.com:2222/kubeshop/testkube.git",
			expected: "ssh://git@github.com:2222/kubeshop/testkube.git",
		},
		{
			name:     "file path with backslash unchanged",
			input:    "C:\\Users\\test\\repo",
			expected: "c:\\Users\\test\\repo", // URL parsing lowercases the scheme
		},
		{
			name:     "file URL unchanged",
			input:    "file:///home/user/repo",
			expected: "file:///home/user/repo",
		},
		{
			name:     "git protocol URL unchanged",
			input:    "git://github.com/kubeshop/testkube.git",
			expected: "git://github.com/kubeshop/testkube.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeGitURI(tt.input)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

func TestSetupAuthentication(t *testing.T) {
	tests := []struct {
		name         string
		opts         *CloneOptions
		inputURL     string
		expectedUser string
		expectedArgs []string
		checkResult  func(t *testing.T, uri *url.URL, authArgs []string)
	}{
		{
			name: "basic auth with username and token",
			opts: &CloneOptions{
				AuthType: "basic",
				Username: "user",
				Token:    "token",
			},
			inputURL:     "https://github.com/kubeshop/testkube.git",
			expectedUser: "user:token",
			expectedArgs: []string{},
		},
		{
			name: "basic auth with username only",
			opts: &CloneOptions{
				AuthType: "basic",
				Username: "user",
			},
			inputURL:     "https://github.com/kubeshop/testkube.git",
			expectedUser: "user",
			expectedArgs: []string{},
		},
		{
			name: "basic auth with token only",
			opts: &CloneOptions{
				AuthType: "basic",
				Token:    "token",
			},
			inputURL:     "https://github.com/kubeshop/testkube.git",
			expectedUser: "token",
			expectedArgs: []string{},
		},
		{
			name: "header auth with token",
			opts: &CloneOptions{
				AuthType: "header",
				Username: "user",
				Token:    "token",
			},
			inputURL:     "https://github.com/kubeshop/testkube.git",
			expectedUser: "user",
			expectedArgs: []string{"-c", "http.extraHeader='Authorization: Bearer token'"},
		},
		{
			name: "header auth without token",
			opts: &CloneOptions{
				AuthType: "header",
				Username: "user",
			},
			inputURL:     "https://github.com/kubeshop/testkube.git",
			expectedUser: "user",
			expectedArgs: []string{},
		},
		{
			name: "empty auth",
			opts: &CloneOptions{
				AuthType: "basic",
			},
			inputURL:     "https://github.com/kubeshop/testkube.git",
			expectedUser: "",
			expectedArgs: []string{},
		},
		{
			name: "proper username and token usage",
			opts: &CloneOptions{
				AuthType: "basic",
				Username: "x-token-auth",
				Token:    "actualtoken",
			},
			inputURL:     "https://bitbucket.org/example/repo.git",
			expectedArgs: []string{},
			checkResult: func(t *testing.T, uri *url.URL, _ []string) {
				assert.Equal(t, "x-token-auth", uri.User.Username())
				pass, hasPass := uri.User.Password()
				assert.True(t, hasPass)
				assert.Equal(t, "actualtoken", pass)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := url.Parse(tt.inputURL)
			require.NoError(t, err)

			authArgs, err := setupAuthentication(context.Background(), uri, tt.opts)
			require.NoError(t, err)

			if tt.checkResult != nil {
				tt.checkResult(t, uri, authArgs)
			} else {
				// Default checks for backward compatibility
				if tt.expectedUser != "" {
					assert.NotNil(t, uri.User)
					if uri.User != nil {
						assert.Equal(t, tt.expectedUser, uri.User.String())
					}
				} else {
					assert.Nil(t, uri.User)
				}
			}

			assert.Equal(t, tt.expectedArgs, authArgs)
		})
	}
}

func TestSetupSSHKey(t *testing.T) {
	tests := []struct {
		name      string
		sshKey    string
		shouldSet bool
	}{
		{
			name:      "valid SSH key",
			sshKey:    "-----BEGIN RSA PRIVATE KEY-----\ntest key content\n-----END RSA PRIVATE KEY-----",
			shouldSet: true,
		},
		{
			name:      "empty SSH key",
			sshKey:    "",
			shouldSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp directory for this test
			tmpDir := t.TempDir()
			t.Setenv("TMPDIR", tmpDir)

			// Clear environment before test
			t.Setenv("GIT_SSH_COMMAND", "")

			cleanup, err := setupSSHKey(tt.sshKey)
			require.NoError(t, err)
			defer cleanup()

			if tt.shouldSet {
				assert.NotEmpty(t, os.Getenv("GIT_SSH_COMMAND"))
				assert.Contains(t, os.Getenv("GIT_SSH_COMMAND"), "StrictHostKeyChecking=no")
			} else {
				assert.Empty(t, os.Getenv("GIT_SSH_COMMAND"))
			}
		})
	}
}

func TestCleanPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		cone     bool
		expected []string
	}{
		{
			name:     "basic paths",
			input:    []string{"src", "tests", "docs"},
			cone:     false,
			expected: []string{"src", "tests", "docs"},
		},
		{
			name:     "paths with leading slash - cone mode",
			input:    []string{"/src", "/tests", "/docs"},
			cone:     true,
			expected: []string{"src", "tests", "docs"},
		},
		{
			name:     "paths with leading slash - no cone mode",
			input:    []string{"/src", "/tests", "/docs"},
			cone:     false,
			expected: []string{"/src", "/tests", "/docs"},
		},
		{
			name:     "mixed paths",
			input:    []string{"src", "./tests", "../docs", "", "."},
			cone:     false,
			expected: []string{"src", "tests", "../docs"},
		},
		{
			name:     "root path in cone mode",
			input:    []string{"/", "/src"},
			cone:     true,
			expected: []string{"/", "src"},
		},
		{
			name:     "paths needing cleaning",
			input:    []string{"src//subdir", "tests/./unit", "docs/../README"},
			cone:     false,
			expected: []string{"src/subdir", "tests/unit", "README"},
		},
		{
			name:     "empty paths",
			input:    []string{},
			cone:     false,
			expected: []string{},
		},
		{
			name:     "paths with dots and empty strings",
			input:    []string{"", ".", "..", "../..", "src/.."},
			cone:     false,
			expected: []string{"..", "../.."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanPaths(tt.input, tt.cone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCloneOptions(t *testing.T) {
	opts := &CloneOptions{
		RawPaths: []string{"src", "docs"},
		Username: "testuser",
		Token:    "testtoken",
		SSHKey:   "test-ssh-key",
		AuthType: "basic",
		Revision: "main",
		Cone:     true,
	}

	// Verify all fields are accessible
	assert.Equal(t, []string{"src", "docs"}, opts.RawPaths)
	assert.Equal(t, "testuser", opts.Username)
	assert.Equal(t, "testtoken", opts.Token)
	assert.Equal(t, "test-ssh-key", opts.SSHKey)
	assert.Equal(t, "basic", opts.AuthType)
	assert.Equal(t, "main", opts.Revision)
	assert.True(t, opts.Cone)
}

func TestIsCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid SHA-1 hash lowercase",
			input:    "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
			expected: true,
		},
		{
			name:     "valid SHA-1 hash uppercase",
			input:    "A94A8FE5CCB19BA61C4C0873D391E987982FBBD3",
			expected: true,
		},
		{
			name:     "valid SHA-1 hash mixed case",
			input:    "a94A8Fe5ccB19bA61c4C0873d391e987982FBbD3",
			expected: true,
		},
		{
			name:     "too short",
			input:    "a94a8fe5ccb19ba61c4c0873d391e987982fbb",
			expected: false,
		},
		{
			name:     "too long",
			input:    "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3a",
			expected: false,
		},
		{
			name:     "contains invalid character",
			input:    "a94a8fe5ccb19ba61c4c0873d391e987982fbbg3",
			expected: false,
		},
		{
			name:     "branch name",
			input:    "main",
			expected: false,
		},
		{
			name:     "tag name",
			input:    "v1.0.0",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommitHash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
