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

// prCacheEntry tracks the last seen state for a PR in a trigger's context.
type prCacheEntry struct {
	HeadSHA string
	State   string
}

var githubRepoPattern = regexp.MustCompile(`(?:github\.com)[/:]([^/]+)/([^/.]+?)(?:\.git)?$`)

// parseGitHubRepo extracts owner/repo from a GitHub URL (HTTPS or SSH).
func parseGitHubRepo(uri string) (owner, repo string, ok bool) {
	matches := githubRepoPattern.FindStringSubmatch(uri)
	if len(matches) < 3 {
		return "", "", false
	}
	return matches[1], matches[2], true
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

// isPullRequestTrigger returns true if the trigger is configured for git-pull-request events.
func isPullRequestTrigger(trigger testkube.TestTrigger) bool {
	return strings.ToLower(trigger.Event) == string(testtriggersv1.TestTriggerEventGitPullRequest)
}

// checkPullRequests polls GitHub for PRs matching the trigger configuration and fires events.
func (i *Informer) checkPullRequests(ctx context.Context, key string, trigger testkube.TestTrigger) (matchResult, error) {
	gitConfig := trigger.ContentSelector.Git
	if gitConfig == nil {
		return matchResult{}, nil
	}

	owner, repo, ok := parseGitHubRepo(gitConfig.Uri)
	if !ok {
		return matchResult{}, fmt.Errorf("git-pull-request trigger requires a GitHub repository URL, got: %s", gitConfig.Uri)
	}

	// Resolve token for API authentication.
	token := i.resolvePRToken(ctx, trigger.Namespace, gitConfig)
	apiBase := githubAPIBaseFromURI(gitConfig.Uri)

	// Fetch PRs updated since last check (or all if first run).
	prs, err := fetchGitHubPRs(ctx, apiBase, owner, repo, token, time.Time{})
	if err != nil {
		return matchResult{}, fmt.Errorf("failed to fetch PRs: %w", err)
	}

	prConfig := gitConfig.PullRequest

	paths := normalizePaths(gitConfig.Paths)
	pathsIgnore := normalizePaths(gitConfig.PathsIgnore)

	for _, pr := range prs {
		// Apply base branch filters.
		if !prMatchesBaseBranch(pr.Base.Ref, prConfig) {
			continue
		}

		// Determine the action (state change) for this PR.
		prKey := prCacheKey(key, pr.Number)
		prev, hasPrev := i.commits[prKey]

		// Encode current state as "sha:state" for tracking.
		currentState := pr.Head.SHA + ":" + pr.State
		i.commits[prKey] = currentState

		if !hasPrev {
			// First time seeing this PR — treat as baseline without firing.
			log.DefaultLogger.Infof("git informer: initializing PR baseline for trigger %s/%s PR #%d",
				trigger.Namespace, trigger.Name, pr.Number)
			continue
		}

		if prev == currentState {
			continue // No change
		}

		// Determine action type.
		action := determinePRAction(prev, currentState, pr)

		// Apply type filter.
		if !prMatchesTypes(action, prConfig) {
			continue
		}

		// Apply path filters if configured.
		if len(paths) > 0 || len(pathsIgnore) > 0 {
			changedFiles, fileErr := fetchGitHubPRFiles(ctx, apiBase, owner, repo, token, pr.Number)
			if fileErr != nil {
				log.DefaultLogger.Warnf("git informer: failed to fetch PR #%d files: %v", pr.Number, fileErr)
				continue
			}
			if !prPathsMatch(changedFiles, paths, pathsIgnore) {
				continue
			}
		}

		// Build metadata and return.
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

	return matchResult{}, nil
}

func (i *Informer) resolvePRToken(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit) string {
	if i.kubeClient != nil {
		return i.resolveCredentialValue(ctx, gitConfig.Token, namespace, gitConfig.TokenFrom)
	}
	return resolveCredentialValue(gitConfig.Token, gitConfig.TokenFrom)
}

func prCacheKey(triggerKey string, prNumber int) string {
	return triggerKey + "|pr:" + strconv.Itoa(prNumber)
}

// prMatchesBaseBranch checks if a PR's base branch matches the trigger's branch filters.
func prMatchesBaseBranch(baseBranch string, prConfig *testkube.TestTriggerContentGitPullRequest) bool {
	if prConfig == nil {
		return true
	}
	// Check ignore first (takes precedence)
	if nameMatchesAny(baseBranch, prConfig.BranchesIgnore) {
		return false
	}
	// If branches list is empty, match all
	if len(prConfig.Branches) == 0 {
		return true
	}
	return nameMatchesAny(baseBranch, prConfig.Branches)
}

// prMatchesTypes checks if a PR action matches the configured types filter.
func prMatchesTypes(action string, prConfig *testkube.TestTriggerContentGitPullRequest) bool {
	if prConfig == nil || len(prConfig.Types) == 0 {
		return true
	}
	for _, t := range prConfig.Types {
		if strings.EqualFold(strings.TrimSpace(t), action) {
			return true
		}
	}
	return false
}

// determinePRAction determines the PR action based on state transitions.
func determinePRAction(prevEncoded, currentEncoded string, pr githubPR) string {
	prevParts := strings.SplitN(prevEncoded, ":", 2)
	currParts := strings.SplitN(currentEncoded, ":", 2)

	prevSHA := ""
	prevState := ""
	if len(prevParts) == 2 {
		prevSHA = prevParts[0]
		prevState = prevParts[1]
	}

	currState := ""
	if len(currParts) == 2 {
		currState = currParts[1]
	}

	// State changed to closed
	if currState == "closed" && prevState != "closed" {
		return "closed"
	}
	// State changed from closed to open (reopened)
	if currState == "open" && prevState == "closed" {
		return "reopened"
	}
	// SHA changed (new commits pushed)
	if prevSHA != pr.Head.SHA {
		return "synchronize"
	}
	// Draft changed to ready
	if !pr.Draft && prevState == "open" && currState == "open" {
		return "ready_for_review"
	}
	// Catch-all
	return "synchronize"
}

// prPathsMatch checks if any changed file in a PR matches the path filters.
func prPathsMatch(changedFiles, paths, pathsIgnore []string) bool {
	for _, file := range changedFiles {
		if pathIsIgnored(pathsIgnore, file) {
			continue
		}
		if len(paths) == 0 || pathMatchesNormalized(paths, file) {
			return true
		}
	}
	return false
}
