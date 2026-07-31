package informer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

const gitlabTestURI = "https://gitlab.com/group/sub/project.git"

// gitlabTestProjectPath is the percent-encoded form the provider must put on the
// wire for the namespaced project in gitlabTestURI.
const gitlabTestProjectPath = "group%2Fsub%2Fproject"

func TestGitLabAPIBaseFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{"gitlab.com", "https://gitlab.com/group/project.git", "https://gitlab.com/api/v4"},
		{"self managed", "https://gitlab.example.com/group/project", "https://gitlab.example.com/api/v4"},
		{"https port is preserved", "https://gitlab.example.com:8443/group/project.git", "https://gitlab.example.com:8443/api/v4"},
		// An SSH port must never be reused for HTTPS.
		{"ssh port is dropped", "ssh://git@gitlab.example.com:2222/group/project.git", "https://gitlab.example.com/api/v4"},
		{"scp syntax", "git@gitlab.example.com:group/project.git", "https://gitlab.example.com/api/v4"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, gitlabAPIBaseFromURI(tt.uri))
		})
	}
}

func TestGitLabEscapeProjectPath(t *testing.T) {
	assert.Equal(t, "group%2Fproject", gitlabEscapeProjectPath("group/project"))
	assert.Equal(t, "group%2Fsub%2Fproject", gitlabEscapeProjectPath("group/sub/project"))
	assert.Equal(t, "group%2Fsub%2Fsubsub%2Fproject", gitlabEscapeProjectPath("group/sub/subsub/project"))
	assert.Equal(t, "my-group%2Fmy.project", gitlabEscapeProjectPath("my-group/my.project"))
	assert.Equal(t, "group%2Fproject", gitlabEscapeProjectPath("/group/project/"))
}

func TestNormalizeGitLabState(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"opened", prStateOpen},
		// locked is a transient merge-in-progress state; mapping it to open keeps
		// opened -> locked -> merged from firing a spurious extra event.
		{"locked", prStateOpen},
		{"closed", prStateClosed},
		{"merged", prStateClosed},
		{"OPENED", prStateOpen},
		{" merged ", prStateClosed},
		// An unrecognized future state must not be reported as closed.
		{"something-new", prStateOpen},
		{"", prStateOpen},
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeGitLabState(tt.state))
		})
	}
}

func TestGitLabProviderHeadRef(t *testing.T) {
	assert.Equal(t, "refs/merge-requests/7/head", newGitLabProvider("", "group/project", "").headRef(7))
}

func TestGitLabProviderList_MockServer(t *testing.T) {
	// Raw JSON rather than a marshalled Go struct, so a field-name typo in the
	// gitlabMR tags is caught rather than papered over.
	const body = `[
	  {
	    "id": 9001,
	    "iid": 12,
	    "state": "opened",
	    "title": "Add feature",
	    "updated_at": "2026-07-30T10:11:12.000Z",
	    "web_url": "https://gitlab.com/group/sub/project/-/merge_requests/12",
	    "source_branch": "feature/x",
	    "target_branch": "main",
	    "sha": "abc123",
	    "draft": true,
	    "author": {"id": 5, "username": "dev", "name": "Dev Eloper"}
	  },
	  {
	    "id": 9002,
	    "iid": 11,
	    "state": "merged",
	    "title": "Old work",
	    "updated_at": "2026-07-29T10:11:12.000Z",
	    "web_url": "https://gitlab.com/group/sub/project/-/merge_requests/11",
	    "source_branch": "feature/y",
	    "target_branch": "develop",
	    "sha": "def456",
	    "draft": false,
	    "author": {"id": 6, "username": "other"}
	  }
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// EscapedPath, not Path: Path is percent-decoded and would hide a
		// regression where the %2F never reached the wire.
		require.Equal(t, "/projects/"+gitlabTestProjectPath+"/merge_requests", r.URL.EscapedPath())
		require.Equal(t, "Bearer glpat-token", r.Header.Get("Authorization"))

		query := r.URL.Query()
		require.Equal(t, "all", query.Get("state"))
		require.Equal(t, "updated_at", query.Get("order_by"))
		require.Equal(t, "desc", query.Get("sort"))
		require.Equal(t, "all", query.Get("scope"))
		require.Equal(t, fmt.Sprint(gitlabMRListPageSize), query.Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	provider := newGitLabProvider(server.URL, "group/sub/project", "glpat-token")
	result, err := provider.list(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 2)

	// iid, not id, is the number users see and the API accepts.
	assert.Equal(t, 12, result[0].Number)
	assert.Equal(t, prStateOpen, result[0].State)
	assert.Equal(t, "Add feature", result[0].Title)
	assert.Equal(t, "feature/x", result[0].HeadRef)
	assert.Equal(t, "abc123", result[0].HeadSHA)
	assert.Equal(t, "main", result[0].BaseRef)
	assert.Equal(t, "dev", result[0].Author)
	assert.Equal(t, "https://gitlab.com/group/sub/project/-/merge_requests/12", result[0].URL)
	assert.True(t, result[0].Draft)
	assert.Equal(t, 2026, result[0].UpdatedAt.Year())

	assert.Equal(t, 11, result[1].Number)
	assert.Equal(t, prStateClosed, result[1].State, "merged must normalize to closed")
}

func TestGitLabProviderList_NullAuthorAndNullSHA(t *testing.T) {
	// GitLab returns a null author for merge requests whose user was deleted, and
	// a null sha for the first seconds after a merge request is created.
	const body = `[{"iid": 3, "state": "opened", "title": "New", "sha": null, "author": null,
	  "source_branch": "f", "target_branch": "main", "web_url": "https://example.com/3"}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	result, err := newGitLabProvider(server.URL, "group/project", "").list(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Empty(t, result[0].HeadSHA)
	assert.Empty(t, result[0].Author)
}

func TestGitLabProviderList_404MentionsProjectAndScope(t *testing.T) {
	// GitLab deliberately answers 404 rather than 403 for a project the token
	// cannot see, so the error has to explain both possibilities.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"404 Project Not Found"}`))
	}))
	defer server.Close()

	_, err := newGitLabProvider(server.URL, "group/sub/project", "glpat-token").list(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group/sub/project")
	assert.Contains(t, err.Error(), "read_api")
}

// gitlabDiffPage renders a page of diff entries with distinct paths.
func gitlabDiffPage(t *testing.T, prefix string, count int) string {
	t.Helper()
	diffs := make([]gitlabMRDiff, 0, count)
	for i := 0; i < count; i++ {
		diffs = append(diffs, gitlabMRDiff{OldPath: fmt.Sprintf("%s/f%d.go", prefix, i), NewPath: fmt.Sprintf("%s/f%d.go", prefix, i)})
	}
	encoded, err := json.Marshal(diffs)
	require.NoError(t, err)
	return string(encoded)
}

func TestGitLabProviderChangedFiles_Paginates(t *testing.T) {
	var requestedPages []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/projects/"+gitlabTestProjectPath+"/merge_requests/12/diffs", r.URL.EscapedPath())
		// The diffs endpoint 500s above 30, so the page size must stay capped.
		require.Equal(t, "30", r.URL.Query().Get("per_page"))

		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			_, _ = w.Write([]byte(gitlabDiffPage(t, "a", gitlabMRDiffPageSize)))
		case "2":
			_, _ = w.Write([]byte(gitlabDiffPage(t, "b", 5)))
		default:
			t.Errorf("unexpected page requested: %s", page)
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	result, truncated, err := newGitLabProvider(server.URL, "group/sub/project", "").changedFiles(context.Background(), 12)
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, requestedPages, "a short page must end pagination")
	assert.False(t, truncated, "a short final page means the list is complete")
	assert.Len(t, result, gitlabMRDiffPageSize+5)
	assert.Equal(t, "a/f0.go", result[0])
	assert.Equal(t, "b/f4.go", result[len(result)-1])
}

func TestGitLabProviderChangedFiles_TruncationWarns(t *testing.T) {
	core, recordedLogs := observer.New(zap.WarnLevel)
	originalLogger := log.DefaultLogger
	log.DefaultLogger = zap.New(core).Sugar()
	t.Cleanup(func() { log.DefaultLogger = originalLogger })

	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		// Always a full page, so pagination only stops at the page cap.
		_, _ = w.Write([]byte(gitlabDiffPage(t, "p"+r.URL.Query().Get("page"), gitlabMRDiffPageSize)))
	}))
	defer server.Close()

	result, truncated, err := newGitLabProvider(server.URL, "group/project", "").changedFiles(context.Background(), 12)
	require.NoError(t, err)
	assert.Equal(t, int32(gitlabMRDiffMaxPages), atomic.LoadInt32(&requests))
	assert.Len(t, result, gitlabMRDiffMaxPages*gitlabMRDiffPageSize)
	assert.True(t, truncated, "hitting the page cap must be reported to the caller")

	// Silent truncation would read as "no matching paths"; it must be logged.
	require.NotEmpty(t, recordedLogs.All())
	assert.Contains(t, recordedLogs.All()[0].Message, "path filtering may be incomplete")
}

func TestGitLabProviderChangedFiles_RenamesAndDeletions(t *testing.T) {
	const body = `[
	  {"old_path": "src/a.go", "new_path": "src/a.go", "new_file": false, "renamed_file": false, "deleted_file": false},
	  {"old_path": "old/b.go", "new_path": "src/b.go", "new_file": false, "renamed_file": true, "deleted_file": false},
	  {"old_path": "src/c.go", "new_path": "src/c.go", "new_file": false, "renamed_file": false, "deleted_file": true},
	  {"old_path": "src/d.go", "new_path": "src/d.go", "new_file": true, "renamed_file": false, "deleted_file": false}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	result, truncated, err := newGitLabProvider(server.URL, "group/project", "").changedFiles(context.Background(), 12)
	require.NoError(t, err)
	assert.False(t, truncated)
	// A rename contributes both paths; a deletion repeats one path and must not
	// be duplicated.
	assert.Equal(t, []string{"src/a.go", "src/b.go", "old/b.go", "src/c.go", "src/d.go"}, result)
}

// gitlabMRJSON renders a single merge request as the GitLab list endpoint would.
func gitlabMRJSON(iid int, state, sha, sourceBranch, targetBranch string) string {
	return fmt.Sprintf(`{"iid": %d, "state": %q, "title": "MR %d", "sha": %q,
	  "source_branch": %q, "target_branch": %q,
	  "web_url": "https://gitlab.com/group/sub/project/-/merge_requests/%d",
	  "author": {"username": "dev"}, "draft": false}`,
		iid, state, iid, sha, sourceBranch, targetBranch, iid)
}

// TestCheckMergeRequests_E2E drives checkPullRequests end to end against a mock
// GitLab API, mirroring TestCheckPullRequests_E2E.
func TestCheckMergeRequests_E2E(t *testing.T) {
	var currentMRs string
	var currentDiffs string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.EscapedPath(), "/diffs"):
			if currentDiffs == "" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"500 Internal Server Error"}`))
				return
			}
			_, _ = w.Write([]byte(currentDiffs))
		case strings.HasSuffix(r.URL.EscapedPath(), "/merge_requests"):
			_, _ = w.Write([]byte(currentMRs))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiBaseFunc := func(_ string) string { return server.URL }
	const key = "v1:default/test-trigger"

	// newInformer builds an informer with the given pre-existing baselines. The
	// gitlab.com host means provider detection needs no probe.
	newInformer := func(commits map[string]string) *Informer {
		if commits == nil {
			commits = make(map[string]string)
		}
		return &Informer{commits: commits, prAPIBaseFunc: apiBaseFunc}
	}

	t.Run("first_reconcile_stores_baseline_without_firing", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-initial", "feature/x", "main") + "]"

		inf := newInformer(nil)
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed, "first reconcile must not fire")

		assert.Equal(t, "sha-initial:open", inf.commits[prCacheKey(key, providerGitLab, 1)])
		assert.Equal(t, "1", inf.commits[prInitKey(key, providerGitLab)])
	})

	t.Run("new_mr_after_init_fires_opened_with_gitlab_ref", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(2, gitlabStateOpened, "sha-new", "feature/y", "main") + "]"

		inf := newInformer(map[string]string{prInitKey(key, providerGitLab): "1"})
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		require.True(t, result.changed)

		assert.Equal(t, "opened", result.metadata[GitMetaKeyPRAction])
		// The iid, not the global id, is what users and the API see.
		assert.Equal(t, "2", result.metadata[GitMetaKeyPRNumber])
		assert.Equal(t, "refs/merge-requests/2/head", result.metadata[GitMetaKeyRef])
		assert.Equal(t, "sha-new", result.metadata[GitMetaKeyCommit])
		assert.Equal(t, "main", result.metadata[GitMetaKeyPRBaseRef])
		assert.Equal(t, "feature/y", result.metadata[GitMetaKeyPRHeadRef])
		assert.Equal(t, "dev", result.metadata[GitMetaKeyPRAuthor])
	})

	t.Run("new_commit_fires_synchronize", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-v2", "feature/x", "main") + "]"

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		require.True(t, result.changed)
		assert.Equal(t, "synchronize", result.metadata[GitMetaKeyPRAction])
	})

	t.Run("locked_state_does_not_fire", func(t *testing.T) {
		// locked is a transient merge-in-progress state, not a user-visible event.
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateLocked, "sha-initial", "feature/x", "main") + "]"

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed, "locked must normalize to open and fire nothing")
		assert.Equal(t, "sha-initial:open", inf.commits[prCacheKey(key, providerGitLab, 1)])
	})

	t.Run("merged_fires_closed", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateMerged, "sha-initial", "feature/x", "main") + "]"

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		require.True(t, result.changed)
		assert.Equal(t, "closed", result.metadata[GitMetaKeyPRAction])
	})

	t.Run("reopened_after_close_fires_reopened", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-initial", "feature/x", "main") + "]"

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:closed",
		})
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		require.True(t, result.changed)
		assert.Equal(t, "reopened", result.metadata[GitMetaKeyPRAction])
	})

	t.Run("null_sha_mr_is_skipped_without_baseline", func(t *testing.T) {
		// GitLab populates sha asynchronously; firing now would hand the workflow
		// an empty commit.
		currentMRs = `[{"iid": 5, "state": "opened", "title": "Fresh", "sha": null,
		  "source_branch": "f", "target_branch": "main", "web_url": "https://example.com/5",
		  "author": {"username": "dev"}}]`

		inf := newInformer(map[string]string{prInitKey(key, providerGitLab): "1"})
		result, err := inf.checkPullRequests(context.Background(), key, buildPRTrigger(gitlabTestURI, nil), newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed)
		assert.NotContains(t, inf.commits, prCacheKey(key, providerGitLab, 5),
			"no baseline may be recorded, so the next pass reports a real opened event")
	})

	t.Run("target_branch_filter_applies", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(6, gitlabStateOpened, "sha-6", "feature/z", "develop") + "]"

		inf := newInformer(map[string]string{prInitKey(key, providerGitLab): "1"})
		trigger := buildPRTrigger(gitlabTestURI, &testkube.TestTriggerContentGitPullRequest{Branches: []string{"main"}})
		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed, "an MR targeting develop must not match branches: [main]")
	})

	t.Run("type_filter_rejection_advances_baseline", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateMerged, "sha-initial", "feature/x", "main") + "]"

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		trigger := buildPRTrigger(gitlabTestURI, &testkube.TestTriggerContentGitPullRequest{Types: []string{"opened"}})
		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed)
		assert.Equal(t, "sha-initial:closed", inf.commits[prCacheKey(key, providerGitLab, 1)],
			"baseline must advance so the same state is not re-evaluated")
	})

	t.Run("path_filter_matches_new_path", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-v3", "feature/x", "main") + "]"
		currentDiffs = `[{"old_path": "src/main.go", "new_path": "src/main.go"}]`
		t.Cleanup(func() { currentDiffs = "" })

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		trigger := buildPRTrigger(gitlabTestURI, nil, "src/**")
		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		require.True(t, result.changed)
		assert.Equal(t, "synchronize", result.metadata[GitMetaKeyPRAction])
	})

	t.Run("path_filter_mismatch_advances_baseline", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-v4", "feature/x", "main") + "]"
		currentDiffs = `[{"old_path": "docs/readme.md", "new_path": "docs/readme.md"}]`
		t.Cleanup(func() { currentDiffs = "" })

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		trigger := buildPRTrigger(gitlabTestURI, nil, "src/**")
		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed)
		assert.Equal(t, "sha-v4:open", inf.commits[prCacheKey(key, providerGitLab, 1)])
	})

	// A truncated changed-file list cannot prove the absence of a match, so
	// advancing the baseline would drop the event permanently. Fire instead.
	t.Run("truncated_diffs_with_no_match_fires_instead_of_skipping", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-v6", "feature/x", "main") + "]"
		// Every page full, so pagination stops at the cap with no matching path in
		// the part that was fetched.
		currentDiffs = gitlabDiffPage(t, "docs", gitlabMRDiffPageSize)
		t.Cleanup(func() { currentDiffs = "" })

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		trigger := buildPRTrigger(gitlabTestURI, nil, "src/**")
		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		require.True(t, result.changed,
			"a match may sit in the diff pages that were never fetched, so the event must not be dropped")
		assert.Equal(t, "synchronize", result.metadata[GitMetaKeyPRAction])
	})

	t.Run("transient_diffs_error_does_not_advance_baseline", func(t *testing.T) {
		currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-v5", "feature/x", "main") + "]"
		// currentDiffs stays empty, so the mock answers 500.
		currentDiffs = ""

		inf := newInformer(map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		})
		trigger := buildPRTrigger(gitlabTestURI, nil, "src/**")
		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed)
		assert.Equal(t, "sha-initial:open", inf.commits[prCacheKey(key, providerGitLab, 1)],
			"baseline must not advance so the event is retried next pass")
	})
}

func TestResolvePRToken_GitLabIgnoresGitHubAuthType(t *testing.T) {
	core, recordedLogs := observer.New(zap.WarnLevel)
	originalLogger := log.DefaultLogger
	log.DefaultLogger = zap.New(core).Sugar()
	t.Cleanup(func() { log.DefaultLogger = originalLogger })

	provider := &stubGitHubTokenProvider{token: "github-app-token"}
	inf := &Informer{githubTokenProvider: provider}

	gitConfig := &testkube.TestTriggerContentGit{
		Uri:      gitlabTestURI,
		AuthType: string(testkube.GITHUB_ContentGitAuthType),
		Token:    "glpat-configured",
	}

	token := inf.resolvePRToken(context.Background(), "default", gitConfig, providerGitLab, newReconcileCache())

	// The control plane only mints GitHub App tokens, so it must not be called
	// with a GitLab URL.
	assert.Equal(t, "glpat-configured", token)
	assert.Zero(t, provider.calls, "GitHub token provider must not be consulted for GitLab")
	require.NotEmpty(t, recordedLogs.All())
	assert.Equal(t, prGitHubAuthTypeWrongProviderWarning, recordedLogs.All()[0].Message)
}

// TestCheckMergeRequests_ListWindowEvictionIsRecoverable documents the bound on the
// single-page merge request list: an MR pushed out of the "most recently updated"
// window by newer activity is DEFERRED, not lost. Matching is state-based, so the
// unchanged baseline still differs once the MR reappears and the event fires then.
func TestCheckMergeRequests_ListWindowEvictionIsRecoverable(t *testing.T) {
	var currentMRs string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(currentMRs))
	}))
	defer server.Close()

	const key = "v1:default/test-trigger"
	inf := &Informer{
		commits: map[string]string{
			prInitKey(key, providerGitLab):     "1",
			prCacheKey(key, providerGitLab, 1): "sha-initial:open",
		},
		prAPIBaseFunc: func(_ string) string { return server.URL },
	}
	trigger := buildPRTrigger(gitlabTestURI, nil)

	// Pass 1: MR 1 got a new commit but is absent from the page, crowded out by
	// other recently updated merge requests.
	currentMRs = "[" + gitlabMRJSON(99, gitlabStateOpened, "sha-other", "noise", "main") + "]"
	result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
	require.NoError(t, err)
	require.True(t, result.changed, "the crowding MR itself fires")
	assert.Equal(t, "99", result.metadata[GitMetaKeyPRNumber])

	// MR 1's baseline is untouched, so nothing about it has been consumed.
	assert.Equal(t, "sha-initial:open", inf.commits[prCacheKey(key, providerGitLab, 1)])

	// Pass 2: MR 1 is visible again. Its new commit is still detected.
	currentMRs = "[" + gitlabMRJSON(1, gitlabStateOpened, "sha-v2", "feature/x", "main") + "]"
	result, err = inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
	require.NoError(t, err)
	require.True(t, result.changed, "a deferred merge request must still fire once visible")
	assert.Equal(t, "1", result.metadata[GitMetaKeyPRNumber])
	assert.Equal(t, "synchronize", result.metadata[GitMetaKeyPRAction])
	assert.Equal(t, "sha-v2", result.metadata[GitMetaKeyCommit])
}
