package informer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestSplitGitURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		host     string
		repoPath string
		ok       bool
	}{
		{"https", "https://gitlab.com/group/project.git", "gitlab.com", "group/project", true},
		{"https no .git", "https://gitlab.com/group/project", "gitlab.com", "group/project", true},
		{"nested subgroups", "https://gitlab.com/group/sub/subsub/project.git", "gitlab.com", "group/sub/subsub/project", true},
		{"scp syntax", "git@gitlab.com:group/sub/project.git", "gitlab.com", "group/sub/project", true},
		{"ssh url with port", "ssh://git@gitlab.example.com:2222/group/project.git", "gitlab.example.com", "group/project", true},
		{"userinfo is dropped", "https://oauth2:token@gitlab.com/group/project.git", "gitlab.com", "group/project", true},
		{"uppercase host is normalized", "https://GitLab.COM/group/project", "gitlab.com", "group/project", true},
		{"trailing slash", "https://gitlab.com/group/project/", "gitlab.com", "group/project", true},
		// A URL copied out of the GitLab web UI still resolves to the project.
		{"web ui suffix is stripped", "https://gitlab.com/group/project/-/merge_requests/7", "gitlab.com", "group/project", true},
		{"github", "https://github.com/kubeshop/testkube.git", "github.com", "kubeshop/testkube", true},
		{"single segment path", "https://gitlab.com/project", "gitlab.com", "project", true},
		{"no path", "https://gitlab.com", "", "", false},
		{"empty", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, repoPath, ok := splitGitURI(tt.uri)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.host, host)
				assert.Equal(t, tt.repoPath, repoPath)
			}
		})
	}
}

func TestGitAPIHostPort(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		host      string
		authority string
	}{
		{"https", "https://gitlab.com/g/p", "gitlab.com", "gitlab.com"},
		// An https port belongs to the API endpoint too.
		{"https with port", "https://gitlab.example.com:8443/g/p", "gitlab.example.com", "gitlab.example.com:8443"},
		// An SSH port must never be reused for HTTPS.
		{"ssh with port", "ssh://git@gitlab.example.com:2222/g/p.git", "gitlab.example.com", "gitlab.example.com"},
		{"scp syntax", "git@ghe.example.com:o/r.git", "ghe.example.com", "ghe.example.com"},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, authority := gitAPIHostPort(tt.uri)
			assert.Equal(t, tt.host, host)
			assert.Equal(t, tt.authority, authority)
		})
	}
}

func TestProviderKindFromHost(t *testing.T) {
	tests := []struct {
		host     string
		expected providerKind
		ok       bool
	}{
		{"github.com", providerGitHub, true},
		{"github.example.com", providerGitHub, true},
		{"gitlab.com", providerGitLab, true},
		{"gitlab.example.com", providerGitLab, true},
		// GitLab wins so a host carrying both names is not misread as GitHub.
		{"gitlab.github-mirror.example.com", providerGitLab, true},
		{"git.example.com", providerUnknown, false},
		{"", providerUnknown, false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			kind, ok := providerKindFromHost(tt.host)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, kind)
		})
	}
}

// TestPRCacheKeyProviderScoping locks in that GitHub keeps its historical
// unprefixed keys, so baselines survive an upgrade, while other providers are
// namespaced so repointing a trigger's uri re-baselines instead of firing a burst
// of events against numbering that means something else.
func TestPRCacheKeyProviderScoping(t *testing.T) {
	const triggerKey = "v1:default/my-trigger"

	assert.Equal(t, triggerKey+refSeparator+"pr:42", prCacheKey(triggerKey, providerGitHub, 42))
	assert.Equal(t, triggerKey+refSeparator+"pr:__init__", prInitKey(triggerKey, providerGitHub))

	assert.Equal(t, triggerKey+refSeparator+"pr:gitlab:42", prCacheKey(triggerKey, providerGitLab, 42))
	assert.Equal(t, triggerKey+refSeparator+"pr:gitlab:__init__", prInitKey(triggerKey, providerGitLab))

	assert.NotEqual(t, prCacheKey(triggerKey, providerGitHub, 42), prCacheKey(triggerKey, providerGitLab, 42))

	// Both shapes must still round-trip through the snapshot/cleanup helpers.
	for _, key := range []string{
		prCacheKey(triggerKey, providerGitHub, 42),
		prCacheKey(triggerKey, providerGitLab, 42),
		prInitKey(triggerKey, providerGitLab),
	} {
		assert.Equal(t, triggerKey, triggerKeyFromRefSubKey(key), key)
	}

	inf := &Informer{commits: map[string]string{
		prCacheKey(triggerKey, providerGitHub, 42): "sha1:open",
		prCacheKey(triggerKey, providerGitLab, 42): "sha2:open",
		prInitKey(triggerKey, providerGitLab):      "1",
	}}
	assert.Len(t, inf.snapshotRefCommits(triggerKey), 3)
}

func TestPRProviderFor_Detection(t *testing.T) {
	var requests int32
	// Any request means detection escaped to the network when it should not have.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		uri      string
		authType string
		expected providerKind
	}{
		{"github.com", "https://github.com/owner/repo.git", "", providerGitHub},
		{"ghes by host name", "https://github.example.com/owner/repo.git", "", providerGitHub},
		{"gitlab.com", "https://gitlab.com/group/project.git", "", providerGitLab},
		{"self managed gitlab by host name", "https://gitlab.example.com/group/sub/project.git", "", providerGitLab},
		// authType github can only mean GitHub, so no probe is needed even for a
		// GitHub Enterprise Server host that does not carry "github" in its name.
		{"unnamed ghes host with github authType", "https://git.example.com/owner/repo.git",
			string(testkube.GITHUB_ContentGitAuthType), providerGitHub},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atomic.StoreInt32(&requests, 0)
			inf := &Informer{prAPIBaseFunc: func(_ string) string { return server.URL }}

			gitConfig := &testkube.TestTriggerContentGit{Uri: tt.uri, AuthType: tt.authType}
			provider, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, provider.kind())
			assert.Zero(t, atomic.LoadInt32(&requests), "provider must be resolved without a probe")
		})
	}

	t.Run("gitlab requires a namespaced project path", func(t *testing.T) {
		inf := &Informer{prAPIBaseFunc: func(_ string) string { return server.URL }}
		gitConfig := &testkube.TestTriggerContentGit{Uri: "https://gitlab.com/project.git"}
		_, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespaced GitLab project path")
	})

	t.Run("uri without a path is rejected", func(t *testing.T) {
		inf := &Informer{}
		gitConfig := &testkube.TestTriggerContentGit{Uri: "not a uri"}
		_, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host and project path")
	})
}

func TestPRProviderFor_ProbesUnknownHost(t *testing.T) {
	newProbeServer := func(status int, counter *int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(counter, 1)
			if r.URL.Path == "/version" {
				w.WriteHeader(status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		}))
	}

	// A 401/403 still proves the /api/v4 route exists, so the probe works whether
	// or not the configured token is valid.
	for _, tt := range []struct {
		name     string
		status   int
		expected providerKind
	}{
		{"200 is gitlab", http.StatusOK, providerGitLab},
		{"401 is gitlab", http.StatusUnauthorized, providerGitLab},
		{"403 is gitlab", http.StatusForbidden, providerGitLab},
		{"404 falls back to github", http.StatusNotFound, providerGitHub},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests int32
			server := newProbeServer(tt.status, &requests)
			defer server.Close()

			inf := &Informer{prAPIBaseFunc: func(_ string) string { return server.URL }}
			// A neutral host carrying neither provider name, so only the probe can
			// decide. Two path segments so either provider can accept it.
			gitConfig := &testkube.TestTriggerContentGit{Uri: "https://git.example.com/group/project.git"}

			provider, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, provider.kind())
			assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "exactly one probe")

			// A second resolution must reuse the cached kind.
			provider, err = inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, provider.kind())
			assert.Equal(t, int32(1), atomic.LoadInt32(&requests), "the probe result must be cached per host")
		})
	}

	t.Run("unexpected status is not cached", func(t *testing.T) {
		var requests int32
		server := newProbeServer(http.StatusBadGateway, &requests)
		defer server.Close()

		inf := &Informer{prAPIBaseFunc: func(_ string) string { return server.URL }}
		gitConfig := &testkube.TestTriggerContentGit{Uri: "https://git.example.com/group/project.git"}

		_, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
		require.Error(t, err)
		assert.Empty(t, inf.prProviders, "a transient failure must not pin the host to a provider")

		// The next pass retries rather than reusing an inconclusive result.
		_, err = inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
		require.Error(t, err)
		assert.Equal(t, int32(2), atomic.LoadInt32(&requests))
	})

	// TestPRProviderFor_ProbeSendsNoCredential is a security regression guard. The
	// probe host comes straight from the trigger's uri and has not yet been shown
	// to be a supported provider, so it must never receive the trigger's token: an
	// attacker who can create a TestTrigger could otherwise name their own host and
	// have the informer hand over any Secret in the namespace.
	t.Run("probe sends no credential", func(t *testing.T) {
		var probes int32
		var sawAuthHeader bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/version" {
				atomic.AddInt32(&probes, 1)
				if r.Header.Get("Authorization") != "" || r.Header.Get("Private-Token") != "" {
					sawAuthHeader = true
				}
				// 401 is enough to prove the /api/v4 route exists.
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		}))
		defer server.Close()

		inf := &Informer{prAPIBaseFunc: func(_ string) string { return server.URL }}
		gitConfig := &testkube.TestTriggerContentGit{
			Uri:   "https://attacker.example.com/group/project.git",
			Token: "super-secret-token",
		}

		provider, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
		require.NoError(t, err)
		require.Equal(t, providerGitLab, provider.kind())
		assert.Equal(t, int32(1), atomic.LoadInt32(&probes))
		assert.False(t, sawAuthHeader, "the detection probe must not carry the trigger credential")
	})

	t.Run("unreachable host is not cached", func(t *testing.T) {
		server := newProbeServer(http.StatusOK, new(int32))
		unreachable := server.URL
		server.Close()

		inf := &Informer{prAPIBaseFunc: func(_ string) string { return unreachable }}
		gitConfig := &testkube.TestTriggerContentGit{Uri: "https://git.example.com/group/project.git"}

		_, err := inf.prProviderFor(context.Background(), "default", gitConfig, newReconcileCache())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detect git provider")
		assert.Empty(t, inf.prProviders)
	})
}
