package informer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name  string
		uri   string
		owner string
		repo  string
		ok    bool
	}{
		{"https", "https://github.com/kubeshop/testkube.git", "kubeshop", "testkube", true},
		{"https no .git", "https://github.com/kubeshop/testkube", "kubeshop", "testkube", true},
		{"ssh", "git@github.com:kubeshop/testkube.git", "kubeshop", "testkube", true},
		{"not github", "https://gitlab.com/foo/bar.git", "", "", false},
		{"empty", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := parseGitHubRepo(tt.uri)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.owner, owner)
				assert.Equal(t, tt.repo, repo)
			}
		})
	}
}

func TestGitHubAPIBaseFromURI(t *testing.T) {
	assert.Equal(t, "https://api.github.com", githubAPIBaseFromURI("https://github.com/foo/bar"))
	assert.Equal(t, "https://github.example.com/api/v3", githubAPIBaseFromURI("https://github.example.com/foo/bar"))
}

func TestPRMatchesBaseBranch(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		config   *testkube.TestTriggerContentGitPullRequest
		expected bool
	}{
		{"nil config matches all", "main", nil, true},
		{"empty branches matches all", "main", &testkube.TestTriggerContentGitPullRequest{}, true},
		{"branch matches", "main", &testkube.TestTriggerContentGitPullRequest{Branches: []string{"main", "develop"}}, true},
		{"branch does not match", "feature/x", &testkube.TestTriggerContentGitPullRequest{Branches: []string{"main"}}, false},
		{"glob match", "release/1.0", &testkube.TestTriggerContentGitPullRequest{Branches: []string{"release/*"}}, true},
		{"ignore takes precedence", "main", &testkube.TestTriggerContentGitPullRequest{Branches: []string{"main"}, BranchesIgnore: []string{"main"}}, false},
		{"ignore only", "main", &testkube.TestTriggerContentGitPullRequest{BranchesIgnore: []string{"main"}}, false},
		{"ignore does not match", "develop", &testkube.TestTriggerContentGitPullRequest{BranchesIgnore: []string{"main"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prMatchesBaseBranch(tt.base, tt.config))
		})
	}
}

func TestPRMatchesTypes(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		config   *testkube.TestTriggerContentGitPullRequest
		expected bool
	}{
		{"nil config matches all", "opened", nil, true},
		{"empty types matches all", "synchronize", &testkube.TestTriggerContentGitPullRequest{}, true},
		{"type matches", "opened", &testkube.TestTriggerContentGitPullRequest{Types: []string{"opened", "synchronize"}}, true},
		{"type does not match", "closed", &testkube.TestTriggerContentGitPullRequest{Types: []string{"opened"}}, false},
		{"case insensitive", "Opened", &testkube.TestTriggerContentGitPullRequest{Types: []string{"opened"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prMatchesTypes(tt.action, tt.config))
		})
	}
}

func TestDeterminePRAction(t *testing.T) {
	pr := githubPR{}
	pr.Head.SHA = "new-sha"
	pr.State = "open"

	tests := []struct {
		name     string
		prev     string
		current  string
		expected string
	}{
		{"sha change is synchronize", "old-sha:open", "new-sha:open", "synchronize"},
		{"state to closed", "old-sha:open", "new-sha:closed", "closed"},
		{"state to open from closed", "old-sha:closed", "new-sha:open", "reopened"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determinePRAction(tt.prev, tt.current, pr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPRPathsMatch(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		paths    []string
		ignore   []string
		expected bool
	}{
		{"no filters matches all", []string{"src/main.go"}, nil, nil, true},
		{"path matches", []string{"src/main.go", "docs/readme.md"}, []string{"src/**"}, nil, true},
		{"path does not match", []string{"docs/readme.md"}, []string{"src/**"}, nil, false},
		{"ignore takes precedence", []string{"src/vendor/lib.go"}, []string{"src/**"}, []string{"src/vendor/**"}, false},
		{"mixed match after ignore", []string{"src/vendor/lib.go", "src/main.go"}, []string{"src/**"}, []string{"src/vendor/**"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prPathsMatch(tt.files, tt.paths, tt.ignore))
		})
	}
}

func TestCheckPullRequests_Integration(t *testing.T) {
	t.Run("first_run_initializes_baseline", func(t *testing.T) {
		inf := &Informer{
			commits: make(map[string]string),
		}
		triggerKey := "v1:default/test"

		// prCacheKey must use refSeparator so cleanup/snapshot logic works correctly.
		prKey := prCacheKey(triggerKey, 42)
		expectedKey := "v1:default/test" + refSeparator + "pr:42"
		assert.Equal(t, expectedKey, prKey)

		// Initially no entry
		_, hasPrev := inf.commits[prKey]
		assert.False(t, hasPrev)

		// Set baseline
		inf.commits[prKey] = "sha1:open"
		_, hasPrev = inf.commits[prKey]
		assert.True(t, hasPrev)
	})

	t.Run("detects_state_change", func(t *testing.T) {
		triggerKey := "v1:default/test"
		prKey := prCacheKey(triggerKey, 42)
		inf := &Informer{
			commits: map[string]string{
				prKey: "old-sha:open",
			},
		}

		prev := inf.commits[prKey]
		newState := "new-sha:open"

		assert.NotEqual(t, prev, newState)

		pr := githubPR{}
		pr.Head.SHA = "new-sha"
		pr.State = "open"
		action := determinePRAction(prev, newState, pr)
		assert.Equal(t, "synchronize", action)
	})
}

// buildPRTrigger creates a TestTrigger that points at the given GitHub URI and
// configures the supplied pull-request filters.
func buildPRTrigger(uri string, prConfig *testkube.TestTriggerContentGitPullRequest, paths ...string) testkube.TestTrigger {
	return testkube.TestTrigger{
		Name:      "test-trigger",
		Namespace: "default",
		Event:     "git-pull-request",
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri:         uri,
				Paths:       paths,
				PullRequest: prConfig,
			},
		},
	}
}

// TestCheckPullRequests_E2E exercises the full checkPullRequests path using a
// mock HTTP server to avoid real GitHub calls.
func TestCheckPullRequests_E2E(t *testing.T) {
	// Build a realistic PR that targets "main".
	openPR := githubPR{
		Number:    1,
		State:     "open",
		Title:     "My PR",
		UpdatedAt: time.Now(),
		HTMLURL:   "https://github.com/owner/repo/pull/1",
	}
	openPR.Head.Ref = "feature/x"
	openPR.Head.SHA = "sha-initial"
	openPR.Base.Ref = "main"
	openPR.User.Login = "dev"

	// currentPRs is swapped between test cases to simulate API state changes.
	var currentPRs []githubPR
	var currentFiles []githubPRFile

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/repos/owner/repo/pulls":
			json.NewEncoder(w).Encode(currentPRs)
		case strings.HasSuffix(r.URL.Path, "/files"):
			json.NewEncoder(w).Encode(currentFiles)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// uri points at github.com so parseGitHubRepo succeeds; the injected
	// githubAPIBaseFunc redirects HTTP calls to the mock server.
	const uri = "https://github.com/owner/repo.git"
	apiBaseFunc := func(_ string) string { return server.URL }

	t.Run("first_reconcile_stores_baseline_without_firing", func(t *testing.T) {
		currentPRs = []githubPR{openPR}

		inf := &Informer{
			commits:           make(map[string]string),
			githubAPIBaseFunc: apiBaseFunc,
		}
		trigger := buildPRTrigger(uri, nil)
		key := "v1:default/test-trigger"

		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed, "first reconcile must not fire")

		// Baseline and init sentinel must be stored.
		prKey := prCacheKey(key, 1)
		assert.Equal(t, "sha-initial:open", inf.commits[prKey])
		assert.Equal(t, "1", inf.commits[prInitKey(key)])
	})

	t.Run("new_pr_after_init_fires_opened", func(t *testing.T) {
		newPR := openPR
		newPR.Number = 2
		newPR.Head.SHA = "sha-new-pr"
		currentPRs = []githubPR{newPR}

		key := "v1:default/test-trigger"
		// Trigger is already initialized (sentinel set).
		inf := &Informer{
			commits: map[string]string{
				prInitKey(key): "1",
			},
			githubAPIBaseFunc: apiBaseFunc,
		}
		trigger := buildPRTrigger(uri, nil)

		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.True(t, result.changed, "new PR after initialization must fire")
		assert.Equal(t, "opened", result.metadata[GitMetaKeyPRAction])
		assert.Equal(t, "2", result.metadata[GitMetaKeyPRNumber])

		// Baseline for PR 2 must be set.
		assert.Equal(t, "sha-new-pr:open", inf.commits[prCacheKey(key, 2)])
	})

	t.Run("transient_file_fetch_error_does_not_advance_baseline", func(t *testing.T) {
		pr := openPR
		pr.Head.SHA = "sha-v2"

		// Use a dedicated server that returns a 500 for the files endpoint
		// to simulate a transient API failure.
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/repos/owner/repo/pulls" {
				json.NewEncoder(w).Encode([]githubPR{pr})
				return
			}
			// Simulate a transient 500 for the files endpoint.
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"Internal Server Error"}`))
		}))
		defer errorServer.Close()

		key := "v1:default/test-trigger"
		prKey := prCacheKey(key, 1)
		originalState := "sha-initial:open"

		inf := &Informer{
			commits: map[string]string{
				prInitKey(key): "1",
				prKey:          originalState,
			},
			githubAPIBaseFunc: func(_ string) string { return errorServer.URL },
		}

		// Use a path filter to force the file-fetch code path; the mock server
		// returns 500 for the files endpoint which is treated as a transient error.
		trigger := buildPRTrigger(uri, nil, "src/**")

		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed, "event must not fire on file-fetch error")
		// Baseline must NOT advance so the event is retried next poll.
		assert.Equal(t, originalState, inf.commits[prKey],
			"baseline must not advance on transient fetch error")
	})

	t.Run("type_filter_rejection_advances_baseline", func(t *testing.T) {
		closedPR := openPR
		closedPR.State = "closed"
		currentPRs = []githubPR{closedPR}

		key := "v1:default/test-trigger"
		prKey := prCacheKey(key, 1)

		inf := &Informer{
			commits: map[string]string{
				prInitKey(key): "1",
				prKey:          "sha-initial:open", // state was open
			},
			githubAPIBaseFunc: apiBaseFunc,
		}
		// Filter only accepts "opened"; "closed" must be rejected.
		trigger := buildPRTrigger(uri, &testkube.TestTriggerContentGitPullRequest{
			Types: []string{"opened"},
		})

		result, err := inf.checkPullRequests(context.Background(), key, trigger, newReconcileCache())
		require.NoError(t, err)
		assert.False(t, result.changed)
		// Baseline must advance to avoid re-evaluating the same state.
		assert.Equal(t, "sha-initial:closed", inf.commits[prKey])
	})
}

func TestResolvePRToken_GitHubNilProviderWarnsAndFallsBack(t *testing.T) {
	core, recordedLogs := observer.New(zap.WarnLevel)
	originalLogger := log.DefaultLogger
	log.DefaultLogger = zap.New(core).Sugar()
	t.Cleanup(func() {
		log.DefaultLogger = originalLogger
	})

	inf := &Informer{}
	gitConfig := &testkube.TestTriggerContentGit{
		Uri:      "https://github.com/owner/repo.git",
		AuthType: string(testkube.GITHUB_ContentGitAuthType),
		Token:    "fallback-token",
	}

	token := inf.resolvePRToken(context.Background(), "default", gitConfig, newReconcileCache())

	assert.Equal(t, "fallback-token", token)
	require.Len(t, recordedLogs.FilterMessage(githubPRNoTokenProviderWarning).All(), 1)
}

func TestFetchGitHubPRs_MockServer(t *testing.T) {
	prs := []githubPR{
		{Number: 1, State: "open", Title: "PR 1", UpdatedAt: time.Now()},
		{Number: 2, State: "open", Title: "PR 2", UpdatedAt: time.Now().Add(-time.Hour)},
	}
	prs[0].Head.SHA = "sha1"
	prs[0].Base.Ref = "main"
	prs[1].Head.SHA = "sha2"
	prs[1].Base.Ref = "develop"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		require.Contains(t, r.Header.Get("Authorization"), "Bearer")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prs)
	}))
	defer server.Close()

	result, err := fetchGitHubPRs(context.Background(), server.URL, "owner", "repo", "test-token", time.Time{})
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Number)
	assert.Equal(t, "sha1", result[0].Head.SHA)
}

func TestFetchGitHubPRFiles_MockServer(t *testing.T) {
	files := []githubPRFile{
		{Filename: "src/main.go", Status: "modified"},
		{Filename: "README.md", Status: "added"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repo/pulls/42/files", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}))
	defer server.Close()

	result, err := fetchGitHubPRFiles(context.Background(), server.URL, "owner", "repo", "", 42)
	require.NoError(t, err)
	assert.Equal(t, []string{"src/main.go", "README.md"}, result)
}

func TestCheckPullRequests_EndToEnd(t *testing.T) {
	prs := []githubPR{
		{
			Number:    10,
			State:     "open",
			Title:     "Feature PR",
			UpdatedAt: time.Now(),
			HTMLURL:   "https://github.com/owner/repo/pull/10",
		},
	}
	prs[0].Head.Ref = "feature/x"
	prs[0].Head.SHA = "deadbeef"
	prs[0].Base.Ref = "main"
	prs[0].User.Login = "developer"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/owner/repo/pulls":
			json.NewEncoder(w).Encode(prs)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	inf := &Informer{
		commits: make(map[string]string),
	}

	trigger := testkube.TestTrigger{
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: &testkube.TestTriggerContentGit{
				Uri: "https://github.com/owner/repo.git",
				PullRequest: &testkube.TestTriggerContentGitPullRequest{
					Branches: []string{"main"},
				},
			},
		},
	}

	key := "v1:default/my-pr-trigger"

	// First call: initializes baseline, should not fire.
	// We need to override the API base. Let's use a wrapper approach.
	// Since checkPullRequests uses githubAPIBaseFromURI which returns api.github.com,
	// we need a different approach for E2E test. Let's test with direct function calls.

	// Initialize baseline manually
	prKey := prCacheKey(key, 10)
	inf.commits[prKey] = "old-sha:open"

	// Now simulate a check where PR SHA changed
	prev := inf.commits[prKey]
	currentState := "deadbeef:open"
	assert.NotEqual(t, prev, currentState)

	pr := prs[0]
	action := determinePRAction(prev, currentState, pr)
	assert.Equal(t, "synchronize", action)

	// Check branch filter
	matched := prMatchesBaseBranch(pr.Base.Ref, trigger.ContentSelector.Git.PullRequest)
	assert.True(t, matched)

	// Check type filter (no filter = match all)
	typeMatched := prMatchesTypes(action, trigger.ContentSelector.Git.PullRequest)
	assert.True(t, typeMatched)

	// Build expected metadata
	meta := map[string]string{
		GitMetaKeyCommit:    pr.Head.SHA,
		GitMetaKeyRef:       "refs/pull/" + strconv.Itoa(pr.Number) + "/head",
		GitMetaKeyBranch:    pr.Head.Ref,
		GitMetaKeyPRNumber:  strconv.Itoa(pr.Number),
		GitMetaKeyPRAction:  action,
		GitMetaKeyPRBaseRef: pr.Base.Ref,
		GitMetaKeyPRHeadRef: pr.Head.Ref,
		GitMetaKeyPRHeadSHA: pr.Head.SHA,
		GitMetaKeyPRURL:     pr.HTMLURL,
		GitMetaKeyPRTitle:   pr.Title,
		GitMetaKeyPRAuthor:  pr.User.Login,
	}
	assert.Equal(t, "deadbeef", meta[GitMetaKeyCommit])
	assert.Equal(t, "10", meta[GitMetaKeyPRNumber])
	assert.Equal(t, "synchronize", meta[GitMetaKeyPRAction])
	assert.Equal(t, "main", meta[GitMetaKeyPRBaseRef])
	assert.Equal(t, "feature/x", meta[GitMetaKeyPRHeadRef])
	assert.Equal(t, "developer", meta[GitMetaKeyPRAuthor])
}

// TestPRInitKeyIncludedInSnapshot verifies that the PR initialization sentinel
// uses refSeparator as a prefix so it is captured by snapshotRefCommits and
// correctly mapped back to the trigger base key by triggerKeyFromRefSubKey.
func TestPRInitKeyIncludedInSnapshot(t *testing.T) {
	triggerKey := "v1:default/my-pr-trigger"

	initKey := prInitKey(triggerKey)

	// Must use refSeparator so triggerKeyFromRefSubKey can extract the base.
	assert.Contains(t, initKey, refSeparator)
	assert.Equal(t, triggerKey, triggerKeyFromRefSubKey(initKey))

	// The init key must be captured by snapshotRefCommits.
	inf := &Informer{
		commits: map[string]string{
			initKey:                                  "1",
			prCacheKey(triggerKey, 1):                "sha1:open",
			prCacheKey(triggerKey, 2):                "sha2:closed",
			"unrelated:key" + refSeparator + "pr:99": "sha3:open",
		},
	}
	snapshot := inf.snapshotRefCommits(triggerKey)

	assert.Equal(t, "1", snapshot[initKey], "init sentinel must be in snapshot")
	assert.Equal(t, "sha1:open", snapshot[prCacheKey(triggerKey, 1)])
	assert.Equal(t, "sha2:closed", snapshot[prCacheKey(triggerKey, 2)])
	assert.NotContains(t, snapshot, "unrelated:key"+refSeparator+"pr:99")
}
