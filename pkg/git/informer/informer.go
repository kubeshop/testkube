package informer

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	"github.com/go-git/go-git/v6/storage/memory"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
)

const (
	// EventGitPush fires when new commits are pushed to a watched branch.
	EventGitPush = "git-push"
	// EventGitTagPush fires when a new tag matching the filter is pushed.
	EventGitTagPush = "git-tag-push"
)

const defaultReconcileInterval = time.Minute
const defaultGitUsername = "git"
const testTriggerSource = "v1"
const allNamespacesMarker = "*"
const refDelimiter = "__"
const refSeparator = "|ref|"

// Git metadata keys passed to MatchGitTrigger.
const (
	GitMetaKeyCommit          = "TESTKUBE_GIT_COMMIT"
	GitMetaKeyRef             = "TESTKUBE_GIT_REF"
	GitMetaKeyBranch          = "TESTKUBE_GIT_BRANCH"
	GitMetaKeyTag             = "TESTKUBE_GIT_TAG"
	GitMetaKeyCommitMessage   = "TESTKUBE_GIT_COMMIT_MESSAGE"
	GitMetaKeyAuthor          = "TESTKUBE_GIT_AUTHOR"
	GitMetaKeyCommitTimestamp = "TESTKUBE_GIT_COMMIT_TIMESTAMP"
)

// envVarNameSanitizer normalizes Secret/ConfigMap name+key into env-var-safe tokens.
var envVarNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)
var gitCommitSHAPattern = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

type Options struct {
	ReconcileInterval  time.Duration
	RepoDepth          int
	ListTimeoutSeconds int
	MaxCommitsScan     int
	PullRetries        int
	PullRetryDelay     time.Duration
	WatcherNamespaces  string
	KubeClient         kubernetes.Interface
}

func normalizeOptions(opts Options) Options {
	if opts.ReconcileInterval <= 0 {
		opts.ReconcileInterval = defaultReconcileInterval
	}
	if opts.RepoDepth < 0 {
		opts.RepoDepth = 0
	}
	if opts.ListTimeoutSeconds <= 0 {
		opts.ListTimeoutSeconds = 15
	}
	if opts.MaxCommitsScan < 0 {
		opts.MaxCommitsScan = 0
	}
	if opts.PullRetries < 0 {
		opts.PullRetries = 0
	}
	if opts.PullRetryDelay < 0 {
		opts.PullRetryDelay = 0
	}
	return opts
}

// Matcher fires trigger events when git content changes.
type Matcher interface {
	MatchGitTrigger(ctx context.Context, triggerName, namespace string, gitMeta map[string]string) error
}

// Informer polls git repositories referenced by content triggers and fires
// events when matching commits are detected.
type Informer struct {
	mu                sync.Mutex
	testTriggerClient testtriggerclient.TestTriggerClient
	matcher           Matcher
	commits           map[string]string // key -> last seen head hash
	revisions         map[string]string // key -> last seen revision selector
	namespaces        []string
	environmentID     string
	options           Options
	kubeClient        kubernetes.Interface
}

type reconcileCache struct {
	states map[string]*reconcileState
}

type reconcileState struct {
	allRefsLoaded bool
	allRefs       []refHashPair
	allRefsErr    error

	deltas map[string]commitDelta
}

type commitDelta struct {
	paths     []string
	foundPrev bool
	err       error
}

// matchResult holds the outcome of a git commit check including metadata.
type matchResult struct {
	changed  bool
	metadata map[string]string
}

// NewInformer returns a new git content informer.
func NewInformer(
	testTriggerClient testtriggerclient.TestTriggerClient,
	matcher Matcher,
	namespace string,
	environmentID string,
	options Options,
) *Informer {
	options = normalizeOptions(options)
	return &Informer{
		testTriggerClient: testTriggerClient,
		matcher:           matcher,
		commits:           make(map[string]string),
		revisions:         make(map[string]string),
		namespaces:        resolveNamespaces(options.WatcherNamespaces, namespace),
		environmentID:     environmentID,
		options:           options,
		kubeClient:        options.KubeClient,
	}
}

func resolveNamespaces(watcherNamespaces, _ string) []string {
	namespaces := make([]string, 0)
	seen := make(map[string]struct{})
	for _, namespace := range strings.Split(watcherNamespaces, ",") {
		value := strings.TrimSpace(namespace)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		namespaces = append(namespaces, value)
	}

	if len(namespaces) > 0 {
		return namespaces
	}
	return []string{allNamespacesMarker}
}

// Reconcile periodically polls git repositories and emits trigger events.
func (i *Informer) Reconcile(ctx context.Context) {
	log.DefaultLogger.Info("git informer: starting reconciler")

	i.updateRepositories(ctx)

	ticker := time.NewTicker(i.options.ReconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Info("git informer: stopping reconciler")
			return
		case <-ticker.C:
			i.updateRepositories(ctx)
		}
	}
}

func (i *Informer) updateRepositories(ctx context.Context) {
	i.mu.Lock()
	defer i.mu.Unlock()

	testTriggerMap := make(map[string]testkube.TestTrigger)
	testTriggerListSucceeded := false
	testTriggerListedNamespaces := make(map[string]struct{})
	for _, namespace := range i.namespaces {
		testTriggerList, err := i.testTriggerClient.List(ctx, i.environmentID, testtriggerclient.ListOptions{}, namespace)
		if err != nil {
			log.DefaultLogger.Errorf("git informer: error listing test triggers in namespace %q: %v", namespace, err)
			continue
		}
		testTriggerListSucceeded = true
		testTriggerListedNamespaces[namespace] = struct{}{}
		for _, trigger := range testTriggerList {
			testTriggerMap[triggerKey(testTriggerSource, trigger.Namespace, trigger.Name)] = trigger
		}
	}
	if !testTriggerListSucceeded {
		return
	}

	active := make(map[string]struct{}, len(testTriggerMap))
	cache := newReconcileCache()
	for _, trigger := range testTriggerMap {
		if err := ctx.Err(); err != nil {
			return
		}
		if !isGitContentTrigger(trigger) {
			continue
		}
		key := triggerKey(testTriggerSource, trigger.Namespace, trigger.Name)
		active[key] = struct{}{}

		// Snapshot per-ref commit state before checking so we can restore on error.
		prevCommits := i.snapshotRefCommits(key)

		match, err := i.hasNewMatchingCommitWithCache(ctx, key, trigger, cache)
		if err != nil {
			log.DefaultLogger.Errorf("git informer: error checking trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
			i.restoreRefCommits(key, prevCommits)
			continue
		}
		if !match.changed {
			continue
		}
		if i.matcher == nil {
			i.restoreRefCommits(key, prevCommits)
			continue
		}
		if err := i.matcher.MatchGitTrigger(ctx, trigger.Name, trigger.Namespace, match.metadata); err != nil {
			log.DefaultLogger.Errorf("git informer: error matching trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
			i.restoreRefCommits(key, prevCommits)
		}
	}

	// Clean up commits for removed triggers
	for k := range i.commits {
		triggerBase := triggerKeyFromRefSubKey(k)
		if _, ok := active[triggerBase]; !ok {
			source, namespace, _, parsed := parseTriggerKey(triggerBase)
			if parsed {
				if source == testTriggerSource {
					if _, all := testTriggerListedNamespaces[allNamespacesMarker]; !all {
						if _, listed := testTriggerListedNamespaces[namespace]; !listed {
							continue
						}
					}
				}
			}
			delete(i.commits, k)
			delete(i.revisions, triggerBase)
			removeTriggerRepositories(triggerRepositoryPathFromKey(triggerBase))
		}
	}
}

func removeTriggerRepositories(basePath string) {
	if err := os.RemoveAll(basePath); err != nil {
		log.DefaultLogger.Warnf("git informer: failed removing repository path %s: %v", basePath, err)
	}
	matches, err := filepath.Glob(basePath + refDelimiter + "*")
	if err != nil {
		log.DefaultLogger.Warnf("git informer: failed listing per-ref repository paths for %s: %v", basePath, err)
		return
	}
	for _, path := range matches {
		if err := os.RemoveAll(path); err != nil {
			log.DefaultLogger.Warnf("git informer: failed removing per-ref repository path %s: %v", path, err)
		}
	}
}

func (i *Informer) restoreCommitBaseline(key, previousHash string, hadPreviousHash bool) {
	if hadPreviousHash {
		i.commits[key] = previousHash
		return
	}
	delete(i.commits, key)
}

// snapshotRefCommits captures all per-ref commit entries for a trigger key.
func (i *Informer) snapshotRefCommits(triggerKey string) map[string]string {
	snapshot := make(map[string]string)
	prefix := triggerKey + refSeparator
	for k, v := range i.commits {
		if k == triggerKey || strings.HasPrefix(k, prefix) {
			snapshot[k] = v
		}
	}
	return snapshot
}

// restoreRefCommits restores per-ref commit entries from a snapshot, removing any new keys.
func (i *Informer) restoreRefCommits(triggerKey string, snapshot map[string]string) {
	prefix := triggerKey + refSeparator
	// Remove all current keys for this trigger
	for k := range i.commits {
		if k == triggerKey || strings.HasPrefix(k, prefix) {
			delete(i.commits, k)
		}
	}
	// Restore snapshot
	for k, v := range snapshot {
		i.commits[k] = v
	}
}

// triggerKeyFromRefSubKey extracts the base trigger key from a ref sub-key.
func triggerKeyFromRefSubKey(k string) string {
	if idx := strings.Index(k, refSeparator); idx >= 0 {
		return k[:idx]
	}
	return k
}

func isGitContentTrigger(trigger testkube.TestTrigger) bool {
	return !trigger.Disabled &&
		isContentResource(trigger) &&
		isGitContentEvent(trigger.Event) &&
		trigger.ContentSelector != nil &&
		trigger.ContentSelector.Git != nil &&
		trigger.ContentSelector.Git.Uri != ""
}

func isGitContentEvent(event string) bool {
	e := strings.ToLower(event)
	return e == EventGitPush || e == EventGitTagPush
}

func isContentResource(trigger testkube.TestTrigger) bool {
	if trigger.Resource != nil && *trigger.Resource == testkube.CONTENT_TestTriggerResources {
		return true
	}

	if trigger.ResourceRef != nil && strings.EqualFold(trigger.ResourceRef.Kind, string(testkube.CONTENT_TestTriggerResources)) {
		return true
	}

	return false
}

func (i *Informer) hasNewMatchingCommitWithCache(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (matchResult, error) {
	if err := ctx.Err(); err != nil {
		return matchResult{}, err
	}

	gitConfig := trigger.ContentSelector.Git
	paths := normalizePaths(gitConfig.Paths)
	pathsIgnore := normalizePaths(gitConfig.PathsIgnore)

	if len(paths) == 0 && len(pathsIgnore) == 0 {
		return i.hasNewHeadCommitWithCache(ctx, key, trigger, cache)
	}

	// Use multi-ref tracking even with path filters. First get all matching
	// refs via remote ls-refs so that triggers with multiple branches still
	// track each one independently.
	matchingRefs, err := i.remoteAllMatchingRefsWithCache(ctx, trigger.Namespace, gitConfig, cache)
	if err != nil {
		return matchResult{}, err
	}

	// Migrate legacy plain-key entry once before iterating refs.
	legacyHash, hasLegacy := i.commits[key]
	if hasLegacy {
		delete(i.commits, key)
	}

	// Check each matching ref independently. Fire if ANY ref has a new commit
	// with matching path changes.
	for _, pair := range matchingRefs {
		refKey := refSubKey(key, pair.Ref)
		prevHash, hasPrev := i.commits[refKey]
		if !hasPrev && hasLegacy {
			prevHash = legacyHash
			hasPrev = true
			hasLegacy = false
		}
		i.commits[refKey] = pair.Hash
		if !hasPrev {
			log.DefaultLogger.Warnf(
				"git informer: initializing baseline at current HEAD for trigger %s/%s ref %s; commits pushed while informer was not running are not replayed",
				trigger.Namespace, trigger.Name, pair.Ref,
			)
			continue
		}
		if prevHash == pair.Hash {
			continue
		}

		// Ref changed – clone/open the repo for this ref and check path diffs.
		repo, repoErr := i.openOrUpdateRepositoryForRef(ctx, key, trigger, pair.Ref)
		if repoErr != nil {
			return matchResult{}, repoErr
		}

		delta := commitDelta{}
		delta.paths, delta.foundPrev, delta.err = collectChangedPathsSince(repo, pair.Hash, prevHash, i.options.MaxCommitsScan)

		if delta.err != nil {
			return matchResult{}, delta.err
		}

		matched := false
		for _, changedPath := range delta.paths {
			if pathIsIgnored(pathsIgnore, changedPath) {
				continue
			}
			if len(paths) == 0 || pathMatchesNormalized(paths, changedPath) {
				matched = true
				break
			}
		}

		if matched {
			meta := i.collectHeadMetadata(repo, pair.Hash, gitConfig, pair.Ref)
			return matchResult{changed: true, metadata: meta}, nil
		}

		if !delta.foundPrev {
			log.DefaultLogger.Warnf(
				"git informer: history boundary reached before previous commit for trigger %s/%s ref %s (repo depth/max scan limit); advancing baseline without firing",
				trigger.Namespace, trigger.Name, pair.Ref,
			)
		}
	}
	return matchResult{}, nil
}

func (i *Informer) hasNewHeadCommitWithCache(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (matchResult, error) {
	gitConfig := trigger.ContentSelector.Git
	matchingRefs, err := i.remoteAllMatchingRefsWithCache(ctx, trigger.Namespace, gitConfig, cache)
	if err != nil {
		return matchResult{}, err
	}

	// Migrate legacy plain-key entry once before iterating refs.
	legacyHash, hasLegacy := i.commits[key]
	if hasLegacy {
		delete(i.commits, key)
	}

	// Check each matching ref independently. Fire if ANY ref has a new commit.
	for _, pair := range matchingRefs {
		refKey := refSubKey(key, pair.Ref)
		prevHash, hasPrev := i.commits[refKey]
		if !hasPrev && hasLegacy {
			// Apply legacy hash as the baseline for the first ref that needs it.
			prevHash = legacyHash
			hasPrev = true
			hasLegacy = false
		}
		i.commits[refKey] = pair.Hash
		if !hasPrev {
			log.DefaultLogger.Warnf(
				"git informer: initializing baseline at current HEAD for trigger %s/%s ref %s; commits pushed while informer was not running are not replayed",
				trigger.Namespace, trigger.Name, pair.Ref,
			)
			continue
		}
		if prevHash == pair.Hash {
			continue
		}

		// Build metadata from the changed ref and load commit details.
		meta := make(map[string]string)
		meta[GitMetaKeyCommit] = pair.Hash
		if pair.Ref != "" {
			meta[GitMetaKeyRef] = pair.Ref
			if branch := branchFromRef(pair.Ref); branch != "" {
				meta[GitMetaKeyBranch] = branch
			}
			if tag := tagFromRef(pair.Ref); tag != "" {
				meta[GitMetaKeyTag] = tag
			}
		}

		// Try to load full commit metadata (message, author, timestamp).
		repo, repoErr := i.openOrUpdateRepositoryForRef(ctx, key, trigger, pair.Ref)
		if repoErr == nil && repo != nil {
			commitObj, commitErr := repo.CommitObject(plumbing.NewHash(pair.Hash))
			if commitErr == nil {
				meta[GitMetaKeyCommitMessage] = strings.TrimSpace(commitObj.Message)
				meta[GitMetaKeyAuthor] = commitObj.Author.Name
				if commitObj.Author.Email != "" {
					meta[GitMetaKeyAuthor] = commitObj.Author.Name + " <" + commitObj.Author.Email + ">"
				}
				meta[GitMetaKeyCommitTimestamp] = commitObj.Author.When.UTC().Format(time.RFC3339)
			}
		}
		return matchResult{changed: true, metadata: meta}, nil
	}
	return matchResult{}, nil
}

// refSubKey returns a composite key for tracking per-ref state within a trigger.
func refSubKey(triggerKey, ref string) string {
	if ref == "" {
		return triggerKey
	}
	return triggerKey + refSeparator + ref
}

func (i *Informer) remoteAllMatchingRefsWithCache(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit, cache *reconcileCache) ([]refHashPair, error) {
	if cache == nil {
		return i.remoteAllMatchingRefs(ctx, namespace, gitConfig)
	}

	state := cache.stateFor(namespace, gitConfig)
	if state.allRefsLoaded {
		return state.allRefs, state.allRefsErr
	}

	state.allRefs, state.allRefsErr = i.remoteAllMatchingRefs(ctx, namespace, gitConfig)
	state.allRefsLoaded = true
	return state.allRefs, state.allRefsErr
}

func (i *Informer) remoteAllMatchingRefs(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit) ([]refHashPair, error) {
	clientOptions, err := i.authClientOptions(ctx, namespace, gitConfig)
	if err != nil {
		return nil, err
	}
	return remoteAllMatchingRefsWithClientOptions(gitConfig, i.options, clientOptions)
}

func normalizeRevision(revision string) string {
	return strings.TrimSpace(revision)
}

// openOrUpdateRepositoryForRef clones or pulls the repository for a specific ref.
// It uses a per-ref subdirectory to avoid conflicts when multiple refs are tracked.
func (i *Informer) openOrUpdateRepositoryForRef(ctx context.Context, key string, trigger testkube.TestTrigger, ref string) (*git.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Use an encoded ref suffix to avoid collisions like refs/heads/a-b vs refs/heads/a/b.
	refSuffix := refDirectorySuffix(ref)
	repoDir := triggerRepositoryPathFromKey(key) + refDelimiter + refSuffix
	gitConfig := trigger.ContentSelector.Git
	clientOptions, err := i.authClientOptions(ctx, trigger.Namespace, gitConfig)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(repoDir)
	if err == nil {
		if !repositoryOriginMatches(repo, gitConfig.Uri) {
			log.DefaultLogger.Warnf(
				"git informer: origin URL changed for %s/%s ref %s, recreating local clone",
				trigger.Namespace, trigger.Name, ref,
			)
		} else {
			worktree, wtErr := repo.Worktree()
			if wtErr == nil {
				pullOpts, poErr := pullOptionsForRefWithClientOptions(i.options, ref, clientOptions)
				if poErr != nil {
					return nil, poErr
				}
				pullErr := worktree.Pull(pullOpts)
				if pullErr == nil || errors.Is(pullErr, git.NoErrAlreadyUpToDate) {
					return repo, nil
				}
				log.DefaultLogger.Warnf("git informer: pull failed for %s/%s ref %s, recreating local clone: %v",
					trigger.Namespace, trigger.Name, ref, pullErr)
			}
		}
	}

	_ = os.RemoveAll(repoDir)
	parentDir := filepath.Dir(repoDir)
	if err = os.MkdirAll(parentDir, 0o700); err != nil {
		return nil, err
	}

	cloneOpts, err := cloneOptionsForRefWithClientOptions(gitConfig, i.options, ref, clientOptions)
	if err != nil {
		return nil, err
	}
	repo, err = git.PlainClone(repoDir, cloneOpts)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// effectiveRefs derives git references to watch from Branches and Tags fields.
// If both are empty, returns nil (meaning watch default HEAD).
func effectiveRefs(gitConfig *testkube.TestTriggerContentGit) []string {
	var refs []string
	for _, b := range gitConfig.Branches {
		b = strings.TrimSpace(b)
		if b != "" && !strings.Contains(b, "*") && !strings.Contains(b, "?") {
			refs = append(refs, "refs/heads/"+b)
		}
	}
	for _, t := range gitConfig.Tags {
		t = strings.TrimSpace(t)
		if t != "" && !strings.Contains(t, "*") && !strings.Contains(t, "?") {
			refs = append(refs, "refs/tags/"+t)
		}
	}
	return refs
}

// refHashPair holds a single remote reference and its hash.
type refHashPair struct {
	Hash string
	Ref  string
}

// remoteAllMatchingRefsWithClientOptions returns ALL remote refs that match the
// configured branch/tag patterns and are not excluded by ignore filters.
// When no branch/tag filters are set, all branches are returned.
func remoteAllMatchingRefsWithClientOptions(gitConfig *testkube.TestTriggerContentGit, options Options, clientOptions []client.Option) ([]refHashPair, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitConfig.Uri},
	})
	refs, err := remote.List(&git.ListOptions{
		ClientOptions: clientOptions,
		Timeout:       options.ListTimeoutSeconds,
	})
	if err != nil {
		return nil, err
	}

	hasBranchFilters := len(gitConfig.Branches) > 0
	hasTagFilters := len(gitConfig.Tags) > 0
	hasBranchIgnore := len(gitConfig.BranchesIgnore) > 0
	hasTagIgnore := len(gitConfig.TagsIgnore) > 0

	if hasBranchFilters || hasTagFilters || hasBranchIgnore || hasTagIgnore {
		var results []refHashPair
		for _, r := range refs {
			refName := string(r.Name())
			if branch := branchFromRef(refName); branch != "" {
				// When branches is empty but branchesIgnore is set, match all branches minus ignored.
				matchesBranch := hasBranchFilters && nameMatchesPatterns(branch, gitConfig.Branches)
				matchesAllBranches := !hasBranchFilters && !hasTagFilters && hasBranchIgnore
				if (matchesBranch || matchesAllBranches) && !nameMatchesAny(branch, gitConfig.BranchesIgnore) {
					results = append(results, refHashPair{Hash: r.Hash().String(), Ref: refName})
				}
			}
			if tag := tagFromRef(refName); tag != "" {
				matchesTag := hasTagFilters && nameMatchesPatterns(tag, gitConfig.Tags)
				matchesAllTags := !hasBranchFilters && !hasTagFilters && hasTagIgnore
				if (matchesTag || matchesAllTags) && !nameMatchesAny(tag, gitConfig.TagsIgnore) {
					results = append(results, refHashPair{Hash: r.Hash().String(), Ref: refName})
				}
			}
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("no matching reference found for branches=%v branchesIgnore=%v tags=%v tagsIgnore=%v", gitConfig.Branches, gitConfig.BranchesIgnore, gitConfig.Tags, gitConfig.TagsIgnore)
		}
		return results, nil
	}

	// No specific branches/tags/ignore filters: watch all branches
	var results []refHashPair
	for _, r := range refs {
		if r.Name().IsBranch() {
			results = append(results, refHashPair{Hash: r.Hash().String(), Ref: string(r.Name())})
		}
	}
	if len(results) > 0 {
		return results, nil
	}

	// Fallback to remote HEAD when branch refs are unavailable.
	for _, r := range refs {
		if r.Name() == plumbing.HEAD {
			return []refHashPair{{Hash: r.Hash().String(), Ref: string(r.Name())}}, nil
		}
	}

	return nil, errors.New("unable to determine remote HEAD")
}

func cloneOptions(gitConfig *testkube.TestTriggerContentGit, options Options) (*git.CloneOptions, error) {
	references := effectiveRefs(gitConfig)
	if len(references) == 0 {
		return cloneOptionsForRef(gitConfig, options, "")
	}
	return cloneOptionsForRef(gitConfig, options, references[0])
}

func cloneOptionsForRef(gitConfig *testkube.TestTriggerContentGit, options Options, reference string) (*git.CloneOptions, error) {
	clientOptions, err := authClientOptions(gitConfig)
	if err != nil {
		return nil, err
	}
	return cloneOptionsForRefWithClientOptions(gitConfig, options, reference, clientOptions)
}

func cloneOptionsForRefWithClientOptions(gitConfig *testkube.TestTriggerContentGit, options Options, reference string, clientOptions []client.Option) (*git.CloneOptions, error) {
	cloneOpts := &git.CloneOptions{
		URL:           gitConfig.Uri,
		SingleBranch:  true,
		ClientOptions: clientOptions,
		Depth:         options.RepoDepth,
	}
	if reference != "" && !isCommitSHA(reference) {
		cloneOpts.ReferenceName = plumbing.ReferenceName(reference)
	}

	return cloneOpts, nil
}

func pullOptions(gitConfig *testkube.TestTriggerContentGit, options Options) (*git.PullOptions, error) {
	references := effectiveRefs(gitConfig)
	if len(references) == 0 {
		return pullOptionsForRef(gitConfig, options, "")
	}
	return pullOptionsForRef(gitConfig, options, references[0])
}

func pullOptionsForRef(gitConfig *testkube.TestTriggerContentGit, options Options, reference string) (*git.PullOptions, error) {
	clientOptions, err := authClientOptions(gitConfig)
	if err != nil {
		return nil, err
	}
	return pullOptionsForRefWithClientOptions(options, reference, clientOptions)
}

func pullOptionsForRefWithClientOptions(options Options, reference string, clientOptions []client.Option) (*git.PullOptions, error) {
	pullOpts := &git.PullOptions{
		RemoteName:    "origin",
		SingleBranch:  true,
		Force:         true,
		ClientOptions: clientOptions,
		Depth:         options.RepoDepth,
	}
	if reference != "" && !isCommitSHA(reference) {
		pullOpts.ReferenceName = plumbing.ReferenceName(reference)
	}

	return pullOpts, nil
}

func authClientOptions(gitConfig *testkube.TestTriggerContentGit) ([]client.Option, error) {
	return authClientOptionsWithResolver(gitConfig, resolveCredentialValue)
}

func (i *Informer) authClientOptions(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit) ([]client.Option, error) {
	return authClientOptionsWithResolver(gitConfig, func(value string, source *testkube.EnvVarSource) string {
		return i.resolveCredentialValue(ctx, value, namespace, source)
	})
}

func authClientOptionsWithResolver(
	gitConfig *testkube.TestTriggerContentGit,
	resolver func(value string, source *testkube.EnvVarSource) string,
) ([]client.Option, error) {
	if err := validateCredentialSource("usernameFrom", gitConfig.Username, gitConfig.UsernameFrom); err != nil {
		return nil, err
	}
	if err := validateCredentialSource("tokenFrom", gitConfig.Token, gitConfig.TokenFrom); err != nil {
		return nil, err
	}
	if err := validateCredentialSource("sshKeyFrom", gitConfig.SshKey, gitConfig.SshKeyFrom); err != nil {
		return nil, err
	}

	username := resolver(gitConfig.Username, gitConfig.UsernameFrom)
	token := resolver(gitConfig.Token, gitConfig.TokenFrom)
	sshKey := resolver(gitConfig.SshKey, gitConfig.SshKeyFrom)

	authType := strings.ToLower(gitConfig.AuthType)

	opts := make([]client.Option, 0, 1)
	switch {
	case sshKey != "":
		user := username
		if user == "" {
			user = defaultGitUsername
		}
		// Passphrase-protected keys are not supported by TestTriggerContentGit.
		publicKeys, err := ssh.NewPublicKeys(user, []byte(sshKey), "")
		if err != nil {
			return nil, err
		}
		hostKeyCallback, err := ssh.NewKnownHostsCallback()
		if err != nil {
			return nil, fmt.Errorf("ssh auth requires known_hosts-based host key verification: %w", err)
		}
		publicKeys.HostKeyCallback = hostKeyCallback
		opts = append(opts, client.WithSSHAuth(publicKeys))
	case token != "" && authType == string(testkube.HEADER_ContentGitAuthType):
		opts = append(opts, client.WithHTTPAuth(&http.TokenAuth{Token: token}))
	case token != "" || username != "":
		if username == "" {
			username = defaultGitUsername
		}
		opts = append(opts, client.WithHTTPAuth(&http.BasicAuth{
			Username: username,
			Password: token,
		}))
	}

	return opts, nil
}

func validateCredentialSource(fieldName, value string, source *testkube.EnvVarSource) error {
	if strings.TrimSpace(value) != "" || source == nil {
		return nil
	}

	if source.FieldRef != nil {
		return fmt.Errorf("unsupported %s source: fieldRef is not supported for git credentials, use secretKeyRef or configMapKeyRef", fieldName)
	}

	if source.ResourceFieldRef != nil {
		return fmt.Errorf("unsupported %s source: resourceFieldRef is not supported for git credentials, use secretKeyRef or configMapKeyRef", fieldName)
	}

	return nil
}

func (i *Informer) resolveCredentialValue(ctx context.Context, value, namespace string, source *testkube.EnvVarSource) string {
	if strings.TrimSpace(value) != "" || source == nil {
		return value
	}
	if i.kubeClient == nil {
		return resolveCredentialValue(value, source)
	}
	hasRequiredRef := false
	if source.SecretKeyRef != nil {
		hasRequiredRef = hasRequiredRef || isRequiredRef(source.SecretKeyRef.Optional)
		if resolved, ok := i.resolveSecretKeyRefValue(ctx, namespace, source.SecretKeyRef); ok {
			return resolved
		}
	}
	if source.ConfigMapKeyRef != nil {
		hasRequiredRef = hasRequiredRef || isRequiredRef(source.ConfigMapKeyRef.Optional)
		if resolved, ok := i.resolveConfigMapKeyRefValue(ctx, namespace, source.ConfigMapKeyRef); ok {
			return resolved
		}
	}
	if hasRequiredRef {
		return ""
	}
	return resolveCredentialValue(value, source)
}

func isRequiredRef(optional *bool) bool {
	return optional == nil || !*optional
}

func (i *Informer) resolveSecretKeyRefValue(ctx context.Context, namespace string, ref *testkube.EnvVarSourceSecretKeyRef) (string, bool) {
	if ref == nil || ref.Name == "" || ref.Key == "" {
		return "", false
	}
	secret, err := i.kubeClient.CoreV1().Secrets(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) || ref.Optional == nil || !*ref.Optional {
			log.DefaultLogger.Warnf("git informer: failed to read secret %s/%s key %s: %v", namespace, ref.Name, ref.Key, err)
		}
		return "", false
	}
	if value, ok := secret.Data[ref.Key]; ok {
		return string(value), true
	}
	if ref.Optional == nil || !*ref.Optional {
		log.DefaultLogger.Warnf("git informer: key %s not found in secret %s/%s", ref.Key, namespace, ref.Name)
	}
	return "", false
}

func (i *Informer) resolveConfigMapKeyRefValue(ctx context.Context, namespace string, ref *testkube.EnvVarSourceConfigMapKeyRef) (string, bool) {
	if ref == nil || ref.Name == "" || ref.Key == "" {
		return "", false
	}
	configMap, err := i.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) || ref.Optional == nil || !*ref.Optional {
			log.DefaultLogger.Warnf("git informer: failed to read configmap %s/%s key %s: %v", namespace, ref.Name, ref.Key, err)
		}
		return "", false
	}
	if value, ok := configMap.Data[ref.Key]; ok {
		return value, true
	}
	if value, ok := configMap.BinaryData[ref.Key]; ok {
		return string(value), true
	}
	if ref.Optional == nil || !*ref.Optional {
		log.DefaultLogger.Warnf("git informer: key %s not found in configmap %s/%s", ref.Key, namespace, ref.Name)
	}
	return "", false
}

func resolveCredentialValue(value string, source *testkube.EnvVarSource) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	if source == nil {
		return ""
	}
	if source.SecretKeyRef != nil {
		return resolveCredentialValueFromRef(source.SecretKeyRef.Name, source.SecretKeyRef.Key)
	}
	if source.ConfigMapKeyRef != nil {
		return resolveCredentialValueFromRef(source.ConfigMapKeyRef.Name, source.ConfigMapKeyRef.Key)
	}
	return ""
}

// resolveCredentialValueFromRef resolves credentials from process env only.
// Resolution order is: key -> sanitized NAME_KEY -> name.
func resolveCredentialValueFromRef(name, key string) string {
	for _, envVarName := range []string{
		key,
		normalizeSecretOrConfigMapEnvVarName(name, key),
		name,
	} {
		if envVarName == "" {
			continue
		}
		if value := os.Getenv(envVarName); value != "" {
			return value
		}
	}
	return ""
}

func collectChangedPathsSince(repo *git.Repository, headHash, prevHash string, maxCommitsScan int) ([]string, bool, error) {
	head := plumbing.NewHash(headHash)
	iter, err := repo.Log(&git.LogOptions{From: head})
	if err != nil {
		return nil, false, err
	}
	defer iter.Close()

	paths := make([]string, 0)
	seen := make(map[string]struct{})
	foundPrev := false
	scanned := 0

	err = iter.ForEach(func(c *object.Commit) error {
		if maxCommitsScan > 0 && scanned >= maxCommitsScan {
			return storer.ErrStop
		}
		scanned++

		if c.Hash.String() == prevHash {
			foundPrev = true
			return storer.ErrStop
		}

		stats, err := c.Stats()
		if err != nil {
			return err
		}
		for _, stat := range stats {
			if _, ok := seen[stat.Name]; ok {
				continue
			}
			seen[stat.Name] = struct{}{}
			paths = append(paths, stat.Name)
		}
		return nil
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		return nil, false, err
	}

	return paths, foundPrev, nil
}

func newReconcileCache() *reconcileCache {
	return &reconcileCache{states: make(map[string]*reconcileState)}
}

func (c *reconcileCache) stateFor(namespace string, gitConfig *testkube.TestTriggerContentGit) *reconcileState {
	key := gitConfigCacheKey(namespace, gitConfig)
	if state, ok := c.states[key]; ok {
		return state
	}
	state := &reconcileState{deltas: make(map[string]commitDelta)}
	c.states[key] = state
	return state
}

func gitConfigCacheKey(namespace string, gitConfig *testkube.TestTriggerContentGit) string {
	if gitConfig == nil {
		return namespace
	}

	authType := strings.ToLower(gitConfig.AuthType)

	return strings.Join([]string{
		namespace,
		gitConfig.Uri,
		effectiveRefsKey(gitConfig),
		effectiveIgnoreRefsKey(gitConfig),
		authType,
		gitConfig.Username,
		gitConfig.Token,
		gitConfig.SshKey,
		envVarSourceCacheKey(gitConfig.UsernameFrom),
		envVarSourceCacheKey(gitConfig.TokenFrom),
		envVarSourceCacheKey(gitConfig.SshKeyFrom),
	}, "|")
}

func envVarSourceCacheKey(source *testkube.EnvVarSource) string {
	if source == nil {
		return ""
	}

	return strings.Join([]string{
		fieldRefCacheKey(source.FieldRef),
		resourceFieldRefCacheKey(source.ResourceFieldRef),
		configMapKeyRefCacheKey(source.ConfigMapKeyRef),
		secretKeyRefCacheKey(source.SecretKeyRef),
	}, ";")
}

func fieldRefCacheKey(ref *testkube.FieldRef) string {
	if ref == nil {
		return ""
	}
	return strings.Join([]string{ref.ApiVersion, ref.FieldPath}, ":")
}

func resourceFieldRefCacheKey(ref *testkube.ResourceFieldRef) string {
	if ref == nil {
		return ""
	}
	return strings.Join([]string{ref.ContainerName, ref.Resource, ref.Divisor}, ":")
}

func configMapKeyRefCacheKey(ref *testkube.EnvVarSourceConfigMapKeyRef) string {
	if ref == nil {
		return ""
	}
	return strings.Join([]string{ref.Name, ref.Key, boolPtrString(ref.Optional)}, ":")
}

func secretKeyRefCacheKey(ref *testkube.EnvVarSourceSecretKeyRef) string {
	if ref == nil {
		return ""
	}
	return strings.Join([]string{ref.Name, ref.Key, boolPtrString(ref.Optional)}, ":")
}

func boolPtrString(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}

func normalizeSecretOrConfigMapEnvVarName(name, key string) string {
	if name == "" || key == "" {
		return ""
	}
	candidate := strings.ToUpper(name + "_" + key)
	return envVarNameSanitizer.ReplaceAllString(candidate, "_")
}

// refDirectorySuffix produces a filesystem-safe, collision-resistant token for a git ref.
// Raw URL-safe base64 keeps path-safe characters and avoids padding while preserving uniqueness.
func refDirectorySuffix(ref string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

func normalizeRefs(revision string) []string {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		return nil
	}
	if isCommitSHA(revision) {
		return []string{revision}
	}
	if strings.HasPrefix(revision, "refs/") {
		return []string{revision}
	}
	return []string{"refs/heads/" + revision, "refs/tags/" + revision}
}

func isCommitSHA(revision string) bool {
	return gitCommitSHAPattern.MatchString(strings.TrimSpace(revision))
}

func shouldAdvanceBaselineOnScanError(err error) bool {
	return errors.Is(err, plumbing.ErrObjectNotFound)
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.Trim(strings.TrimSpace(p), "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func pathMatches(paths []string, file string) bool {
	return pathMatchesNormalized(normalizePaths(paths), file)
}

func pathMatchesNormalized(paths []string, file string) bool {
	for _, p := range paths {
		if matchGlob(p, file) {
			return true
		}
	}
	return false
}

func triggerKey(source, namespace, name string) string {
	return source + ":" + namespace + "/" + name
}

func triggerRepositoryPath(source, namespace, name string) string {
	return filepath.Join(os.TempDir(), "testkube-git-trigger", source, namespace, name)
}

func triggerRepositoryPathFromKey(key string) string {
	source, namespace, name, ok := parseTriggerKey(key)
	if !ok {
		return filepath.Join(os.TempDir(), "testkube-git-trigger", key)
	}
	return triggerRepositoryPath(source, namespace, name)
}

func parseTriggerKey(key string) (source, namespace, name string, ok bool) {
	sourceAndRest := strings.SplitN(key, ":", 2)
	if len(sourceAndRest) != 2 {
		return "", "", "", false
	}
	namespaceAndName := strings.SplitN(sourceAndRest[1], "/", 2)
	if len(namespaceAndName) != 2 {
		return "", "", "", false
	}
	return sourceAndRest[0], namespaceAndName[0], namespaceAndName[1], true
}

func repositoryOriginMatches(repo *git.Repository, expectedURL string) bool {
	remote, err := repo.Remote("origin")
	if err != nil {
		return false
	}

	expectedURL = strings.TrimSpace(expectedURL)
	if expectedURL == "" {
		return false
	}

	for _, currentURL := range remote.Config().URLs {
		if strings.TrimSpace(currentURL) == expectedURL {
			return true
		}
	}

	return false
}

// effectiveRefsKey produces a stable string from Branches+Tags for use as a cache/revision key.
func effectiveRefsKey(gitConfig *testkube.TestTriggerContentGit) string {
	parts := make([]string, 0, len(gitConfig.Branches)+len(gitConfig.Tags))
	for _, b := range gitConfig.Branches {
		parts = append(parts, "b:"+strings.TrimSpace(b))
	}
	for _, t := range gitConfig.Tags {
		parts = append(parts, "t:"+strings.TrimSpace(t))
	}
	return strings.Join(parts, ",")
}

// effectiveIgnoreRefsKey produces a stable string from BranchesIgnore+TagsIgnore for cache keying.
func effectiveIgnoreRefsKey(gitConfig *testkube.TestTriggerContentGit) string {
	parts := make([]string, 0, len(gitConfig.BranchesIgnore)+len(gitConfig.TagsIgnore))
	for _, b := range gitConfig.BranchesIgnore {
		parts = append(parts, "bi:"+strings.TrimSpace(b))
	}
	for _, t := range gitConfig.TagsIgnore {
		parts = append(parts, "ti:"+strings.TrimSpace(t))
	}
	return strings.Join(parts, ",")
}

// pathIsIgnored returns true if the file matches any of the ignore patterns.
func pathIsIgnored(ignorePatterns []string, file string) bool {
	if len(ignorePatterns) == 0 {
		return false
	}
	for _, p := range ignorePatterns {
		if matchGlob(p, file) {
			return true
		}
	}
	return false
}

// branchFromRef extracts the branch name from a full ref like "refs/heads/main".
func branchFromRef(ref string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ""
}

// tagFromRef extracts the tag name from a full ref like "refs/tags/v1.0.0".
func tagFromRef(ref string) string {
	const prefix = "refs/tags/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ""
}

// matchGlob performs glob-style matching supporting * and ** patterns.
// It matches file paths against patterns like "src/**", "*.md", "docs/*".
// Malformed patterns are treated as non-matching (filepath.Match returns ErrBadPattern).
func matchGlob(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		log.DefaultLogger.Debugf("git informer: malformed glob pattern %q: %v", pattern, err)
		return false
	}
	if matched {
		return true
	}
	// Support ** for recursive directory matching
	if strings.Contains(pattern, "**") {
		// Globstar (**) matching across path segments: ** matches zero or more directories.
		pattern = filepath.ToSlash(strings.TrimSuffix(pattern, "/"))
		name = filepath.ToSlash(strings.TrimSuffix(name, "/"))

		pSegs := strings.Split(pattern, "/")
		nSegs := strings.Split(name, "/")

		type state struct{ i, j int }
		memo := map[state]bool{}
		var match func(i, j int) bool
		match = func(i, j int) bool {
			s := state{i, j}
			if v, ok := memo[s]; ok {
				return v
			}

			var res bool
			switch {
			case i == len(pSegs):
				res = j == len(nSegs)
			case pSegs[i] == "**":
				res = match(i+1, j) || (j < len(nSegs) && match(i, j+1))
			case j < len(nSegs):
				ok, err := filepath.Match(pSegs[i], nSegs[j])
				res = err == nil && ok && match(i+1, j+1)
			default:
				res = false
			}

			memo[s] = res
			return res
		}
		return match(0, 0)
	}

	// Also try prefix match for directory patterns
	normalizedPattern := strings.TrimSuffix(pattern, "/")
	if name == normalizedPattern || strings.HasPrefix(name, normalizedPattern+"/") {
		return true
	}
	return false
}

// nameMatchesPatterns checks if a name (branch or tag) matches any of the given glob patterns.
// Returns true if patterns is empty (matches all).
func nameMatchesPatterns(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	return nameMatchesAny(name, patterns)
}

// nameMatchesAny checks if a name matches any of the given glob patterns.
// Returns false if patterns is empty.
func nameMatchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		matched, err := filepath.Match(p, name)
		if err != nil {
			log.DefaultLogger.Debugf("git informer: malformed glob pattern %q: %v", p, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// collectHeadMetadata extracts metadata from the HEAD commit of a repository.
func (i *Informer) collectHeadMetadata(repo *git.Repository, headHash string, gitConfig *testkube.TestTriggerContentGit, preferredRef string) map[string]string {
	meta := make(map[string]string)
	meta[GitMetaKeyCommit] = headHash

	// Determine branch/tag from the repo's references matching the config patterns.
	// Only include tag metadata when tags are explicitly watched; otherwise a branch
	// push to a commit that is also tagged could be misclassified as git-tag-push.
	hasBranchFilters := len(gitConfig.Branches) > 0
	hasTagFilters := len(gitConfig.Tags) > 0
	hasTagIgnore := len(gitConfig.TagsIgnore) > 0
	watchTags := hasTagFilters || (!hasBranchFilters && !hasTagFilters && hasTagIgnore)

	if preferredRef != "" {
		meta[GitMetaKeyRef] = preferredRef
		if branch := branchFromRef(preferredRef); branch != "" {
			meta[GitMetaKeyBranch] = branch
			delete(meta, GitMetaKeyTag)
		}
		if watchTags {
			if tag := tagFromRef(preferredRef); tag != "" {
				meta[GitMetaKeyTag] = tag
				delete(meta, GitMetaKeyBranch)
			}
		}
	}

	if repo != nil {
		refIter, err := repo.References()
		if err == nil {
			_ = refIter.ForEach(func(ref *plumbing.Reference) error {
				if ref.Hash().String() != headHash {
					return nil
				}
				refName := string(ref.Name())
				if branch := branchFromRef(refName); branch != "" {
					if nameMatchesPatterns(branch, gitConfig.Branches) && !nameMatchesAny(branch, gitConfig.BranchesIgnore) {
						meta[GitMetaKeyRef] = refName
						meta[GitMetaKeyBranch] = branch
					}
				}
				if watchTags {
					if tag := tagFromRef(refName); tag != "" {
						if nameMatchesPatterns(tag, gitConfig.Tags) && !nameMatchesAny(tag, gitConfig.TagsIgnore) {
							meta[GitMetaKeyRef] = refName
							meta[GitMetaKeyTag] = tag
							delete(meta, GitMetaKeyBranch)
						}
					}
				}
				return nil
			})
		}
	}

	// Fallback: if no ref found from repo references, try literal refs
	if _, hasRef := meta[GitMetaKeyRef]; !hasRef {
		refs := effectiveRefs(gitConfig)
		if len(refs) > 0 {
			meta[GitMetaKeyRef] = refs[0]
			if branch := branchFromRef(refs[0]); branch != "" {
				meta[GitMetaKeyBranch] = branch
			}
			if tag := tagFromRef(refs[0]); tag != "" {
				meta[GitMetaKeyTag] = tag
			}
		}
	}

	if repo == nil {
		return meta
	}

	// Get commit details
	head := plumbing.NewHash(headHash)
	commit, err := repo.CommitObject(head)
	if err != nil {
		return meta
	}

	meta[GitMetaKeyCommitMessage] = strings.TrimSpace(commit.Message)
	meta[GitMetaKeyAuthor] = commit.Author.Name
	if commit.Author.Email != "" {
		meta[GitMetaKeyAuthor] = commit.Author.Name + " <" + commit.Author.Email + ">"
	}
	meta[GitMetaKeyCommitTimestamp] = commit.Author.When.UTC().Format(time.RFC3339)

	return meta
}
