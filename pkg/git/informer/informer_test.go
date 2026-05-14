package informer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
	sshknownhosts "golang.org/x/crypto/ssh/knownhosts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/newclients/workflowtriggerclient"
)

type stubTestTriggerClient struct {
	listFn func(ctx context.Context, environmentID string, options testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error)
}

type stubWorkflowTriggerClient struct {
	listFn func(ctx context.Context, environmentID string, options workflowtriggerclient.ListOptions, namespace string) ([]testkube.WorkflowTrigger, error)
}

type stubMatcher struct {
	matchTestTriggerFn     func(context.Context, string, string) error
	matchWorkflowTriggerFn func(context.Context, string, string) error
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

func (s stubWorkflowTriggerClient) Get(context.Context, string, string, string) (*testkube.WorkflowTrigger, error) {
	return nil, nil
}

func (s stubWorkflowTriggerClient) List(ctx context.Context, environmentID string, options workflowtriggerclient.ListOptions, namespace string) ([]testkube.WorkflowTrigger, error) {
	if s.listFn != nil {
		return s.listFn(ctx, environmentID, options, namespace)
	}
	return nil, nil
}

func (s stubWorkflowTriggerClient) Update(context.Context, string, testkube.WorkflowTrigger) error {
	return nil
}

func (s stubWorkflowTriggerClient) Create(context.Context, string, testkube.WorkflowTrigger) error {
	return nil
}

func (s stubWorkflowTriggerClient) Delete(context.Context, string, string, string) error {
	return nil
}

func (s stubWorkflowTriggerClient) DeleteAll(context.Context, string, string) (uint32, error) {
	return 0, nil
}

func (s stubWorkflowTriggerClient) DeleteByLabels(context.Context, string, string, string) (uint32, error) {
	return 0, nil
}

func (s stubMatcher) MatchGitTrigger(ctx context.Context, triggerName, namespace string) error {
	if s.matchTestTriggerFn != nil {
		return s.matchTestTriggerFn(ctx, triggerName, namespace)
	}
	return nil
}

func (s stubMatcher) MatchGitWorkflowTrigger(ctx context.Context, triggerName, namespace string) error {
	if s.matchWorkflowTriggerFn != nil {
		return s.matchWorkflowTriggerFn(ctx, triggerName, namespace)
	}
	return nil
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
		{"0123456789abcdef0123456789abcdef01234567", []string{"0123456789abcdef0123456789abcdef01234567"}},
		{"", nil},
		{"  develop  ", []string{"refs/heads/develop", "refs/tags/develop"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeRefs(tt.input))
		})
	}
}

func TestNormalizeRevision(t *testing.T) {
	assert.Equal(t, "", normalizeRevision(""))
	assert.Equal(t, "main", normalizeRevision(" main "))
	assert.Equal(t, "refs/heads/feature/a", normalizeRevision(" refs/heads/feature/a\t"))
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

func TestResolveCredentialValue_WithKubeClient(t *testing.T) {
	informer := NewInformer(stubTestTriggerClient{}, nil, nil, "testkube", "", Options{
		KubeClient: fake.NewSimpleClientset(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "git-secret", Namespace: "testkube"},
				Data:       map[string][]byte{"token": []byte("secret-token")},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "git-config", Namespace: "testkube"},
				Data:       map[string]string{"username": "config-user"},
			},
		),
	})

	assert.Equal(t, "secret-token", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "git-secret", Key: "token"},
	}))
	assert.Equal(t, "config-user", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Name: "git-config", Key: "username"},
	}))
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
		signer, err := gossh.ParsePrivateKey([]byte(testPrivateKey))
		require.NoError(t, err)
		knownHostsFile := filepath.Join(t.TempDir(), "known_hosts")
		entry := sshknownhosts.Line([]string{"test-host"}, signer.PublicKey())
		require.NoError(t, os.WriteFile(knownHostsFile, []byte(entry+"\n"), 0o600))
		t.Setenv("SSH_KNOWN_HOSTS", knownHostsFile)

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

	t.Run("ssh auth fails when known_hosts is unavailable", func(t *testing.T) {
		testPrivateKey := generateTestPrivateKey(t)
		t.Setenv("SSH_KNOWN_HOSTS", filepath.Join(t.TempDir(), "missing_known_hosts"))

		_, err := authClientOptions(&testkube.TestTriggerContentGit{
			SshKey: testPrivateKey,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssh auth requires known_hosts-based host key verification")
	})

	t.Run("reject unsupported tokenFrom fieldRef source", func(t *testing.T) {
		_, err := authClientOptions(&testkube.TestTriggerContentGit{
			TokenFrom: &testkube.EnvVarSource{
				FieldRef: &testkube.FieldRef{FieldPath: "metadata.name"},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tokenFrom")
		assert.Contains(t, err.Error(), "fieldRef")
	})

	t.Run("reject unsupported sshKeyFrom resourceFieldRef source", func(t *testing.T) {
		_, err := authClientOptions(&testkube.TestTriggerContentGit{
			SshKeyFrom: &testkube.EnvVarSource{
				ResourceFieldRef: &testkube.ResourceFieldRef{Resource: "limits.cpu"},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sshKeyFrom")
		assert.Contains(t, err.Error(), "resourceFieldRef")
	})

}

func TestCloneAndPullOptions_CommitSHARevision(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	opts := Options{RepoDepth: 1}

	cloneOpts, err := cloneOptions(&testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: sha,
	}, opts)
	require.NoError(t, err)
	assert.Empty(t, cloneOpts.ReferenceName)

	pullOpts, err := pullOptions(&testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: sha,
	}, opts)
	require.NoError(t, err)
	assert.Empty(t, pullOpts.ReferenceName)
}

func TestCommitSHARevisionWithPathsIsNotWatchable(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	trigger := testkube.TestTrigger{
		Name:      "trigger-a",
		Namespace: "default",
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: sha,
				Paths:    []string{"pkg/git"},
			},
		},
	}

	informer := NewInformer(stubTestTriggerClient{}, nil, nil, "testkube", "", Options{})

	key := triggerKey(testTriggerSource, trigger.Namespace, trigger.Name)
	changed, err := informer.hasNewMatchingCommit(context.Background(), key, trigger)
	require.NoError(t, err)
	assert.False(t, changed)

	assert.Equal(t, sha, informer.commits[key])

	// Even when local baseline drifts, SHA-pinned triggers stay non-watchable.
	informer.commits[key] = "drifted"
	changed, err = informer.hasNewMatchingCommit(context.Background(), key, trigger)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, sha, informer.commits[key])
}

func TestShouldAdvanceBaselineOnScanError(t *testing.T) {
	assert.True(t, shouldAdvanceBaselineOnScanError(plumbing.ErrObjectNotFound))
	assert.True(t, shouldAdvanceBaselineOnScanError(errors.Join(plumbing.ErrObjectNotFound, errors.New("wrapped"))))
	assert.False(t, shouldAdvanceBaselineOnScanError(errors.New("network timeout")))
}

func TestSleepWithContext(t *testing.T) {
	t.Run("returns canceled when context is canceled before delay", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		start := time.Now()
		err := sleepWithContext(ctx, time.Second)
		require.ErrorIs(t, err, context.Canceled)
		assert.Less(t, time.Since(start), 100*time.Millisecond)
	})

	t.Run("waits for delay when context remains active", func(t *testing.T) {
		start := time.Now()
		err := sleepWithContext(context.Background(), 20*time.Millisecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, time.Since(start), 20*time.Millisecond)
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
		ReconcileInterval:  time.Minute,
		RepoDepth:          0,
		ListTimeoutSeconds: 15,
		MaxCommitsScan:     0,
		PullRetries:        0,
		PullRetryDelay:     0,
	}, normalizeOptions(Options{
		ReconcileInterval:  -time.Second,
		RepoDepth:          -1,
		ListTimeoutSeconds: 0,
		MaxCommitsScan:     -1,
		PullRetries:        -1,
		PullRetryDelay:     -time.Second,
	}))
}

func TestRepositoryOriginMatches(t *testing.T) {
	t.Run("matches expected origin URL", func(t *testing.T) {
		repo := initTestRepoWithOrigin(t, "https://example.com/repo-a.git")
		assert.True(t, repositoryOriginMatches(repo, "https://example.com/repo-a.git"))
	})

	t.Run("returns false for different origin URL", func(t *testing.T) {
		repo := initTestRepoWithOrigin(t, "https://example.com/repo-a.git")
		assert.False(t, repositoryOriginMatches(repo, "https://example.com/repo-b.git"))
	})

	t.Run("returns false when origin remote does not exist", func(t *testing.T) {
		dir := t.TempDir()
		repo, err := git.PlainInit(dir, false)
		require.NoError(t, err)
		assert.False(t, repositoryOriginMatches(repo, "https://example.com/repo-a.git"))
	})
}

func initTestRepoWithOrigin(t *testing.T, originURL string) *git.Repository {
	t.Helper()

	repoDir := filepath.Join(t.TempDir(), "repo")
	err := os.MkdirAll(repoDir, 0o755)
	require.NoError(t, err)

	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{originURL},
	})
	require.NoError(t, err)

	return repo
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

func TestGitInformerConfig_MultiplePathFilters(t *testing.T) {
	gitConfig := &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: "main",
		Paths:    []string{"/test", "/pkg"},
	}

	refs := normalizeRefs(gitConfig.Revision)
	assert.Contains(t, refs, "refs/heads/main")
	assert.Equal(t, "https://github.com/kubeshop/testkube.git", gitConfig.Uri)

	normalizedPaths := normalizePaths(gitConfig.Paths)
	assert.Equal(t, []string{"test", "pkg"}, normalizedPaths)
	assert.True(t, pathMatchesNormalized(normalizedPaths, "test/testkube/ci/crd-workflow/api-server-build-lint.yaml"))
	assert.True(t, pathMatchesNormalized(normalizedPaths, "pkg/triggers/git_trigger.go"))
	assert.False(t, pathMatchesNormalized(normalizedPaths, "cmd/api-server/main.go"))
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
				Event:    "modified",
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
				Event:    "modified",
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
				Event: "modified",
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
				Event: "modified",
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
				Event:    "modified",
				Resource: &resource,
			},
			expected: false,
		},
		{
			name: "non-modified event",
			trigger: testkube.TestTrigger{
				Event:    "created",
				Resource: &resource,
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
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

func TestIsGitContentWorkflowTrigger(t *testing.T) {
	tests := []struct {
		name     string
		trigger  testkube.WorkflowTrigger
		expected bool
	}{
		{
			name: "valid git workflow trigger",
			trigger: testkube.WorkflowTrigger{
				When: testkube.WorkflowTriggerWhen{
					Event: "modified",
					Git:   &testkube.TestTriggerContentGit{Uri: "https://github.com/example/repo.git"},
				},
			},
			expected: true,
		},
		{
			name: "disabled workflow trigger",
			trigger: testkube.WorkflowTrigger{
				Disabled: true,
				When: testkube.WorkflowTriggerWhen{
					Event: "modified",
					Git:   &testkube.TestTriggerContentGit{Uri: "https://github.com/example/repo.git"},
				},
			},
			expected: false,
		},
		{
			name: "watch kind non-content",
			trigger: testkube.WorkflowTrigger{
				Watch: &testkube.WorkflowTriggerWatch{
					Resource: testkube.WorkflowTriggerResource{Kind: "deployment"},
				},
				When: testkube.WorkflowTriggerWhen{
					Event: "modified",
					Git:   &testkube.TestTriggerContentGit{Uri: "https://github.com/example/repo.git"},
				},
			},
			expected: false,
		},
		{
			name: "non-modified workflow event",
			trigger: testkube.WorkflowTrigger{
				When: testkube.WorkflowTriggerWhen{
					Event: "deleted",
					Git:   &testkube.TestTriggerContentGit{Uri: "https://github.com/example/repo.git"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isGitContentWorkflowTrigger(tt.trigger))
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
	informer := NewInformer(client, nil, nil, "testkube", "", Options{})

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

func TestUpdateRepositories_MatchesWorkflowGitTrigger(t *testing.T) {
	const revision = "0123456789abcdef0123456789abcdef01234567"
	workflowTrigger := testkube.WorkflowTrigger{
		Name:      "workflow-a",
		Namespace: "testkube",
		When: testkube.WorkflowTriggerWhen{
			Event: "modified",
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: revision,
			},
		},
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "content"},
		},
	}

	var matched []string
	informer := NewInformer(
		stubTestTriggerClient{},
		stubWorkflowTriggerClient{
			listFn: func(ctx context.Context, environmentID string, options workflowtriggerclient.ListOptions, namespace string) ([]testkube.WorkflowTrigger, error) {
				return []testkube.WorkflowTrigger{workflowTrigger}, nil
			},
		},
		stubMatcher{
			matchWorkflowTriggerFn: func(_ context.Context, triggerName, namespace string) error {
				matched = append(matched, namespace+"/"+triggerName)
				return nil
			},
		},
		"testkube",
		"",
		Options{},
	)
	workflowKey := triggerKey(workflowTriggerSource, workflowTrigger.Namespace, workflowTrigger.Name)
	informer.commits[workflowKey] = "old"

	informer.updateRepositories(context.Background())

	assert.Equal(t, []string{"testkube/workflow-a"}, matched)
}

func TestRestoreCommitBaseline(t *testing.T) {
	informer := NewInformer(stubTestTriggerClient{}, nil, nil, "testkube", "", Options{})

	informer.commits["with-previous"] = "new-hash"
	informer.restoreCommitBaseline("with-previous", "old-hash", true)
	assert.Equal(t, "old-hash", informer.commits["with-previous"])

	informer.commits["without-previous"] = "new-hash"
	informer.restoreCommitBaseline("without-previous", "", false)
	_, exists := informer.commits["without-previous"]
	assert.False(t, exists)
}

func TestUpdateRepositories_RestoresBaselineWhenMatchFails(t *testing.T) {
	const revision = "0123456789abcdef0123456789abcdef01234567"
	resource := testkube.CONTENT_TestTriggerResources
	trigger := testkube.TestTrigger{
		Name:      "trigger-a",
		Namespace: "testkube",
		Event:     "modified",
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: revision,
			},
		},
	}

	key := triggerKey(testTriggerSource, trigger.Namespace, trigger.Name)
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, _ string) ([]testkube.TestTrigger, error) {
				return []testkube.TestTrigger{trigger}, nil
			},
		},
		nil,
		stubMatcher{
			matchTestTriggerFn: func(context.Context, string, string) error {
				return errors.New("temporary matcher failure")
			},
		},
		"testkube",
		"",
		Options{},
	)
	informer.commits[key] = "old-head"

	informer.updateRepositories(context.Background())

	assert.Equal(t, "old-head", informer.commits[key])
}

func TestUpdateRepositories_UsesWatcherNamespaces(t *testing.T) {
	seen := make([]string, 0)
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error) {
				seen = append(seen, namespace)
				return nil, nil
			},
		},
		nil,
		nil,
		"testkube",
		"",
		Options{WatcherNamespaces: "team-a, team-b"},
	)

	informer.updateRepositories(context.Background())

	assert.ElementsMatch(t, []string{"team-a", "team-b"}, seen)
}

func TestUpdateRepositories_DefaultsToAllNamespacesWhenWatcherNamespacesEmpty(t *testing.T) {
	seen := make([]string, 0)
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error) {
				seen = append(seen, namespace)
				return nil, nil
			},
		},
		nil,
		nil,
		"testkube",
		"",
		Options{},
	)

	informer.updateRepositories(context.Background())

	assert.Equal(t, []string{allNamespacesMarker}, seen)
}

func TestUpdateRepositories_ContinuesWhenNamespaceListFails(t *testing.T) {
	resource := testkube.CONTENT_TestTriggerResources
	trigger := testkube.TestTrigger{
		Name:      "trigger-a",
		Namespace: "team-b",
		Event:     "modified",
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: "main",
			},
		},
	}

	var matched []string
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error) {
				if namespace == "team-a" {
					return nil, errors.New("forbidden")
				}
				return []testkube.TestTrigger{trigger}, nil
			},
		},
		nil,
		stubMatcher{
			matchTestTriggerFn: func(_ context.Context, triggerName, namespace string) error {
				matched = append(matched, namespace+"/"+triggerName)
				return nil
			},
		},
		"testkube",
		"",
		Options{WatcherNamespaces: "team-a,team-b"},
	)
	informer.commits[triggerKey(testTriggerSource, trigger.Namespace, trigger.Name)] = "old"
	keepKey := triggerKey(testTriggerSource, "team-a", "stale-a")
	removeKey := triggerKey(testTriggerSource, "team-b", "stale-b")
	informer.commits[keepKey] = "old-a"
	informer.commits[removeKey] = "old-b"

	keepPath := triggerRepositoryPathFromKey(keepKey)
	removePath := triggerRepositoryPathFromKey(removeKey)
	require.NoError(t, os.MkdirAll(keepPath, 0o755))
	require.NoError(t, os.MkdirAll(removePath, 0o755))

	informer.updateRepositories(context.Background())

	assert.Equal(t, []string{"team-b/trigger-a"}, matched)
	assert.Equal(t, "old-a", informer.commits[keepKey])
	_, keepErr := os.Stat(keepPath)
	assert.NoError(t, keepErr)
	_, removeExists := informer.commits[removeKey]
	assert.False(t, removeExists)
	_, removeErr := os.Stat(removePath)
	assert.True(t, os.IsNotExist(removeErr))
}

func TestUpdateRepositories_ContinuesWithWorkflowTriggersWhenAllTestTriggerListsFail(t *testing.T) {
	const revision = "0123456789abcdef0123456789abcdef01234567"

	workflowTrigger := testkube.WorkflowTrigger{
		Name:      "workflow-a",
		Namespace: "team-a",
		When: testkube.WorkflowTriggerWhen{
			Event: "modified",
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: revision,
			},
		},
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "content"},
		},
	}

	var matched []string
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, _ string) ([]testkube.TestTrigger, error) {
				return nil, errors.New("forbidden")
			},
		},
		stubWorkflowTriggerClient{
			listFn: func(_ context.Context, _ string, _ workflowtriggerclient.ListOptions, _ string) ([]testkube.WorkflowTrigger, error) {
				return []testkube.WorkflowTrigger{workflowTrigger}, nil
			},
		},
		stubMatcher{
			matchWorkflowTriggerFn: func(_ context.Context, triggerName, namespace string) error {
				matched = append(matched, namespace+"/"+triggerName)
				return nil
			},
		},
		"testkube",
		"",
		Options{WatcherNamespaces: "team-a"},
	)
	workflowKey := triggerKey(workflowTriggerSource, workflowTrigger.Namespace, workflowTrigger.Name)
	informer.commits[workflowKey] = "old"

	informer.updateRepositories(context.Background())

	assert.Equal(t, []string{"team-a/workflow-a"}, matched)
}

func TestUpdateRepositories_MatchesTestTriggerWithGitPaths(t *testing.T) {
	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	_, err := git.PlainInit(remoteDir, true)
	require.NoError(t, err)

	workDir := filepath.Join(tmpDir, "work")
	workRepo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)
	_, err = workRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteDir},
	})
	require.NoError(t, err)
	worktree, err := workRepo.Worktree()
	require.NoError(t, err)
	require.NoError(t, workRepo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
	))

	commitFile := func(path, content, message string) string {
		fullPath := filepath.Join(workDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
		_, err = worktree.Add(path)
		require.NoError(t, err)
		hash, err := worktree.Commit(message, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "test",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		require.NoError(t, err)
		return hash.String()
	}
	pushMain := func() {
		err = workRepo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs: []config.RefSpec{
				config.RefSpec("refs/heads/main:refs/heads/main"),
			},
		})
		require.NoError(t, err)
	}

	firstHash := commitFile("README.md", "initial\n", "initial")
	pushMain()

	key := triggerKey(testTriggerSource, "testkube", "trigger-a")
	repoDir := triggerRepositoryPathFromKey(key)
	t.Cleanup(func() { _ = os.RemoveAll(repoDir) })
	_ = os.RemoveAll(repoDir)

	localRepo, err := git.PlainClone(repoDir, &git.CloneOptions{
		URL:           remoteDir,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
		SingleBranch:  true,
	})
	require.NoError(t, err)
	require.NoError(t, localRepo.DeleteRemote("origin"))
	// Keep the real test remote first (used for fetch/pull) and include the
	// GitHub URL to satisfy repositoryOriginMatches for kubeshop/testkube.
	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{
			remoteDir,
			"https://github.com/kubeshop/testkube.git",
		},
	})
	require.NoError(t, err)

	_ = commitFile("pkg/triggers/git_trigger.go", "package triggers\n", "pkg change")
	pushMain()

	resource := testkube.CONTENT_TestTriggerResources
	trigger := testkube.TestTrigger{
		Name:      "trigger-a",
		Namespace: "testkube",
		Event:     "modified",
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: "main",
				Paths:    []string{"/test", "/pkg"},
			},
		},
	}

	var matched []string
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(ctx context.Context, environmentID string, options testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error) {
				return []testkube.TestTrigger{trigger}, nil
			},
		},
		nil,
		stubMatcher{
			matchTestTriggerFn: func(_ context.Context, triggerName, namespace string) error {
				matched = append(matched, namespace+"/"+triggerName)
				return nil
			},
		},
		"testkube",
		"",
		Options{},
	)
	informer.commits[key] = firstHash

	informer.updateRepositories(context.Background())

	assert.Equal(t, []string{"testkube/trigger-a"}, matched)
	assert.NotEqual(t, firstHash, informer.commits[key])
}

func TestGitConfigCacheKey_GroupsByNormalizedGitConfigAndNamespace(t *testing.T) {
	authType := testkube.BASIC_ContentGitAuthType
	keyA := gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: "main",
		AuthType: &authType,
		Paths:    []string{"pkg"},
	})
	keyB := gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: "main",
		AuthType: &authType,
		Paths:    []string{"test"},
	})
	keyOtherNamespace := gitConfigCacheKey("team-a", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Revision: "main",
		AuthType: &authType,
	})

	assert.Equal(t, keyA, keyB)
	assert.NotEqual(t, keyA, keyOtherNamespace)
}

func TestUpdateRepositories_CleanupSkipsWorkflowNamespacesWithListErrors(t *testing.T) {
	const revision = "0123456789abcdef0123456789abcdef01234567"

	workflowTrigger := testkube.WorkflowTrigger{
		Name:      "workflow-active",
		Namespace: "team-b",
		When: testkube.WorkflowTriggerWhen{
			Event: "modified",
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Revision: revision,
			},
		},
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "content"},
		},
	}

	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, _ string) ([]testkube.TestTrigger, error) {
				return nil, nil
			},
		},
		stubWorkflowTriggerClient{
			listFn: func(_ context.Context, _ string, _ workflowtriggerclient.ListOptions, namespace string) ([]testkube.WorkflowTrigger, error) {
				if namespace == "team-a" {
					return nil, errors.New("forbidden")
				}
				return []testkube.WorkflowTrigger{workflowTrigger}, nil
			},
		},
		nil,
		"testkube",
		"",
		Options{WatcherNamespaces: "team-a,team-b"},
	)
	informer.commits[triggerKey(workflowTriggerSource, workflowTrigger.Namespace, workflowTrigger.Name)] = "old-active"

	keepKey := triggerKey(workflowTriggerSource, "team-a", "stale-a")
	removeKey := triggerKey(workflowTriggerSource, "team-b", "stale-b")
	informer.commits[keepKey] = "old-a"
	informer.commits[removeKey] = "old-b"

	keepPath := triggerRepositoryPathFromKey(keepKey)
	removePath := triggerRepositoryPathFromKey(removeKey)
	require.NoError(t, os.MkdirAll(keepPath, 0o755))
	require.NoError(t, os.MkdirAll(removePath, 0o755))

	informer.updateRepositories(context.Background())

	assert.Equal(t, "old-a", informer.commits[keepKey])
	_, keepErr := os.Stat(keepPath)
	assert.NoError(t, keepErr)

	_, removeExists := informer.commits[removeKey]
	assert.False(t, removeExists)
	_, removeErr := os.Stat(removePath)
	assert.True(t, os.IsNotExist(removeErr))
}
