package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestPathMatches(t *testing.T) {
	tests := []struct {
		paths    []string
		file     string
		expected bool
	}{
		{[]string{"src"}, "src/main.go", true},
		{[]string{"src"}, "src", true},
		{[]string{"src/"}, "src/main.go", true},
		{[]string{"other"}, "src/main.go", false},
		{[]string{"src", "pkg"}, "pkg/util.go", true},
		{[]string{""}, "anything.go", false},
		{[]string{"src/sub"}, "src/sub/file.go", true},
		{[]string{"src/sub"}, "src/other/file.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.expected, pathMatches(tt.paths, tt.file))
		})
	}
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "refs/heads/main"},
		{"refs/heads/main", "refs/heads/main"},
		{"refs/tags/v1.0", "refs/tags/v1.0"},
		{"", ""},
		{"  develop  ", "refs/heads/develop"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeRef(tt.input))
		})
	}
}

func TestNormalizePaths(t *testing.T) {
	paths := []string{" /a ", "/b/c", "", "///", "d/"}
	assert.Equal(t, []string{"a", "b/c", "d"}, normalizePaths(paths))
}

func TestResolveCredentialValue(t *testing.T) {
	t.Setenv("TK_GIT_USERNAME", "env-user")
	t.Setenv("TK_GIT_TOKEN", "env-token")

	assert.Equal(t, "inline", resolveCredentialValue("inline", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Key: "TK_GIT_USERNAME"},
	}))
	assert.Equal(t, "env-user", resolveCredentialValue("", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Key: "TK_GIT_USERNAME"},
	}))
	assert.Equal(t, "env-token", resolveCredentialValue("", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Key: "TK_GIT_TOKEN"},
	}))
	assert.Equal(t, "", resolveCredentialValue("", nil))
}

func TestAuthClientOptions(t *testing.T) {
	t.Run("basic auth default", func(t *testing.T) {
		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			Username: "user",
			Token:    "token",
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("header auth", func(t *testing.T) {
		authType := testkube.HEADER_ContentGitAuthType
		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			Token:    "token",
			AuthType: &authType,
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("ssh auth", func(t *testing.T) {
		_, err := authClientOptions(&testkube.TestTriggerContentGit{
			SshKey: "invalid-private-key",
		})
		require.Error(t, err)
	})
}

func TestIsGitContentTrigger(t *testing.T) {
	resource := testkube.CONTENT_TestTriggerResources

	tests := []struct {
		name     string
		trigger  testkube.TestTrigger
		expected bool
	}{
		{
			name: "valid git content trigger",
			trigger: testkube.TestTrigger{
				Resource: &resource,
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: true,
		},
		{
			name: "disabled trigger",
			trigger: testkube.TestTrigger{
				Disabled: true,
				Resource: &resource,
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: false,
		},
		{
			name: "valid git content trigger via resourceRef",
			trigger: testkube.TestTrigger{
				ResourceRef: &testkube.TestTriggerResourceRef{
					Kind: "Content",
				},
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: true,
		},
		{
			name: "resourceRef non-content",
			trigger: testkube.TestTrigger{
				ResourceRef: &testkube.TestTriggerResourceRef{
					Kind: "Deployment",
				},
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: false,
		},
		{
			name:     "no resource",
			trigger:  testkube.TestTrigger{},
			expected: false,
		},
		{
			name: "no content selector",
			trigger: testkube.TestTrigger{
				Resource: &resource,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isGitContentTrigger(tt.trigger))
		})
	}
}
