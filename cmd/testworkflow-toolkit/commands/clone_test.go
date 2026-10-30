package commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"

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
			expectedArgs: []string{"-c", "http.extraHeader=Authorization: Bearer token"},
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

func TestAdjustFilePermissions(t *testing.T) {
	t.Run("successful permission adjustment", func(t *testing.T) {
		// Create temporary directory structure
		tmpDir := t.TempDir()

		// Create test files with restrictive permissions
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0400) // read-only
		require.NoError(t, err)

		// Create subdirectory with file
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0700)
		require.NoError(t, err)

		subFile := filepath.Join(subDir, "subtest.txt")
		err = os.WriteFile(subFile, []byte("test"), 0400)
		require.NoError(t, err)

		// Adjust permissions
		err = adjustFilePermissions(tmpDir, &CloneOptions{})
		require.NoError(t, err)

		// Check permissions were adjusted
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.True(t, info.Mode().Perm()&0o060 == 0o060, "File should have group read/write permissions")

		info, err = os.Stat(subFile)
		require.NoError(t, err)
		assert.True(t, info.Mode().Perm()&0o060 == 0o060, "Subdir file should have group read/write permissions")
	})

	t.Run("non-existent directory", func(t *testing.T) {
		err := adjustFilePermissions("/non/existent/path", &CloneOptions{})
		assert.Error(t, err)
	})

	t.Run("handles chmod errors gracefully", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission denied as root")
		}

		// Create a directory structure where we can't change permissions
		tmpDir := t.TempDir()

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		// Make the parent directory read-only
		err = os.Chmod(tmpDir, 0555)
		require.NoError(t, err)
		defer os.Chmod(tmpDir, 0755) // Restore for cleanup

		// adjustFilePermissions should not fail, just log warnings
		err = adjustFilePermissions(tmpDir, &CloneOptions{})
		// The function should complete without error even if chmod fails
		assert.NoError(t, err)
	})
}

func TestCopyRepositoryContents(t *testing.T) {
	// Create source directory with content
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create test file in source
	testFile := filepath.Join(srcDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create subdirectory with file
	subDir := filepath.Join(srcDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	subFile := filepath.Join(subDir, "subtest.txt")
	err = os.WriteFile(subFile, []byte("sub content"), 0644)
	require.NoError(t, err)

	// Copy contents
	err = copyRepositoryContents(srcDir, destDir, &CloneOptions{})
	require.NoError(t, err)

	// Verify files were copied
	destFile := filepath.Join(destDir, "test.txt")
	content, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	destSubFile := filepath.Join(destDir, "subdir", "subtest.txt")
	content, err = os.ReadFile(destSubFile)
	require.NoError(t, err)
	assert.Equal(t, "sub content", string(content))

	// Note: copyRepositoryContents intentionally swallows errors in OnError callback
	// to continue copying even if some files fail. This is by design.
}

func TestCloneOptions(t *testing.T) {
	opts := &CloneOptions{
		RawPaths:   []string{"src", "docs"},
		Username:   "testuser",
		Token:      "testtoken",
		SSHKey:     "test-ssh-key",
		AuthType:   "basic",
		Revision:   "main",
		Cone:       true,
		Verbosity:  "quiet",
		RetryCount: 10,
		RetryDelay: 500 * time.Millisecond,
	}

	// Verify all fields are accessible
	assert.Equal(t, []string{"src", "docs"}, opts.RawPaths)
	assert.Equal(t, "testuser", opts.Username)
	assert.Equal(t, "testtoken", opts.Token)
	assert.Equal(t, "test-ssh-key", opts.SSHKey)
	assert.Equal(t, "basic", opts.AuthType)
	assert.Equal(t, "main", opts.Revision)
	assert.True(t, opts.Cone)
	assert.Equal(t, "quiet", opts.Verbosity)
	assert.Equal(t, 10, opts.RetryCount)
	assert.Equal(t, 500*time.Millisecond, opts.RetryDelay)
}

func TestCloneOptionsGitLogArgs(t *testing.T) {
	assert.Equal(t, []string{"--quiet"}, (&CloneOptions{Verbosity: "quiet"}).gitLogArgs())
	assert.Nil(t, (&CloneOptions{Verbosity: "normal"}).gitLogArgs())
	assert.Equal(t, []string{"--verbose"}, (&CloneOptions{Verbosity: "verbose"}).gitLogArgs())
	assert.Equal(t, []string{"--verbose"}, (&CloneOptions{}).gitLogArgs())
}

func TestCloneOptionsRetrySettings(t *testing.T) {
	count, delay := (&CloneOptions{}).retrySettings()
	assert.Equal(t, CloneRetryOnFailureMaxAttempts, count)
	assert.Equal(t, CloneRetryOnFailureBaseDelay, delay)

	count, delay = (&CloneOptions{RetryCount: 10, RetryDelay: 500 * time.Millisecond}).retrySettings()
	assert.Equal(t, 10, count)
	assert.Equal(t, 500*time.Millisecond, delay)

	count, delay = (&CloneOptions{RetryCount: 1000, RetryDelay: time.Hour}).retrySettings()
	assert.Equal(t, CloneRetryCountCap, count)
	assert.Equal(t, CloneRetryDelayCap, delay)
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

func TestSetupCredentialStore(t *testing.T) {
	t.Run("https with username and token creates credential file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		uri := &url.URL{
			Scheme: "https",
			Host:   "gitlab.com",
			Path:   "/group/repo.git",
			User:   url.UserPassword("myuser", "mytoken"),
		}

		cleanup := setupCredentialStore(uri)
		defer cleanup()

		// Verify git config was created with credential helper
		gitconfig, err := os.ReadFile(filepath.Join(tmpDir, ".gitconfig"))
		require.NoError(t, err)
		assert.Contains(t, string(gitconfig), "store --file")
	})

	t.Run("ssh scheme does not create credential file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		uri := &url.URL{
			Scheme: "ssh",
			Host:   "gitlab.com",
			User:   url.User("git"),
		}

		cleanup := setupCredentialStore(uri)
		defer cleanup()

		_, err := os.Stat(filepath.Join(tmpDir, ".gitconfig"))
		assert.True(t, os.IsNotExist(err), "git config should not be created for SSH")
	})

	t.Run("no credentials does not create credential file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		uri := &url.URL{
			Scheme: "https",
			Host:   "gitlab.com",
		}

		cleanup := setupCredentialStore(uri)
		defer cleanup()

		_, err := os.Stat(filepath.Join(tmpDir, ".gitconfig"))
		assert.True(t, os.IsNotExist(err), "git config should not be created without credentials")
	})
}

func TestCloneError_Reason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want testkube.StopReason
	}{
		{
			name: "a repository without a credential",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("fatal: could not read Username for 'https://github.com': terminal prompts disabled")),
			want: testkube.StopReasonGitAuthFailed,
		},
		{
			name: "a credential that the server refused",
			err:  gitOutputError("remote: Invalid username or token.", "fatal: Authentication failed for 'https://github.com/org/private.git/'"),
			want: testkube.StopReasonGitAuthFailed,
		},
		{
			name: "an SSH key that the server refused",
			err:  gitOutputError("git@github.com: Permission denied (publickey).", "fatal: Could not read from remote repository."),
			want: testkube.StopReasonGitAuthFailed,
		},
		{
			name: "a token without access to the repository",
			err:  gitOutputError("fatal: unable to access 'https://github.com/org/private.git/': The requested URL returned error: HTTP 403"),
			want: testkube.StopReasonGitAuthFailed,
		},
		{
			name: "a repository that does not exist",
			err:  gitOutputError("fatal: repository 'https://github.com/org/absent.git/' not found"),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a revision that does not exist",
			err:  gitOutputError("fatal: couldn't find remote ref refs/heads/absent"),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a host that does not resolve is unreachable",
			err:  gitOutputError("fatal: unable to access 'https://git.invalid/': Could not resolve host: git.invalid"),
			want: testkube.StopReasonGitUnreachable,
		},
		{
			name: "a server that refuses the connection is unreachable",
			err:  gitOutputError("fatal: unable to access 'https://git.example.com/org/repo.git/': Failed to connect to git.example.com port 443: Connection refused"),
			want: testkube.StopReasonGitUnreachable,
		},
		{
			name: "an error that git did not write is a clone failure, even with the text of a refusal",
			err:  errors.New("setting up authentication: GitHub App token: Authentication failed"),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a token that git sends and the server refuses with 403",
			err:  gitOutputError("fatal: unable to access 'https://github.com/org/private.git/': The requested URL returned error: 403"),
			want: testkube.StopReasonGitAuthFailed,
		},
		{
			name: "a branch that the remote does not have",
			err:  gitOutputError("warning: Could not find remote branch absent to clone.", "fatal: Remote branch absent not found in upstream origin"),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a revision that checkout does not know",
			err:  fmt.Errorf("checking out revision: %w", gitOutputError("error: pathspec 'absent' did not match any file(s) known to git")),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a URI that does not parse",
			err:  errors.New("invalid repository URI: parse \"://bad\": missing protocol scheme"),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a toolkit image without git has no code",
			err:  fmt.Errorf("error cloning repository: %w", errors.New("exec: \"git\": executable file not found in $PATH")),
			want: "",
		},
		{
			name: "a certificate that git does not trust has no code",
			err:  gitOutputError("fatal: unable to access 'https://git.example.com/org/repo.git/': SSL certificate problem: unable to get local issuer certificate"),
			want: "",
		},
		{
			name: "a server error from a proxy has no code",
			err:  gitOutputError("fatal: unable to access 'https://git.example.com/org/repo.git/': The requested URL returned error: 502"),
			want: "",
		},
		{
			name: "a text of a refusal outside the output of git does not count",
			err:  fmt.Errorf("Authentication failed: %w", gitOutputError("fatal: repository 'https://github.com/org/absent.git/' not found")),
			want: testkube.StopReasonGitCloneFailed,
		},
		{
			name: "a text of the network outside the output of git does not count",
			err:  fmt.Errorf("connection refused: %w", gitOutputError("fatal: couldn't find remote ref refs/heads/absent")),
			want: testkube.StopReasonGitCloneFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, (&cloneError{err: tt.err}).Reason())
		})
	}
}

// gitOutputError builds the error that runGit returns when git writes the lines to its standard error
// and exits with the code 128.
func gitOutputError(lines ...string) error {
	diagnostic := &diagnosticLine{diagnostics: gitDiagnostics}
	for _, line := range lines {
		_, _ = diagnostic.Write([]byte(line + "\n"))
	}
	return &CommandError{Err: errors.New("exit status 128"), Line: diagnostic.Line(), Note: diagnostic.note}
}

func TestCloneMessage(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		err  error
		want string
	}{
		{
			name: "a repository without credentials says that git received none",
			uri:  "https://github.com/org/private.git",
			err:  fmt.Errorf("error cloning repository: initializing sparse repository: %w", gitOutputError("fatal: could not read Username for 'https://github.com': terminal prompts disabled")),
			want: "Git cannot authenticate to github.com: the repository needs credentials, and git received none.",
		},
		{
			name: "refused credentials say that the server refused them",
			uri:  "https://github.com/org/private.git",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("fatal: Authentication failed for 'https://github.com/org/private.git/'")),
			want: "Git cannot authenticate to github.com: the server refused the credentials.",
		},
		{
			name: "a token without access says that the server refused the credentials",
			uri:  "https://github.com/org/private.git",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("fatal: unable to access 'https://github.com/org/private.git/': The requested URL returned error: HTTP 403")),
			want: "Git cannot authenticate to github.com: the server refused the credentials.",
		},
		{
			name: "a refusal in another case of letters is found",
			uri:  "https://github.com/org/private.git",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("remote: Invalid username or password.")),
			want: "Git cannot authenticate to github.com: the server refused the credentials.",
		},
		{
			name: "a refused SSH key in the line of git says that the server refused the key",
			uri:  "git@gitlab.com:org/private.git",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("fatal: Could not read from remote repository. Permission denied (publickey).")),
			want: "Git cannot authenticate to gitlab.com: the server refused the SSH key.",
		},
		{
			name: "a refused SSH key on its own line says that the server refused the key",
			uri:  "git@github.com:org/private.git",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("git@github.com: Permission denied (publickey).", "fatal: Could not read from remote repository.")),
			want: "Git cannot authenticate to github.com: the server refused the SSH key.",
		},
		{
			name: "another failure keeps the line of git without its severity",
			uri:  "https://github.com/org/absent.git",
			err:  fmt.Errorf("error cloning repository: %w", gitOutputError("fatal: repository 'https://github.com/org/absent.git/' not found")),
			want: "repository 'https://github.com/org/absent.git/' not found",
		},
		{
			name: "a failure without a line of git keeps the error",
			uri:  "https://github.com/org/repo.git",
			err:  errors.New("setting up SSH key: invalid key"),
			want: "setting up SSH key: invalid key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cloneMessage(tt.uri, tt.err))
		})
	}
}

type fixtureGitLabAccess struct {
	url, revision string
	err           error
}

func (f *fixtureGitLabAccess) GetGitAccess(_ context.Context, url, revision string) (*cloud.GetGitAccessResponse, error) {
	f.url = url
	f.revision = revision
	return &cloud.GetGitAccessResponse{Url: "https://relay.example.test/git-relay/lease/repo.git", Username: "testkube", Token: "temporary-lease-token"}, f.err
}

// Checkout requests the exact revision and receives relay access instead of a provider token.
func TestGitLabCheckoutAuthentication(t *testing.T) {
	uri, err := url.Parse("https://gitlab.example.test/group/nested/repo.git")
	require.NoError(t, err)
	client := &fixtureGitLabAccess{}
	opts := &CloneOptions{AuthType: "gitlab", Revision: "exact-admitted-sha"}
	require.NoError(t, setupGitLabAuthentication(t.Context(), uri, opts, client))
	require.Equal(t, "exact-admitted-sha", client.revision)
	require.Equal(t, "https://gitlab.example.test/group/nested/repo.git", client.url)
	require.Equal(t, "relay.example.test", uri.Host)
	token, ok := uri.User.Password()
	require.True(t, ok)
	require.Equal(t, "temporary-lease-token", token)
	client.err = errors.New("unsupported RPC")
	err = setupGitLabAuthentication(t.Context(), uri, opts, client)
	require.ErrorContains(t, err, "update the control plane and Runner")
}

// A branch advancing after admission must not change the commit checked out by the toolkit.
func TestCloneUsesAdmittedCommitAfterBranchAdvances(t *testing.T) {
	source := t.TempDir()
	git := func(dir string, args ...string) string {
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		return strings.TrimSpace(string(output))
	}
	git(source, "init", "--initial-branch=main")
	git(source, "config", "user.name", "Fixture")
	git(source, "config", "user.email", "fixture@example.test")
	require.NoError(t, os.WriteFile(filepath.Join(source, "commit.txt"), []byte("admitted"), 0600))
	git(source, "add", "commit.txt")
	git(source, "commit", "-m", "initial")
	admitted := git(source, "rev-parse", "HEAD")
	require.NoError(t, os.WriteFile(filepath.Join(source, "commit.txt"), []byte("newer"), 0600))
	git(source, "commit", "-am", "advance")
	destination := filepath.Join(t.TempDir(), "checkout")
	require.NoError(t, performFullClone((&url.URL{Scheme: "file", Path: source}).String(), destination, nil, nil, &CloneOptions{Revision: admitted, Verbosity: "quiet"}))
	require.Equal(t, admitted, git(destination, "rev-parse", "HEAD"))
	data, err := os.ReadFile(filepath.Join(destination, "commit.txt"))
	require.NoError(t, err)
	require.Equal(t, "admitted", string(data))
}

// The production toolkit runs without a writable HOME. Git must still receive
// its relay lease, and cleanup must remove both the configuration and credential.
func TestGitLabCredentialStoreWithoutWritableHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(home, []byte("fixture"), 0600))
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	uri, err := url.Parse("http://testkube:fixture-lease-token@relay.example.test/git-relay/lease/repo.git")
	require.NoError(t, err)
	cleanup, err := setupGitLabCredentialStore(uri)
	require.NoError(t, err)
	defer cleanup()
	config := os.Getenv("GIT_CONFIG_GLOBAL")
	cmd := exec.CommandContext(t.Context(), "git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=http\nhost=relay.example.test\n\n")
	data, err := cmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(data), "username=testkube")
	require.Contains(t, string(data), "password=fixture-lease-token")
	uri.User = nil
	require.NotContains(t, uri.String(), "fixture-lease-token")
	cleanup()
	require.Equal(t, "/dev/null", os.Getenv("GIT_CONFIG_GLOBAL"))
	_, err = os.Stat(filepath.Dir(config))
	require.True(t, os.IsNotExist(err))
}
