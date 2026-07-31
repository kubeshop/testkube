package informer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

// prHTTPClient is used for git provider API requests with an appropriate timeout.
var prHTTPClient = &http.Client{Timeout: 30 * time.Second}

// Normalized pull request states. Provider-specific states are mapped onto these
// before they reach the shared matching logic, which keeps determinePRAction and
// the baseline encoding provider-agnostic.
const (
	prStateOpen   = "open"
	prStateClosed = "closed"
)

// Pull request action names. GitHub's vocabulary is used for every provider so a
// single trigger definition works regardless of where the repository is hosted.
const (
	prActionOpened      = "opened"
	prActionSynchronize = "synchronize"
	prActionReopened    = "reopened"
	prActionClosed      = "closed"
)

// pullRequest is the provider-agnostic view of a GitHub pull request or a GitLab
// merge request.
type pullRequest struct {
	// Number is the GitHub pull request number or the GitLab merge request iid,
	// i.e. the identifier shown in the web UI.
	Number int
	// State is normalized to prStateOpen or prStateClosed.
	State     string
	Title     string
	HeadRef   string
	HeadSHA   string
	BaseRef   string
	Author    string
	URL       string
	UpdatedAt time.Time
	Draft     bool
}

// prProvider polls a git hosting provider for pull requests / merge requests. A
// provider is constructed per trigger with its API base, repository identity and
// token already resolved.
type prProvider interface {
	// kind reports which provider this is; used for logging and cache scoping.
	kind() providerKind
	// list returns the most recently updated pull requests, newest first.
	list(ctx context.Context) ([]pullRequest, error)
	// changedFiles returns the paths touched by the given pull request. truncated
	// reports that the provider capped the file list, so the absence of a path
	// proves nothing.
	changedFiles(ctx context.Context, number int) (paths []string, truncated bool, err error)
	// headRef returns the canonical remote ref for the pull request head.
	headRef(number int) string
}

// isPullRequestTrigger returns true if the trigger is configured for git-pull-request events.
func isPullRequestTrigger(trigger testkube.TestTrigger) bool {
	return strings.ToLower(trigger.Event) == string(testtriggersv1.TestTriggerEventGitPullRequest)
}

// sanitizeTokenURI strips userinfo from repository URIs before they are sent to a
// token provider or used as cache keys. If parsing fails or no userinfo is
// present, the original URI is returned unchanged.
func sanitizeTokenURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.User == nil {
		return uri
	}
	u.User = nil
	return u.String()
}

// prAPIGet issues an authenticated GET against a provider REST API and decodes the
// JSON response into out. Both supported providers accept OAuth-style bearer
// tokens, so a single auth scheme covers GitHub tokens as well as GitLab
// personal, project and group access tokens.
func prAPIGet(ctx context.Context, label, endpoint, token, accept string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := prHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%s API returned %d: %s", label, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// checkPullRequests polls the configured provider for pull requests matching the
// trigger configuration and fires an event for the first match.
func (i *Informer) checkPullRequests(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (matchResult, error) {
	gitConfig := trigger.ContentSelector.Git
	if gitConfig == nil {
		return matchResult{}, nil
	}

	provider, err := i.prProviderFor(ctx, trigger.Namespace, gitConfig, cache)
	if err != nil {
		return matchResult{}, err
	}
	kind := provider.kind()

	prs, err := provider.list(ctx)
	if err != nil {
		return matchResult{}, fmt.Errorf("failed to fetch %s pull requests: %w", kind, err)
	}

	prConfig := gitConfig.PullRequest

	paths := normalizePaths(gitConfig.Paths)
	pathsIgnore := normalizePaths(gitConfig.PathsIgnore)

	// On the first reconcile pass for a trigger all currently-visible PRs are
	// recorded as baselines without firing.  Once the sentinel is set, a PR
	// that has never been seen before is treated as "opened".
	initKey := prInitKey(key, kind)
	prInitialized := i.commits[initKey] != ""

	for _, pr := range prs {
		// GitLab populates the head commit asynchronously, so a freshly created
		// merge request can be visible before its SHA exists. Skip it without
		// recording a baseline so the next pass reports a real "opened" event
		// instead of firing now with empty commit metadata.
		if pr.HeadSHA == "" {
			log.DefaultLogger.Debugf("git informer: skipping pull request #%d for trigger %s/%s: head commit not available yet",
				pr.Number, trigger.Namespace, trigger.Name)
			continue
		}

		// Apply base branch filters before state tracking.
		if !prMatchesBaseBranch(pr.BaseRef, prConfig) {
			continue
		}

		prKey := prCacheKey(key, kind, pr.Number)
		prev, hasPrev := i.commits[prKey]
		currentState := pr.HeadSHA + ":" + pr.State

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
			action = prActionOpened
		} else {
			action = determinePRAction(prev, currentState)
		}

		// Apply type filter. Advance baseline so the same state is not re-evaluated
		// on the next reconcile regardless of whether the action matches.
		if !prMatchesTypes(action, prConfig) {
			i.commits[prKey] = currentState
			continue
		}

		// Apply path filters if configured.
		if len(paths) > 0 || len(pathsIgnore) > 0 {
			changedFiles, truncated, fileErr := provider.changedFiles(ctx, pr.Number)
			if fileErr != nil {
				// Transient error: do NOT advance the baseline so the event can be
				// retried on the next reconcile.
				log.DefaultLogger.Warnf("git informer: failed to fetch PR #%d files: %v", pr.Number, fileErr)
				continue
			}
			if !prPathsMatch(changedFiles, paths, pathsIgnore) {
				if !truncated {
					// Paths do not match; advance baseline to skip this state on the next pass.
					i.commits[prKey] = currentState
					continue
				}
				// The provider capped the file list, so a matching path may sit in
				// the part that was never fetched. Advancing the baseline here would
				// drop this event permanently, so fire instead: a superfluous run is
				// recoverable, a silently skipped trigger is not.
				log.DefaultLogger.Warnf(
					"git informer: PR #%d for trigger %s/%s has a truncated changed-file list and no match among the files fetched; firing anyway because a match cannot be ruled out",
					pr.Number, trigger.Namespace, trigger.Name)
			}
		}

		// All filters passed: advance baseline and fire event.
		i.commits[prKey] = currentState

		meta := map[string]string{
			GitMetaKeyCommit:    pr.HeadSHA,
			GitMetaKeyRef:       provider.headRef(pr.Number),
			GitMetaKeyBranch:    pr.HeadRef,
			GitMetaKeyPRNumber:  strconv.Itoa(pr.Number),
			GitMetaKeyPRAction:  action,
			GitMetaKeyPRBaseRef: pr.BaseRef,
			GitMetaKeyPRHeadRef: pr.HeadRef,
			GitMetaKeyPRHeadSHA: pr.HeadSHA,
			GitMetaKeyPRURL:     pr.URL,
			GitMetaKeyPRTitle:   pr.Title,
			GitMetaKeyPRAuthor:  pr.Author,
		}
		return matchResult{changed: true, metadata: meta}, nil
	}

	// Mark the trigger as initialized after completing the first baseline pass.
	if !prInitialized {
		i.commits[initKey] = "1"
	}

	return matchResult{}, nil
}

// resolvePRToken resolves the API token used to poll the provider. The GitHub App
// flow only applies to GitHub, since the control plane mints GitHub installation
// tokens exclusively.
func (i *Informer) resolvePRToken(
	ctx context.Context,
	namespace string,
	gitConfig *testkube.TestTriggerContentGit,
	kind providerKind,
	cache *reconcileCache,
) string {
	// If authType is "github", fetch token from control plane.
	authType := strings.ToLower(gitConfig.AuthType)
	if authType == string(testkube.GITHUB_ContentGitAuthType) {
		switch {
		case kind != providerGitHub:
			log.DefaultLogger.Warnw(prGitHubAuthTypeWrongProviderWarning, "provider", string(kind))
		case i.githubTokenProvider == nil:
			log.DefaultLogger.Warnw(githubPRNoTokenProviderWarning)
		default:
			// Use the per-reconcile cache to avoid repeated gRPC calls for the same URI.
			uri := sanitizeTokenURI(gitConfig.Uri)

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
	return i.resolvePlainToken(ctx, namespace, gitConfig)
}

// prKeyPrefix returns the commit-map sub-key prefix for a provider. GitHub keeps
// the historical unprefixed form so existing baselines survive an upgrade; other
// providers are namespaced so that repointing a trigger's uri at a different
// provider re-baselines instead of firing a burst of stale events against
// numbering that means something else.
func prKeyPrefix(kind providerKind) string {
	if kind == providerGitHub {
		return "pr:"
	}
	return "pr:" + string(kind) + ":"
}

func prCacheKey(triggerKey string, kind providerKind, prNumber int) string {
	// Use refSeparator so PR baselines are treated like other per-trigger sub-keys
	// by snapshot/restore and cleanup logic.
	return triggerKey + refSeparator + prKeyPrefix(kind) + strconv.Itoa(prNumber)
}

// prInitKey returns the commit-map key used as a sentinel to mark that the
// trigger has completed its initial PR baseline pass. After the sentinel is
// set, new PRs that have never been seen before are treated as "opened".
func prInitKey(triggerKey string, kind providerKind) string {
	return triggerKey + refSeparator + prKeyPrefix(kind) + "__init__"
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

// determinePRAction determines the PR action based on state transitions. Both
// baselines are encoded as "<headSHA>:<normalizedState>".
func determinePRAction(prevEncoded, currentEncoded string) string {
	prevSHA, prevState := splitPRBaseline(prevEncoded)
	currSHA, currState := splitPRBaseline(currentEncoded)

	// State changed to closed
	if currState == prStateClosed && prevState != prStateClosed {
		return prActionClosed
	}
	// State changed from closed to open (reopened)
	if currState == prStateOpen && prevState == prStateClosed {
		return prActionReopened
	}
	// SHA changed (new commits pushed)
	if prevSHA != currSHA {
		return prActionSynchronize
	}
	// Catch-all
	return prActionSynchronize
}

// splitPRBaseline splits an encoded "<headSHA>:<normalizedState>" baseline. Both
// components are empty when the encoding is not recognised.
func splitPRBaseline(encoded string) (sha, state string) {
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
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
