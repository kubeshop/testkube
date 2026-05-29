package informer

import (
	"context"
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

const defaultReconcileInterval = time.Minute
const defaultGitUsername = "git"
const testTriggerSource = "v1"
const allNamespacesMarker = "*"

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
	remoteHeadLoaded bool
	remoteHeadHash   string
	remoteHeadRef    string
	remoteHeadErr    error

	repoLoaded bool
	repo       *git.Repository
	repoHead   string
	repoErr    error

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
		prevHash, hadPrevHash := i.commits[key]

		changed, err := i.hasNewMatchingCommitWithCache(ctx, key, trigger, cache)
		if err != nil {
			log.DefaultLogger.Errorf("git informer: error checking trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
			continue
		}
		if !changed.changed {
			continue
		}
		if i.matcher == nil {
			i.restoreCommitBaseline(key, prevHash, hadPrevHash)
			continue
		}
		if err := i.matcher.MatchGitTrigger(ctx, trigger.Name, trigger.Namespace, changed.metadata); err != nil {
			log.DefaultLogger.Errorf("git informer: error matching trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
			i.restoreCommitBaseline(key, prevHash, hadPrevHash)
		}
	}

	// Clean up commits for removed triggers
	for k := range i.commits {
		if _, ok := active[k]; !ok {
			source, namespace, _, parsed := parseTriggerKey(k)
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
			delete(i.revisions, k)
			_ = os.RemoveAll(triggerRepositoryPathFromKey(k))
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

func isGitContentTrigger(trigger testkube.TestTrigger) bool {
	return !trigger.Disabled &&
		isContentResource(trigger) &&
		isModifiedGitContentEvent(trigger.Event) &&
		trigger.ContentSelector != nil &&
		trigger.ContentSelector.Git != nil &&
		trigger.ContentSelector.Git.Uri != ""
}

func isModifiedGitContentEvent(event string) bool {
	return strings.EqualFold(event, "modified")
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

func (i *Informer) hasNewMatchingCommit(ctx context.Context, key string, trigger testkube.TestTrigger) (matchResult, error) {
	return i.hasNewMatchingCommitWithCache(ctx, key, trigger, nil)
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

	repo, headHash, state, err := i.openOrUpdateRepositoryWithCache(ctx, key, trigger, cache)
	if err != nil {
		return matchResult{}, err
	}
	prevHash, hasPrev := i.commits[key]

	if !hasPrev {
		i.commits[key] = headHash
		logBaselineInitialization(trigger.Namespace, trigger.Name)
		return matchResult{}, nil
	}
	if prevHash == headHash {
		return matchResult{}, nil
	}

	delta := commitDelta{}
	if state != nil {
		if existing, ok := state.deltas[prevHash]; ok {
			delta = existing
		} else {
			delta.paths, delta.foundPrev, delta.err = collectChangedPathsSince(repo, headHash, prevHash, i.options.MaxCommitsScan)
			state.deltas[prevHash] = delta
		}
	} else {
		delta.paths, delta.foundPrev, delta.err = collectChangedPathsSince(repo, headHash, prevHash, i.options.MaxCommitsScan)
	}

	if delta.err != nil {
		err = delta.err
		if shouldAdvanceBaselineOnScanError(err) {
			i.commits[key] = headHash
		}
		return matchResult{}, err
	}

	i.commits[key] = headHash

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

	if !matched {
		if !delta.foundPrev {
			log.DefaultLogger.Warnf(
				"git informer: history boundary reached before previous commit for trigger %s/%s (repo depth/max scan limit); advancing baseline without firing",
				trigger.Namespace,
				trigger.Name,
			)
		}
		return matchResult{}, nil
	}

	// Collect git metadata from HEAD commit
	meta := i.collectHeadMetadata(repo, headHash, gitConfig)
	return matchResult{changed: true, metadata: meta}, nil
}

func (i *Informer) hasNewHeadCommitWithCache(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (matchResult, error) {
	gitConfig := trigger.ContentSelector.Git
	headHash, ref, err := i.remoteHeadHashAndRefWithCache(ctx, trigger.Namespace, gitConfig, cache)
	if err != nil {
		return matchResult{}, err
	}

	prevHash, hasPrev := i.commits[key]
	i.commits[key] = headHash
	if !hasPrev {
		logBaselineInitialization(trigger.Namespace, trigger.Name)
	}

	if !hasPrev || prevHash == headHash {
		return matchResult{}, nil
	}

	// Build metadata from remote HEAD info
	meta := make(map[string]string)
	meta[GitMetaKeyCommit] = headHash
	if ref != "" {
		meta[GitMetaKeyRef] = ref
		if branch := branchFromRef(ref); branch != "" {
			meta[GitMetaKeyBranch] = branch
		}
		if tag := tagFromRef(ref); tag != "" {
			meta[GitMetaKeyTag] = tag
		}
	}

	return matchResult{changed: true, metadata: meta}, nil
}

func logBaselineInitialization(namespace, triggerName string) {
	log.DefaultLogger.Warnf(
		"git informer: initializing baseline at current HEAD for trigger %s/%s; commits pushed while informer was not running are not replayed",
		namespace,
		triggerName,
	)
}

func (i *Informer) remoteHeadHashWithCache(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit, cache *reconcileCache) (string, error) {
	if cache == nil {
		return i.remoteHeadHash(ctx, namespace, gitConfig)
	}

	state := cache.stateFor(namespace, gitConfig)
	if state.remoteHeadLoaded {
		return state.remoteHeadHash, state.remoteHeadErr
	}

	state.remoteHeadHash, state.remoteHeadErr = i.remoteHeadHash(ctx, namespace, gitConfig)
	state.remoteHeadLoaded = true
	return state.remoteHeadHash, state.remoteHeadErr
}

func (i *Informer) remoteHeadHashAndRefWithCache(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit, cache *reconcileCache) (string, string, error) {
	if cache == nil {
		return i.remoteHeadHashAndRef(ctx, namespace, gitConfig)
	}

	state := cache.stateFor(namespace, gitConfig)
	if state.remoteHeadLoaded {
		return state.remoteHeadHash, state.remoteHeadRef, state.remoteHeadErr
	}

	state.remoteHeadHash, state.remoteHeadRef, state.remoteHeadErr = i.remoteHeadHashAndRef(ctx, namespace, gitConfig)
	state.remoteHeadLoaded = true
	return state.remoteHeadHash, state.remoteHeadRef, state.remoteHeadErr
}

func (i *Informer) openOrUpdateRepository(ctx context.Context, key string, trigger testkube.TestTrigger) (*git.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repoDir := triggerRepositoryPathFromKey(key)
	gitConfig := trigger.ContentSelector.Git
	revision := effectiveRefsKey(gitConfig)
	previousRevision, hasPreviousRevision := i.revisions[key]
	revisionChanged := hasPreviousRevision && previousRevision != revision
	references := effectiveRefs(gitConfig)
	if len(references) == 0 {
		references = []string{""}
	}
	clientOptions, err := i.authClientOptions(ctx, trigger.Namespace, gitConfig)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(repoDir)
	if err == nil {
		if revisionChanged {
			log.DefaultLogger.Warnf(
				"git informer: revision changed for %s/%s from %q to %q, recreating local clone",
				trigger.Namespace,
				trigger.Name,
				previousRevision,
				revision,
			)
		} else if !repositoryOriginMatches(repo, gitConfig.Uri) {
			log.DefaultLogger.Warnf(
				"git informer: origin URL changed for %s/%s, recreating local clone",
				trigger.Namespace,
				trigger.Name,
			)
		} else {
			worktree, wtErr := repo.Worktree()
			if wtErr == nil {
				var pullErr error
				for _, reference := range references {
					if err := ctx.Err(); err != nil {
						return nil, err
					}

					pullOpts, err := pullOptionsForRefWithClientOptions(gitConfig, i.options, reference, clientOptions)
					if err != nil {
						return nil, err
					}
					for attempt := 0; attempt <= i.options.PullRetries; attempt++ {
						if err := ctx.Err(); err != nil {
							return nil, err
						}

						pullErr = worktree.Pull(pullOpts)
						if pullErr == nil || errors.Is(pullErr, git.NoErrAlreadyUpToDate) {
							i.revisions[key] = revision
							return repo, nil
						}
						if attempt < i.options.PullRetries && i.options.PullRetryDelay > 0 {
							if err := sleepWithContext(ctx, i.options.PullRetryDelay); err != nil {
								return nil, err
							}
						}
					}
				}
				if pullErr != nil {
					log.DefaultLogger.Warnf("git informer: pull failed for %s/%s, recreating local clone: %v", trigger.Namespace, trigger.Name, pullErr)
				}
			}
		}
	}

	_ = os.RemoveAll(repoDir)
	parentDir := filepath.Dir(repoDir)
	if err = os.MkdirAll(parentDir, 0o700); err != nil {
		return nil, err
	}
	if err = os.Chmod(parentDir, 0o700); err != nil {
		return nil, err
	}

	var cloneErr error
	for _, reference := range references {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		_ = os.RemoveAll(repoDir)
		cloneOpts, err := cloneOptionsForRefWithClientOptions(gitConfig, i.options, reference, clientOptions)
		if err != nil {
			return nil, err
		}
		repo, err := git.PlainClone(repoDir, cloneOpts)
		if err == nil {
			i.revisions[key] = revision
			return repo, nil
		}
		cloneErr = err
	}
	return nil, cloneErr
}

func normalizeRevision(revision string) string {
	return strings.TrimSpace(revision)
}

func (i *Informer) openOrUpdateRepositoryWithCache(
	ctx context.Context,
	key string,
	trigger testkube.TestTrigger,
	cache *reconcileCache,
) (*git.Repository, string, *reconcileState, error) {
	if cache == nil {
		repo, err := i.openOrUpdateRepository(ctx, key, trigger)
		if err != nil {
			return nil, "", nil, err
		}
		head, err := repo.Head()
		if err != nil {
			return nil, "", nil, err
		}
		return repo, head.Hash().String(), nil, nil
	}

	state := cache.stateFor(trigger.Namespace, trigger.ContentSelector.Git)
	if state.repoLoaded {
		return state.repo, state.repoHead, state, state.repoErr
	}

	repo, err := i.openOrUpdateRepository(ctx, key, trigger)
	if err != nil {
		state.repoErr = err
		state.repoLoaded = true
		return nil, "", state, err
	}

	head, err := repo.Head()
	if err != nil {
		state.repoErr = err
		state.repoLoaded = true
		return nil, "", state, err
	}

	state.repo = repo
	state.repoHead = head.Hash().String()
	state.repoLoaded = true
	return state.repo, state.repoHead, state, nil
}

func (i *Informer) remoteHeadHash(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit) (string, error) {
	clientOptions, err := i.authClientOptions(ctx, namespace, gitConfig)
	if err != nil {
		return "", err
	}
	hash, _, err := remoteHeadHashAndRefWithClientOptions(gitConfig, i.options, clientOptions)
	return hash, err
}

func (i *Informer) remoteHeadHashAndRef(ctx context.Context, namespace string, gitConfig *testkube.TestTriggerContentGit) (string, string, error) {
	clientOptions, err := i.authClientOptions(ctx, namespace, gitConfig)
	if err != nil {
		return "", "", err
	}
	return remoteHeadHashAndRefWithClientOptions(gitConfig, i.options, clientOptions)
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

func remoteHeadHashAndRefWithClientOptions(gitConfig *testkube.TestTriggerContentGit, options Options, clientOptions []client.Option) (string, string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitConfig.Uri},
	})
	refs, err := remote.List(&git.ListOptions{
		ClientOptions: clientOptions,
		Timeout:       options.ListTimeoutSeconds,
	})
	if err != nil {
		return "", "", err
	}

	if references := effectiveRefs(gitConfig); len(references) > 0 {
		for _, reference := range references {
			for _, r := range refs {
				if r.Name() == plumbing.ReferenceName(reference) {
					return r.Hash().String(), reference, nil
				}
			}
		}
		return "", "", fmt.Errorf("reference not found: %q", strings.Join(references, ", "))
	}

	// No specific branches/tags: watch default HEAD
	for _, r := range refs {
		if r.Name() == plumbing.HEAD {
			return r.Hash().String(), string(r.Name()), nil
		}
	}
	for _, r := range refs {
		if r.Name().IsBranch() {
			return r.Hash().String(), string(r.Name()), nil
		}
	}

	return "", "", errors.New("unable to determine remote HEAD")
}

// remoteHeadHashWithClientOptions is kept for backward compatibility with clone/pull logic.
func remoteHeadHashWithClientOptions(gitConfig *testkube.TestTriggerContentGit, options Options, clientOptions []client.Option) (string, error) {
	hash, _, err := remoteHeadHashAndRefWithClientOptions(gitConfig, options, clientOptions)
	return hash, err
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
	return pullOptionsForRefWithClientOptions(gitConfig, options, reference, clientOptions)
}

func pullOptionsForRefWithClientOptions(gitConfig *testkube.TestTriggerContentGit, options Options, reference string, clientOptions []client.Option) (*git.PullOptions, error) {
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

	authType := ""
	if gitConfig.AuthType != nil {
		authType = strings.ToLower(string(*gitConfig.AuthType))
	}

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

	authType := ""
	if gitConfig.AuthType != nil {
		authType = strings.ToLower(string(*gitConfig.AuthType))
	}

	return strings.Join([]string{
		namespace,
		gitConfig.Uri,
		effectiveRefsKey(gitConfig),
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
		if file == p || strings.HasPrefix(file, p+"/") {
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
		// Replace ** with a pattern that matches any path segment
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			if prefix == "" && suffix == "" {
				return true
			}
			if prefix == "" {
				// **/suffix - match suffix anywhere
				if suffix == "" {
					return true
				}
				suffixMatch, _ := filepath.Match(suffix, filepath.Base(name))
				return suffixMatch || strings.HasSuffix(name, "/"+suffix)
			}
			if strings.HasPrefix(name, prefix+"/") || name == prefix {
				if suffix == "" {
					return true
				}
				remaining := strings.TrimPrefix(name, prefix+"/")
				suffixMatch, _ := filepath.Match(suffix, remaining)
				return suffixMatch || strings.HasSuffix(remaining, "/"+suffix)
			}
		}
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
func (i *Informer) collectHeadMetadata(repo *git.Repository, headHash string, gitConfig *testkube.TestTriggerContentGit) map[string]string {
	meta := make(map[string]string)
	meta[GitMetaKeyCommit] = headHash

	// Determine branch/tag from effective refs
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
