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
	"github.com/kubeshop/testkube/pkg/git/matchers"
	"github.com/kubeshop/testkube/pkg/log"
)

// prHTTPClient is used for git provider API requests with an appropriate timeout.
var prHTTPClient = &http.Client{Timeout: 30 * time.Second}

// Normalized pull request states. Provider-specific states are mapped onto these
// before they reach the shared matching logic, which keeps matchers.DeterminePRAction
// and the baseline encoding provider-agnostic.
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
	// list returns the most recently updated pull requests, newest first. cutoff
	// bounds how far back pagination walks; see paginatePRList.
	list(ctx context.Context, cutoff time.Time) ([]pullRequest, error)
	// fetchPRPage returns a single page of pull requests, newest first.
	fetchPRPage(ctx context.Context, page int) ([]pullRequest, error)
	// listPageSize is the page size fetchPRPage requests, used to recognise a
	// short (final) page.
	listPageSize() int
	// changedFiles returns the paths touched by the given pull request. truncated
	// reports that the provider capped the file list, so the absence of a path
	// proves nothing.
	changedFiles(ctx context.Context, number int) (paths []string, truncated bool, err error)
	// headRef returns the canonical remote ref for the pull request head.
	headRef(number int) string
}

const (
	// prListMaxPages bounds how many pages of pull requests a single reconcile pass
	// will walk, so one trigger cannot monopolise the informer.
	prListMaxPages = 10

	// prListMinLookback is the minimum age of activity that pagination reaches back
	// to. Both providers list every historical pull request when asked for all
	// states, and neither bounds that result set, so pagination has to stop
	// somewhere; the cutoff is what makes the walk finite.
	prListMinLookback = 24 * time.Hour

	// prListLookbackIntervals keeps the lookback comfortably wider than the poll
	// interval, so an unusually long interval cannot outrun the window.
	prListLookbackIntervals = 20
)

// prListCutoff returns the oldest update time pagination should reach back to.
func (i *Informer) prListCutoff(now time.Time) time.Time {
	lookback := time.Duration(prListMinLookback)
	if scaled := i.options.ReconcileInterval * prListLookbackIntervals; scaled > lookback {
		lookback = scaled
	}
	return now.Add(-lookback)
}

// paginatePRList walks pages of pull requests until the provider runs out, until
// the results fall outside the lookback window, or until the page cap.
//
// The first page is ALWAYS fetched in full regardless of cutoff. That matters:
// GitLab could filter server-side with updated_after, and GitHub cannot filter at
// all, but applying a time filter to the first page would shrink coverage below the
// single page this used to fetch, and a quiet project whose merge requests were all
// last touched months ago would lose its baselines entirely. Keeping page one
// unconditional makes coverage a superset of the previous behaviour in every case.
func paginatePRList(ctx context.Context, p prProvider, cutoff time.Time) ([]pullRequest, error) {
	pageSize := p.listPageSize()
	all := make([]pullRequest, 0, pageSize)

	for page := 1; page <= prListMaxPages; page++ {
		batch, err := p.fetchPRPage(ctx, page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)

		// A short page is the last page.
		if len(batch) < pageSize {
			return all, nil
		}
		// Results are ordered by update time descending, so once a full page ends
		// outside the window every later page is older still. A zero timestamp
		// counts as outside, which stops after page one rather than walking the
		// project's whole history.
		if oldest := batch[len(batch)-1].UpdatedAt; oldest.Before(cutoff) {
			return all, nil
		}
	}

	log.DefaultLogger.Warnf(
		"git informer: %s pull request list hit the %d page cap (%d entries); pull requests updated before that point were not examined this pass",
		p.kind(), prListMaxPages, len(all))
	return all, nil
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

	prs, err := provider.list(ctx, i.prListCutoff(time.Now()))
	if err != nil {
		return matchResult{}, fmt.Errorf("failed to fetch %s pull requests: %w", kind, err)
	}

	prConfig := gitConfig.PullRequest

	paths := matchers.NormalizePaths(gitConfig.Paths)
	pathsIgnore := matchers.NormalizePaths(gitConfig.PathsIgnore)

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
			action = matchers.DeterminePRAction(prev, currentState, pr.HeadSHA)
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
			if !matchers.PRPathsMatch(changedFiles, paths, pathsIgnore) {
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

// prMatchesBaseBranch adapts the trigger's pull request config to the shared
// matcher. A nil config means no filters are set, which matches every base branch.
func prMatchesBaseBranch(baseBranch string, prConfig *testkube.TestTriggerContentGitPullRequest) bool {
	if prConfig == nil {
		return true
	}
	return matchers.PRMatchesBaseBranch(baseBranch, prConfig.Branches, prConfig.BranchesIgnore)
}

// prMatchesTypes adapts the trigger's pull request config to the shared matcher.
// A nil config means no filter is set, which matches every action.
func prMatchesTypes(action string, prConfig *testkube.TestTriggerContentGitPullRequest) bool {
	if prConfig == nil {
		return true
	}
	return matchers.PRMatchesTypes(action, prConfig.Types)
}
