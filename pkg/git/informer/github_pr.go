package informer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/git/matchers"
	"github.com/kubeshop/testkube/pkg/log"
)

// githubPR represents a minimal GitHub pull request from the REST API.
type githubPR struct {
	Number    int       `json:"number"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
	Head      struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	Draft bool `json:"draft"`
}

// githubPRFile represents a file changed in a pull request.
type githubPRFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
}

var githubRepoPattern = regexp.MustCompile(`(?:github\.com|github\.[^/:]+)[/:]([^/]+)/([^/]+?)(?:\.git)?/?$`)

// githubHTTPClient is used for GitHub API requests with an appropriate timeout.
var githubHTTPClient = &http.Client{Timeout: 30 * time.Second}

const githubPRNoTokenProviderWarning = "github authType configured for PR polling but no GitHub token provider is available, falling back to configured credentials"

// parseGitHubRepo extracts owner/repo from a GitHub URL (HTTPS or SSH).
func parseGitHubRepo(uri string) (owner, repo string, ok bool) {
	matches := githubRepoPattern.FindStringSubmatch(uri)
	if len(matches) < 3 {
		return "", "", false
	}
	return matches[1], matches[2], true
}

// sanitizeGitHubTokenURI strips userinfo from GitHub URIs before they are sent
// to the token provider or used as cache keys. If parsing fails or no userinfo
// is present, the original URI is returned unchanged.
func sanitizeGitHubTokenURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.User == nil {
		return uri
	}
	u.User = nil
	return u.String()
}

// githubAPIBaseFromURI returns the GitHub API base URL for the given repo URI.
// For github.com it returns "https://api.github.com", for GHES it derives from the host.
func githubAPIBaseFromURI(uri string) string {
	// Try to parse as URL
	u, err := url.Parse(uri)
	if err == nil && u.Host != "" && u.Host != "github.com" && !strings.HasSuffix(u.Host, "github.com") {
		// GHES
		return fmt.Sprintf("https://%s/api/v3", u.Host)
	}
	return "https://api.github.com"
}

// fetchGitHubPRs fetches open pull requests from the GitHub REST API.
// Note: fetches up to 30 most recently updated PRs per call (GitHub default page size).
// PRs updated beyond this window may be missed if the reconciliation interval is too long.
func fetchGitHubPRs(ctx context.Context, apiBase, owner, repo, token string, since time.Time) ([]githubPR, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls?state=all&sort=updated&direction=desc&per_page=30",
		apiBase, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := githubHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var prs []githubPR
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}

	// Filter by update time
	if !since.IsZero() {
		filtered := prs[:0]
		for _, pr := range prs {
			if !pr.UpdatedAt.Before(since) {
				filtered = append(filtered, pr)
			}
		}
		prs = filtered
	}

	return prs, nil
}

// fetchGitHubPRFiles fetches the list of files changed in a pull request.
// Note: fetches up to 100 files per call (GitHub maximum page size).
// PRs with more than 100 changed files will have incomplete path matching.
func fetchGitHubPRFiles(ctx context.Context, apiBase, owner, repo, token string, prNumber int) ([]string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=100",
		apiBase, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := githubHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned %d for PR files: %s", resp.StatusCode, string(body))
	}

	var files []githubPRFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Filename)
	}
	return paths, nil
}

// isPullRequestTrigger returns true if any of the trigger's effective events
// is git-pull-request.
func isPullRequestTrigger(trigger testkube.TestTrigger) bool {
	for _, e := range trigger.EffectiveEvents() {
		if strings.ToLower(e) == string(testtriggersv1.TestTriggerEventGitPullRequest) {
			return true
		}
	}
	return false
}

// checkPullRequests polls GitHub for PRs matching the trigger configuration and fires events.
func (i *Informer) checkPullRequests(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (matchResult, error) {
	gitConfig := trigger.ContentSelector.Git
	if gitConfig == nil {
		return matchResult{}, nil
	}

	owner, repo, ok := parseGitHubRepo(gitConfig.Uri)
	if !ok {
		return matchResult{}, fmt.Errorf("git-pull-request trigger requires a GitHub repository URL, got: %s", gitConfig.Uri)
	}

	// Resolve token for API authentication.
	token := i.resolvePRToken(ctx, trigger.Namespace, gitConfig, cache)
	apiBase := githubAPIBaseFromURI(gitConfig.Uri)
	if i.githubAPIBaseFunc != nil {
		apiBase = i.githubAPIBaseFunc(gitConfig.Uri)
	}

	// Fetch PRs (up to 30 most recently updated, per GitHub default page size).
	prs, err := fetchGitHubPRs(ctx, apiBase, owner, repo, token, time.Time{})
	if err != nil {
		return matchResult{}, fmt.Errorf("failed to fetch PRs: %w", err)
	}

	prConfig := gitConfig.PullRequest

	paths := matchers.NormalizePaths(gitConfig.Paths)
	pathsIgnore := matchers.NormalizePaths(gitConfig.PathsIgnore)

	// On the first reconcile pass for a trigger all currently-visible PRs are
	// recorded as baselines without firing.  Once the sentinel is set, a PR
	// that has never been seen before is treated as "opened".
	initKey := prInitKey(key)
	prInitialized := i.commits[initKey] != ""

	for _, pr := range prs {
		// Apply base branch filters before state tracking. A nil prConfig
		// preserves the pre-refactor "match all" semantic.
		if prConfig != nil && !matchers.PRMatchesBaseBranch(pr.Base.Ref, prConfig.Branches, prConfig.BranchesIgnore) {
			continue
		}

		prKey := prCacheKey(key, pr.Number)
		prev, hasPrev := i.commits[prKey]
		currentState := pr.Head.SHA + ":" + pr.State

		if !hasPrev {
			if !prInitialized {
				// Initial baseline pass: record the current state without firing.
				i.commits[prKey] = currentState
				log.DefaultLogger.Infof("git informer: initializing PR baseline for trigger %s/%s PR #%d",
					trigger.Namespace, trigger.Name, pr.Number)
				continue
			}
			// Trigger is initialized: a PR that has never been seen is "opened".
		} else if prev == currentState {
			continue // No change
		}

		// Determine action.
		var action string
		if !hasPrev {
			action = "opened"
		} else {
			action = matchers.DeterminePRAction(prev, currentState, pr.Head.SHA)
		}

		// Apply type filter. Advance baseline so the same state is not re-evaluated
		// on the next reconcile regardless of whether the action matches. A nil
		// prConfig preserves the pre-refactor "match all" semantic.
		if prConfig != nil && !matchers.PRMatchesTypes(action, prConfig.Types) {
			i.commits[prKey] = currentState
			continue
		}

		// Apply path filters if configured.
		if len(paths) > 0 || len(pathsIgnore) > 0 {
			changedFiles, fileErr := fetchGitHubPRFiles(ctx, apiBase, owner, repo, token, pr.Number)
			if fileErr != nil {
				// Transient error: do NOT advance the baseline so the event can be
				// retried on the next reconcile.
				log.DefaultLogger.Warnf("git informer: failed to fetch PR #%d files: %v", pr.Number, fileErr)
				continue
			}
			if !matchers.PRPathsMatch(changedFiles, paths, pathsIgnore) {
				// Paths do not match; advance baseline to skip this state on the next pass.
				i.commits[prKey] = currentState
				continue
			}
		}

		// All filters passed: advance baseline and fire event.
		i.commits[prKey] = currentState

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
		return matchResult{changed: true, metadata: meta}, nil
	}

	// Mark the trigger as initialized after completing the first baseline pass.
	if !prInitialized {
		i.commits[initKey] = "1"
	}

	return matchResult{}, nil
}

func (i *Informer) resolvePRToken(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit, cache *reconcileCache) string {
	// If authType is "github", fetch token from control plane.
	authType := strings.ToLower(gitConfig.AuthType)
	if authType == string(testkube.GITHUB_ContentGitAuthType) {
		if i.githubTokenProvider == nil {
			log.DefaultLogger.Warnw(githubPRNoTokenProviderWarning)
		} else {
			// Use the per-reconcile cache to avoid repeated gRPC calls for the same URI.
			uri := sanitizeGitHubTokenURI(gitConfig.Uri)

			if token, ok := cache.githubToken(uri); ok {
				return token
			}
			token, err := i.githubTokenProvider.GetGitHubToken(ctx, uri)
			if err != nil {
				log.DefaultLogger.Warnw("failed to get GitHub App token for PR polling, falling back to configured credentials", "error", err)
			} else if strings.TrimSpace(token) == "" {
				log.DefaultLogger.Warnw("received empty GitHub App token for PR polling, falling back to configured credentials")
			} else {
				cache.setGitHubToken(uri, token)
				return token
			}
		}
	}
	if i.kubeClient != nil {
		return i.resolveCredentialValue(ctx, gitConfig.Token, namespace, gitConfig.TokenFrom)
	}
	return resolveCredentialValue(gitConfig.Token, gitConfig.TokenFrom)
}

func prCacheKey(triggerKey string, prNumber int) string {
	// Use refSeparator so PR baselines are treated like other per-trigger sub-keys
	// by snapshot/restore and cleanup logic.
	return triggerKey + refSeparator + "pr:" + strconv.Itoa(prNumber)
}

// prInitKey returns the commit-map key used as a sentinel to mark that the
// trigger has completed its initial PR baseline pass. After the sentinel is
// set, new PRs that have never been seen before are treated as "opened".
func prInitKey(triggerKey string) string {
	return triggerKey + refSeparator + "pr:__init__"
}
