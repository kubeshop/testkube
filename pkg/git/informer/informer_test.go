package informer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
)

type stubTestTriggerClient struct {
	listFn func(ctx context.Context, environmentID string, options testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error)
}

func (s stubTestTriggerClient) Get(context.Context, string, string, string) (*testkube.TestTrigger, error) {
	return nil, nil
}

func (s stubTestTriggerClient) List(ctx context.Context, environmentID string, options testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error) {
	if s.listFn != nil {
		return s.listFn(ctx, environmentID, options, namespace)
	}
	return nil, nil
}

func (s stubTestTriggerClient) Update(context.Context, string, testkube.TestTrigger) error {
	return nil
}

func (s stubTestTriggerClient) Create(context.Context, string, testkube.TestTrigger) error {
	return nil
}

func (s stubTestTriggerClient) Delete(context.Context, string, string, string) error {
	return nil
}

func (s stubTestTriggerClient) DeleteAll(context.Context, string, string) (uint32, error) {
	return 0, nil
}

func (s stubTestTriggerClient) DeleteByLabels(context.Context, string, string, string) (uint32, error) {
	return 0, nil
}

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

func TestPathMatchesNormalized(t *testing.T) {
	paths := normalizePaths([]string{"src/", " pkg "})
	assert.True(t, pathMatchesNormalized(paths, "src/main.go"))
	assert.True(t, pathMatchesNormalized(paths, "pkg/util.go"))
	assert.False(t, pathMatchesNormalized(paths, "internal/main.go"))
}

func TestNormalizeRefs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"main", []string{"refs/heads/main", "refs/tags/main"}},
		{"refs/heads/main", []string{"refs/heads/main"}},
		{"refs/tags/v1.0", []string{"refs/tags/v1.0"}},
		{"", nil},
		{"  develop  ", []string{"refs/heads/develop", "refs/tags/develop"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeRefs(tt.input))
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
	t.Setenv("GIT_CREDENTIALS_USERNAME", "env-user-from-name-key")
	t.Setenv("git-credentials", "env-user-from-name")

	assert.Equal(t, "inline", resolveCredentialValue("inline", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Key: "TK_GIT_USERNAME"},
	}))
	assert.Equal(t, "env-user", resolveCredentialValue("", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Key: "TK_GIT_USERNAME"},
	}))
	assert.Equal(t, "env-token", resolveCredentialValue("", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Key: "TK_GIT_TOKEN"},
	}))
	assert.Equal(t, "env-user-from-name-key", resolveCredentialValue("", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "git-credentials", Key: "username"},
	}))
	assert.NotEqual(t, "env-user-from-name", resolveCredentialValue("", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "git-credentials", Key: "username"},
	}))
	assert.Equal(t, "env-user-from-name", resolveCredentialValue("", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Name: "git-credentials"},
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
		testPrivateKey := generateTestPrivateKey(t)

		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			SshKey: testPrivateKey,
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)

		_, err = authClientOptions(&testkube.TestTriggerContentGit{
			SshKey: "invalid-private-key",
		})
		require.Error(t, err)
	})
}

func generateTestPrivateKey(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NotNil(t, encoded)

	return string(encoded)
}

func TestNormalizeOptions(t *testing.T) {
	assert.Equal(t, Options{
		RepoDepth:          0,
		ListTimeoutSeconds: 15,
		MaxCommitsScan:     0,
		PullRetries:        0,
		PullRetryDelay:     0,
	}, normalizeOptions(Options{
		RepoDepth:          -1,
		ListTimeoutSeconds: 0,
		MaxCommitsScan:     -1,
		PullRetries:        -1,
		PullRetryDelay:     -time.Second,
	}))
}

func TestCloneAndPullOptions_UseRepoDepth(t *testing.T) {
	gitConfig := &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/example/repo.git",
		Revision: "main",
	}
	opts := Options{RepoDepth: 77}

	cloneOpts, err := cloneOptions(gitConfig, opts)
	require.NoError(t, err)
	assert.Equal(t, 77, cloneOpts.Depth)

	pullOpts, err := pullOptions(gitConfig, opts)
	require.NoError(t, err)
	assert.Equal(t, 77, pullOpts.Depth)
}

func TestCloneAndPullOptions_TestkubeRepository(t *testing.T) {
	gitConfig := &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: "main",
	}
	opts := Options{
		RepoDepth:          50,
		ListTimeoutSeconds: 30,
	}

	cloneOpts, err := cloneOptions(gitConfig, opts)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/kubeshop/testkube.git", cloneOpts.URL)
	assert.Equal(t, 50, cloneOpts.Depth)
	assert.Equal(t, "refs/heads/main", cloneOpts.ReferenceName.String())

	pullOpts, err := pullOptions(gitConfig, opts)
	require.NoError(t, err)
	assert.Equal(t, 50, pullOpts.Depth)
	assert.Equal(t, "refs/heads/main", pullOpts.ReferenceName.String())
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

func TestReconcile_StopsPromptlyOnCancel(t *testing.T) {
	listCalled := make(chan struct{}, 1)
	client := stubTestTriggerClient{
		listFn: func(ctx context.Context, environmentID string, options testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error) {
			select {
			case listCalled <- struct{}{}:
			default:
			}
			return nil, nil
		},
	}
	informer := NewInformer(client, nil, "testkube", "", Options{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		informer.Reconcile(ctx)
		close(done)
	}()

	select {
	case <-listCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("initial list was not called")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("reconcile did not stop promptly after cancel")
	}
}
