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
)

type stubTestTriggerClient struct {
	listFn func(ctx context.Context, environmentID string, options testtriggerclient.ListOptions, namespace string) ([]testkube.TestTrigger, error)
}

type stubMatcher struct {
	matchTestTriggerFn func(context.Context, string, string) error
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

func (s stubMatcher) MatchGitTrigger(ctx context.Context, triggerName, namespace string, gitMeta map[string]string) error {
	if s.matchTestTriggerFn != nil {
		return s.matchTestTriggerFn(ctx, triggerName, namespace)
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
	toBoolPtr := func(v bool) *bool { return &v }
	informer := NewInformer(stubTestTriggerClient{}, nil, "testkube", "", Options{
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
	t.Setenv("TOKEN", "env-token-fallback")
	t.Setenv("USERNAME", "env-user-fallback")

	assert.Equal(t, "secret-token", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "git-secret", Key: "token"},
	}))
	assert.Equal(t, "config-user", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Name: "git-config", Key: "username"},
	}))
	assert.Equal(t, "", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "missing-secret", Key: "TOKEN"},
	}))
	assert.Equal(t, "", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Name: "git-config", Key: "USERNAME"},
	}))
	assert.Equal(t, "env-token-fallback", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Name: "missing-secret", Key: "TOKEN", Optional: toBoolPtr(true)},
	}))
	assert.Equal(t, "env-user-fallback", informer.resolveCredentialValue(context.Background(), "", "testkube", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Name: "git-config", Key: "USERNAME", Optional: toBoolPtr(true)},
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
		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			Token:    "token",
			AuthType: string(testkube.HEADER_ContentGitAuthType),
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

func TestCloneAndPullOptions_BranchRef(t *testing.T) {
	opts := Options{RepoDepth: 1}

	cloneOpts, err := cloneOptions(&testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Branches: []string{"main"},
	}, opts)
	require.NoError(t, err)
	assert.Equal(t, plumbing.ReferenceName("refs/heads/main"), cloneOpts.ReferenceName)

	pullOpts, err := pullOptions(&testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Branches: []string{"main"},
	}, opts)
	require.NoError(t, err)
	assert.Equal(t, plumbing.ReferenceName("refs/heads/main"), pullOpts.ReferenceName)
}

func TestEmptyBranchesProduceNoStaticEffectiveRefs(t *testing.T) {
	trigger := testkube.TestTrigger{
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:   "https://github.com/kubeshop/testkube.git",
				Paths: []string{"pkg/git"},
			},
		},
	}

	// With no branches specified, effectiveRefs returns nil and remote refs are resolved dynamically.
	refs := effectiveRefs(trigger.ContentSelector.Git)
	assert.Empty(t, refs)
}

func TestRemoteAllMatchingRefsWithClientOptions_EmptyFiltersWatchAllBranches(t *testing.T) {
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

	require.NoError(t, workRepo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
	))

	worktree, err := workRepo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("hello"), 0o644))
	_, err = worktree.Add("README.md")
	require.NoError(t, err)
	hash, err := worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	require.NoError(t, workRepo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("develop"), hash)))
	require.NoError(t, workRepo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec("refs/heads/main:refs/heads/main"),
			config.RefSpec("refs/heads/develop:refs/heads/develop"),
		},
	}))

	refs, err := remoteAllMatchingRefsWithClientOptions(&testkube.TestTriggerContentGit{Uri: remoteDir}, Options{ListTimeoutSeconds: 15}, nil)
	require.NoError(t, err)

	got := map[string]struct{}{}
	for _, ref := range refs {
		got[ref.Ref] = struct{}{}
	}
	assert.Contains(t, got, "refs/heads/main")
	assert.Contains(t, got, "refs/heads/develop")
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

func TestResolveNamespaces(t *testing.T) {
	t.Run("returns unique trimmed namespaces", func(t *testing.T) {
		namespaces := resolveNamespaces(" team-a,team-b,team-a , ,team-c ", "ignored")
		assert.Equal(t, []string{"team-a", "team-b", "team-c"}, namespaces)
	})

	t.Run("defaults to all namespaces marker when empty", func(t *testing.T) {
		namespaces := resolveNamespaces(" ,  ", "testkube")
		assert.Equal(t, []string{allNamespacesMarker}, namespaces)
	})
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
		Branches: []string{"main"},
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
		Branches: []string{"main"},
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
		Branches: []string{"main"},
		Paths:    []string{"/test", "/pkg"},
	}

	refs := effectiveRefs(gitConfig)
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
				Event:    EventGitPush,
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
				Event:    EventGitPush,
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
				Event: EventGitPush,
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
				Event: EventGitPush,
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
				Event:    EventGitPush,
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

func TestRestoreCommitBaseline(t *testing.T) {
	informer := NewInformer(stubTestTriggerClient{}, nil, "testkube", "", Options{})

	informer.commits["with-previous"] = "new-hash"
	informer.restoreCommitBaseline("with-previous", "old-hash", true)
	assert.Equal(t, "old-hash", informer.commits["with-previous"])

	informer.commits["without-previous"] = "new-hash"
	informer.restoreCommitBaseline("without-previous", "", false)
	_, exists := informer.commits["without-previous"]
	assert.False(t, exists)
}

func TestUpdateRepositories_RestoresBaselineWhenMatchFails(t *testing.T) {
	resource := testkube.CONTENT_TestTriggerResources
	trigger := testkube.TestTrigger{
		Name:      "trigger-a",
		Namespace: "testkube",
		Event:     EventGitPush,
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Branches: []string{"main"},
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

	// updateRepositories will try to contact remote, fail, and not call matcher.
	// Baseline should remain unchanged since the remote call fails before matching.
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
		Event:     EventGitPush,
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Branches: []string{"main"},
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
	keepRefPath := keepPath + refDelimiter + "ref_main"
	removeRefPath := removePath + refDelimiter + "ref_main"
	require.NoError(t, os.MkdirAll(keepPath, 0o755))
	require.NoError(t, os.MkdirAll(removePath, 0o755))
	require.NoError(t, os.MkdirAll(keepRefPath, 0o755))
	require.NoError(t, os.MkdirAll(removeRefPath, 0o755))

	informer.updateRepositories(context.Background())

	assert.Equal(t, []string{"team-b/trigger-a"}, matched)
	assert.Equal(t, "old-a", informer.commits[keepKey])
	_, keepErr := os.Stat(keepPath)
	assert.NoError(t, keepErr)
	_, removeExists := informer.commits[removeKey]
	assert.False(t, removeExists)
	_, removeErr := os.Stat(removePath)
	assert.True(t, os.IsNotExist(removeErr))
	_, keepRefErr := os.Stat(keepRefPath)
	assert.NoError(t, keepRefErr)
	_, removeRefErr := os.Stat(removeRefPath)
	assert.True(t, os.IsNotExist(removeRefErr))
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
		Event:     EventGitPush,
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:      "https://github.com/kubeshop/testkube.git",
				Branches: []string{"main"},
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

func TestUpdateRepositories_TracksMovedTagAcrossMultipleUpdates(t *testing.T) {
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
	require.NoError(t, workRepo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
	))
	worktree, err := workRepo.Worktree()
	require.NoError(t, err)

	commitFile := func(path, content, message string) plumbing.Hash {
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
		return hash
	}

	pushMainAndTag := func(tag string) {
		err = workRepo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs: []config.RefSpec{
				config.RefSpec("refs/heads/main:refs/heads/main"),
				config.RefSpec("+refs/tags/" + tag + ":refs/tags/" + tag),
			},
		})
		require.NoError(t, err)
	}

	moveTag := func(tag string, hash plumbing.Hash) {
		require.NoError(t, workRepo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewTagReferenceName(tag), hash)))
	}

	hash1 := commitFile("app.txt", "v1\n", "commit 1")
	moveTag("v1.0.0", hash1)
	pushMainAndTag("v1.0.0")

	resource := testkube.CONTENT_TestTriggerResources
	trigger := testkube.TestTrigger{
		Name:      "trigger-tag",
		Namespace: "testkube",
		Event:     EventGitTagPush,
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:  remoteDir,
				Tags: []string{"v1.0.0"},
			},
		},
	}

	var matched []string
	informer := NewInformer(
		stubTestTriggerClient{
			listFn: func(_ context.Context, _ string, _ testtriggerclient.ListOptions, _ string) ([]testkube.TestTrigger, error) {
				return []testkube.TestTrigger{trigger}, nil
			},
		},
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

	// First reconcile initializes baseline.
	informer.updateRepositories(context.Background())
	assert.Empty(t, matched)

	hash2 := commitFile("app.txt", "v2\n", "commit 2")
	moveTag("v1.0.0", hash2)
	pushMainAndTag("v1.0.0")

	informer.updateRepositories(context.Background())
	assert.Equal(t, []string{"testkube/trigger-tag"}, matched)

	hash3 := commitFile("app.txt", "v3\n", "commit 3")
	moveTag("v1.0.0", hash3)
	pushMainAndTag("v1.0.0")

	informer.updateRepositories(context.Background())
	assert.Equal(t, []string{"testkube/trigger-tag", "testkube/trigger-tag"}, matched)
}

func TestGitConfigCacheKey_GroupsByNormalizedGitConfigAndNamespace(t *testing.T) {
	keyA := gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Branches: []string{"main"},
		AuthType: string(testkube.BASIC_ContentGitAuthType),
		Paths:    []string{"pkg"},
	})
	keyB := gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Branches: []string{"main"},
		AuthType: string(testkube.BASIC_ContentGitAuthType),
		Paths:    []string{"test"},
	})
	keyWithIgnore := gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:            "https://github.com/kubeshop/testkube.git",
		Branches:       []string{"main"},
		BranchesIgnore: []string{"main"},
		AuthType:       string(testkube.BASIC_ContentGitAuthType),
	})
	keyWithTagIgnore := gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:        "https://github.com/kubeshop/testkube.git",
		Tags:       []string{"v*"},
		TagsIgnore: []string{"v1.*"},
		AuthType:   string(testkube.BASIC_ContentGitAuthType),
	})
	keyOtherNamespace := gitConfigCacheKey("team-a", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Branches: []string{"main"},
		AuthType: string(testkube.BASIC_ContentGitAuthType),
	})

	assert.Equal(t, keyA, keyB)
	assert.NotEqual(t, keyA, keyWithIgnore)
	assert.NotEqual(t, gitConfigCacheKey("testkube", &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/kubeshop/testkube.git",
		Tags:     []string{"v*"},
		AuthType: string(testkube.BASIC_ContentGitAuthType),
	}), keyWithTagIgnore)
	assert.NotEqual(t, keyA, keyOtherNamespace)
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{"exact match", "main.go", "main.go", true},
		{"no match", "main.go", "other.go", false},
		{"star matches extension", "*.go", "main.go", true},
		{"doublestar matches nested", "src/**/*.go", "src/pkg/main.go", true},
		{"doublestar matches multiple dirs", "src/**/*.go", "src/a/b/c.go", true},
		{"doublestar no match outside prefix", "src/**/*.go", "pkg/main.go", false},
		{"prefix directory match", "src", "src/main.go", true},
		{"prefix directory exact", "src", "src", true},
		{"prefix directory trailing slash", "src/", "src/main.go", true},
		{"question mark single char", "?.go", "a.go", true},
		{"question mark no match multi char", "?.go", "ab.go", false},
		{"doublestar prefix only", "src/**", "src/a/b.go", true},
		{"doublestar suffix only", "**/test.go", "a/b/test.go", true},
		{"malformed pattern returns false", "[invalid", "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchGlob(tt.pattern, tt.input))
		})
	}
}

func TestNameMatchesPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		patterns []string
		expected bool
	}{
		{"empty patterns matches all", "main", nil, true},
		{"exact match", "main", []string{"main"}, true},
		{"no match", "develop", []string{"main"}, false},
		{"glob star", "feature/foo", []string{"feature/*"}, true},
		{"glob star no match", "bugfix/foo", []string{"feature/*"}, false},
		{"multiple patterns first matches", "main", []string{"main", "develop"}, true},
		{"multiple patterns second matches", "develop", []string{"main", "develop"}, true},
		{"whitespace trimmed", "main", []string{"  main  "}, true},
		{"empty pattern skipped", "main", []string{""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nameMatchesPatterns(tt.input, tt.patterns))
		})
	}
}

func TestNameMatchesAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		patterns []string
		expected bool
	}{
		{"empty patterns returns false", "main", nil, false},
		{"exact match", "main", []string{"main"}, true},
		{"glob match", "v1.0.0", []string{"v*"}, true},
		{"no match", "develop", []string{"main", "release/*"}, false},
		{"glob question mark", "v1", []string{"v?"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nameMatchesAny(tt.input, tt.patterns))
		})
	}
}

func TestPathIsIgnored(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		file     string
		expected bool
	}{
		{"empty patterns", nil, "any/file.go", false},
		{"matches glob", []string{"*.md"}, "README.md", true},
		{"does not match", []string{"*.md"}, "main.go", false},
		{"directory pattern", []string{"docs"}, "docs/guide.md", true},
		{"multiple patterns second matches", []string{"*.txt", "*.md"}, "notes.md", true},
		{"doublestar ignore", []string{"**/*.test.js"}, "src/deep/file.test.js", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pathIsIgnored(tt.patterns, tt.file))
		})
	}
}

func TestBranchFromRef(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/foo", "feature/foo"},
		{"refs/tags/v1.0", ""},
		{"HEAD", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			assert.Equal(t, tt.expected, branchFromRef(tt.ref))
		})
	}
}

func TestTagFromRef(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"refs/tags/v1.0", "v1.0"},
		{"refs/tags/release/2.0", "release/2.0"},
		{"refs/heads/main", ""},
		{"HEAD", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			assert.Equal(t, tt.expected, tagFromRef(tt.ref))
		})
	}
}

func TestEffectiveRefsKey(t *testing.T) {
	tests := []struct {
		name     string
		config   *testkube.TestTriggerContentGit
		expected string
	}{
		{
			"branches only",
			&testkube.TestTriggerContentGit{Branches: []string{"main", "develop"}},
			"b:main,b:develop",
		},
		{
			"tags only",
			&testkube.TestTriggerContentGit{Tags: []string{"v1.0", "v2.0"}},
			"t:v1.0,t:v2.0",
		},
		{
			"branches and tags",
			&testkube.TestTriggerContentGit{Branches: []string{"main"}, Tags: []string{"v1.0"}},
			"b:main,t:v1.0",
		},
		{
			"empty",
			&testkube.TestTriggerContentGit{},
			"",
		},
		{
			"whitespace trimmed",
			&testkube.TestTriggerContentGit{Branches: []string{" main "}},
			"b:main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, effectiveRefsKey(tt.config))
		})
	}
}

func TestRefSubKey(t *testing.T) {
	tests := []struct {
		name       string
		triggerKey string
		ref        string
		expected   string
	}{
		{"with ref", "v1:ns/trigger", "refs/heads/main", "v1:ns/trigger|ref|refs/heads/main"},
		{"empty ref returns trigger key", "v1:ns/trigger", "", "v1:ns/trigger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, refSubKey(tt.triggerKey, tt.ref))
		})
	}
}

func TestRefDirectorySuffix_AvoidsSanitizerCollisions(t *testing.T) {
	refA := "refs/heads/a-b"
	refB := "refs/heads/a/b"

	assert.Equal(t, envVarNameSanitizer.ReplaceAllString(refA, "_"), envVarNameSanitizer.ReplaceAllString(refB, "_"),
		"sanitizer output should collide for this regression test setup")
	assert.NotEqual(t, refDirectorySuffix(refA), refDirectorySuffix(refB))

	key := triggerKey(testTriggerSource, "ns", "trigger")
	repoA := triggerRepositoryPathFromKey(key) + refDelimiter + refDirectorySuffix(refA)
	repoB := triggerRepositoryPathFromKey(key) + refDelimiter + refDirectorySuffix(refB)
	assert.NotEqual(t, repoA, repoB)
}

func TestEffectiveRefs(t *testing.T) {
	tests := []struct {
		name     string
		config   *testkube.TestTriggerContentGit
		expected []string
	}{
		{
			"exact branches",
			&testkube.TestTriggerContentGit{Branches: []string{"main", "develop"}},
			[]string{"refs/heads/main", "refs/heads/develop"},
		},
		{
			"exact tags",
			&testkube.TestTriggerContentGit{Tags: []string{"v1.0", "v2.0"}},
			[]string{"refs/tags/v1.0", "refs/tags/v2.0"},
		},
		{
			"glob branches skipped",
			&testkube.TestTriggerContentGit{Branches: []string{"feature/*"}},
			nil,
		},
		{
			"glob tags skipped",
			&testkube.TestTriggerContentGit{Tags: []string{"v*"}},
			nil,
		},
		{
			"mixed exact and glob",
			&testkube.TestTriggerContentGit{Branches: []string{"main", "release/*"}},
			[]string{"refs/heads/main"},
		},
		{
			"empty input",
			&testkube.TestTriggerContentGit{},
			nil,
		},
		{
			"whitespace only entries skipped",
			&testkube.TestTriggerContentGit{Branches: []string{"  ", "main"}},
			[]string{"refs/heads/main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := effectiveRefs(tt.config)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPathMatchesNormalized_GlobPatterns(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		file     string
		expected bool
	}{
		{"glob star extension", []string{"*.md"}, "README.md", true},
		{"glob star no match different ext", []string{"*.md"}, "main.go", false},
		{"doublestar recursive", []string{"src/**/*.go"}, "src/pkg/file.go", true},
		{"directory prefix", []string{"pkg"}, "pkg/util.go", true},
		{"exact file", []string{"Makefile"}, "Makefile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizePaths(tt.paths)
			assert.Equal(t, tt.expected, pathMatchesNormalized(normalized, tt.file))
		})
	}
}

func TestTriggerKeyFromRefSubKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with ref separator", "v1:ns/trigger|ref|refs/heads/main", "v1:ns/trigger"},
		{"without ref separator", "v1:ns/trigger", "v1:ns/trigger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, triggerKeyFromRefSubKey(tt.input))
		})
	}
}

func TestCollectHeadMetadata_UsesPreferredRefForGlobBranches(t *testing.T) {
	informer := &Informer{}
	gitConfig := &testkube.TestTriggerContentGit{
		Branches: []string{"release/*"},
	}

	meta := informer.collectHeadMetadata(nil, "abc123", gitConfig, "refs/heads/release/v1")

	assert.Equal(t, "abc123", meta[GitMetaKeyCommit])
	assert.Equal(t, "refs/heads/release/v1", meta[GitMetaKeyRef])
	assert.Equal(t, "release/v1", meta[GitMetaKeyBranch])
	assert.Empty(t, meta[GitMetaKeyTag])
}

func TestCollectHeadMetadata_BranchesOnlyDoesNotSetTag(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)

	require.NoError(t, repo.Storer.SetReference(
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
	))

	filePath := filepath.Join(tmpDir, "README.md")
	require.NoError(t, os.WriteFile(filePath, []byte("test\n"), 0o644))

	worktree, err := repo.Worktree()
	require.NoError(t, err)
	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	hash, err := worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	require.NoError(t, repo.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewTagReferenceName("v1.0.0"), hash),
	))

	informer := &Informer{}
	gitConfig := &testkube.TestTriggerContentGit{
		Branches: []string{"main"},
	}

	meta := informer.collectHeadMetadata(repo, hash.String(), gitConfig, "")

	assert.Equal(t, hash.String(), meta[GitMetaKeyCommit])
	assert.Equal(t, "refs/heads/main", meta[GitMetaKeyRef])
	assert.Equal(t, "main", meta[GitMetaKeyBranch])
	assert.Empty(t, meta[GitMetaKeyTag])
}
