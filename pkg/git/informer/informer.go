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
	"github.com/kubeshop/testkube/pkg/newclients/workflowtriggerclient"
)

const defaultReconcileInterval = time.Minute
const defaultGitUsername = "git"
const testTriggerSource = "v1"
const workflowTriggerSource = "v2"
const allNamespacesMarker = "*"

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
	MatchGitTrigger(ctx context.Context, triggerName, namespace string) error
	MatchGitWorkflowTrigger(ctx context.Context, triggerName, namespace string) error
}

// Informer polls git repositories referenced by content triggers and fires
// events when matching commits are detected.
type Informer struct {
	mu                sync.Mutex
	testTriggerClient testtriggerclient.TestTriggerClient
	workflowClient    workflowtriggerclient.WorkflowTriggerClient
	matcher           Matcher
	commits           map[string]string // key -> last seen head hash
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

// NewInformer returns a new git content informer.
func NewInformer(
	testTriggerClient testtriggerclient.TestTriggerClient,
	workflowClient workflowtriggerclient.WorkflowTriggerClient,
	matcher Matcher,
	namespace string,
	environmentID string,
	options Options,
) *Informer {
	options = normalizeOptions(options)
	return &Informer{
		testTriggerClient: testTriggerClient,
		workflowClient:    workflowClient,
		matcher:           matcher,
		commits:           make(map[string]string),
		namespaces:        resolveNamespaces(options.WatcherNamespaces, namespace),
		environmentID:     environmentID,
		options:           options,
		kubeClient:        options.KubeClient,
	}
}

func resolveNamespaces(watcherNamespaces, defaultNamespace string) []string {
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
	defaultNamespace = strings.TrimSpace(defaultNamespace)
	if defaultNamespace != "" {
		return []string{defaultNamespace}
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
	workflowTriggerMap := make(map[string]testkube.WorkflowTrigger)
	workflowTriggerListSucceeded := false
	workflowTriggerListedNamespaces := make(map[string]struct{})
	if i.workflowClient != nil {
		for _, namespace := range i.namespaces {
			workflowTriggerList, err := i.workflowClient.List(ctx, i.environmentID, workflowtriggerclient.ListOptions{}, namespace)
			if err != nil {
				log.DefaultLogger.Errorf("git informer: error listing workflow triggers in namespace %q: %v", namespace, err)
				continue
			}
			workflowTriggerListSucceeded = true
			workflowTriggerListedNamespaces[namespace] = struct{}{}
			for _, trigger := range workflowTriggerList {
				workflowTriggerMap[triggerKey(workflowTriggerSource, trigger.Namespace, trigger.Name)] = trigger
			}
		}
	}
	if !testTriggerListSucceeded && !workflowTriggerListSucceeded {
		return
	}

	active := make(map[string]struct{}, len(testTriggerMap)+len(workflowTriggerMap))
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

		changed, err := i.hasNewMatchingCommitWithCache(ctx, key, trigger, cache)
		if err != nil {
			log.DefaultLogger.Errorf("git informer: error checking trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
			continue
		}
		if !changed || i.matcher == nil {
			continue
		}
		if err := i.matcher.MatchGitTrigger(ctx, trigger.Name, trigger.Namespace); err != nil {
			log.DefaultLogger.Errorf("git informer: error matching trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
		}
	}

	for _, workflowTrigger := range workflowTriggerMap {
		if err := ctx.Err(); err != nil {
			return
		}
		if !isGitContentWorkflowTrigger(workflowTrigger) {
			continue
		}

		trigger := workflowTriggerToTestTrigger(workflowTrigger)
		key := triggerKey(workflowTriggerSource, trigger.Namespace, trigger.Name)
		active[key] = struct{}{}

		changed, err := i.hasNewMatchingCommitWithCache(ctx, key, trigger, cache)
		if err != nil {
			log.DefaultLogger.Errorf("git informer: error checking workflow trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
			continue
		}
		if !changed || i.matcher == nil {
			continue
		}
		if err := i.matcher.MatchGitWorkflowTrigger(ctx, trigger.Name, trigger.Namespace); err != nil {
			log.DefaultLogger.Errorf("git informer: error matching workflow trigger %s/%s: %v", trigger.Namespace, trigger.Name, err)
		}
	}

	// Clean up commits for removed triggers
	for k := range i.commits {
		if _, ok := active[k]; !ok {
			source, namespace, _, parsed := parseTriggerKey(k)
			if parsed {
				switch source {
				case testTriggerSource:
					if _, all := testTriggerListedNamespaces[allNamespacesMarker]; !all {
						if _, listed := testTriggerListedNamespaces[namespace]; !listed {
							continue
						}
					}
				case workflowTriggerSource:
					if _, all := workflowTriggerListedNamespaces[allNamespacesMarker]; !all {
						if _, listed := workflowTriggerListedNamespaces[namespace]; !listed {
							continue
						}
					}
				}
			}
			delete(i.commits, k)
			_ = os.RemoveAll(triggerRepositoryPathFromKey(k))
		}
	}
}

func isGitContentTrigger(trigger testkube.TestTrigger) bool {
	return !trigger.Disabled &&
		isContentResource(trigger) &&
		isModifiedGitContentEvent(trigger.Event) &&
		trigger.ContentSelector != nil &&
		trigger.ContentSelector.Git != nil &&
		trigger.ContentSelector.Git.Uri != ""
}

func isGitContentWorkflowTrigger(trigger testkube.WorkflowTrigger) bool {
	if trigger.Disabled || trigger.When.Git == nil || trigger.When.Git.Uri == "" {
		return false
	}
	return isModifiedGitContentEvent(trigger.When.Event) &&
		(trigger.Watch == nil || strings.EqualFold(trigger.Watch.Resource.Kind, string(testkube.CONTENT_TestTriggerResources)))
}

func isModifiedGitContentEvent(event string) bool {
	return strings.EqualFold(event, "modified")
}

func workflowTriggerToTestTrigger(trigger testkube.WorkflowTrigger) testkube.TestTrigger {
	resource := testkube.CONTENT_TestTriggerResources
	out := testkube.TestTrigger{
		Name:      trigger.Name,
		Namespace: trigger.Namespace,
		Disabled:  trigger.Disabled,
		Resource:  &resource,
		ContentSelector: &testkube.TestTriggerContentSelector{
			Git: trigger.When.Git,
		},
	}
	if trigger.Watch != nil && trigger.Watch.Resource.Kind != "" {
		out.ResourceRef = &testkube.TestTriggerResourceRef{Kind: trigger.Watch.Resource.Kind}
	}
	return out
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

func (i *Informer) hasNewMatchingCommit(ctx context.Context, key string, trigger testkube.TestTrigger) (bool, error) {
	return i.hasNewMatchingCommitWithCache(ctx, key, trigger, nil)
}

func (i *Informer) hasNewMatchingCommitWithCache(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	paths := normalizePaths(trigger.ContentSelector.Git.Paths)
	if len(paths) == 0 {
		return i.hasNewHeadCommitWithCache(ctx, key, trigger, cache)
	}

	if isCommitSHA(trigger.ContentSelector.Git.Revision) {
		// Commit SHA is immutable; there is no moving ref to watch for path changes.
		i.commits[key] = strings.TrimSpace(trigger.ContentSelector.Git.Revision)
		return false, nil
	}

	repo, headHash, state, err := i.openOrUpdateRepositoryWithCache(ctx, key, trigger, cache)
	if err != nil {
		return false, err
	}
	prevHash, hasPrev := i.commits[key]

	if !hasPrev {
		i.commits[key] = headHash
		return false, nil
	}
	if prevHash == headHash {
		return false, nil
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
		return false, err
	}

	i.commits[key] = headHash

	for _, changedPath := range delta.paths {
		if pathMatchesNormalized(paths, changedPath) {
			return true, nil
		}
	}
	if !delta.foundPrev {
		log.DefaultLogger.Warnf(
			"git informer: history boundary reached before previous commit for trigger %s/%s (repo depth/max scan limit); advancing baseline without firing",
			trigger.Namespace,
			trigger.Name,
		)
		return false, nil
	}
	return false, nil
}

func (i *Informer) hasNewHeadCommit(ctx context.Context, key string, trigger testkube.TestTrigger) (bool, error) {
	return i.hasNewHeadCommitWithCache(ctx, key, trigger, nil)
}

func (i *Informer) hasNewHeadCommitWithCache(ctx context.Context, key string, trigger testkube.TestTrigger, cache *reconcileCache) (bool, error) {
	headHash, err := i.remoteHeadHashWithCache(ctx, trigger.Namespace, trigger.ContentSelector.Git, cache)
	if err != nil {
		return false, err
	}

	prevHash, hasPrev := i.commits[key]
	i.commits[key] = headHash

	return hasPrev && prevHash != headHash, nil
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

func (i *Informer) openOrUpdateRepository(ctx context.Context, key string, trigger testkube.TestTrigger) (*git.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repoDir := triggerRepositoryPathFromKey(key)
	gitConfig := trigger.ContentSelector.Git
	references := normalizeRefs(gitConfig.Revision)
	if len(references) == 0 {
		references = []string{""}
	}
	clientOptions, err := i.authClientOptions(ctx, trigger.Namespace, gitConfig)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(repoDir)
	if err == nil {
		if !repositoryOriginMatches(repo, gitConfig.Uri) {
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
			return repo, nil
		}
		cloneErr = err
	}
	return nil, cloneErr
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
	return remoteHeadHashWithClientOptions(gitConfig, i.options, clientOptions)
}

func remoteHeadHash(gitConfig *testkube.TestTriggerContentGit, options Options) (string, error) {
	clientOptions, err := authClientOptions(gitConfig)
	if err != nil {
		return "", err
	}
	return remoteHeadHashWithClientOptions(gitConfig, options, clientOptions)
}

func remoteHeadHashWithClientOptions(gitConfig *testkube.TestTriggerContentGit, options Options, clientOptions []client.Option) (string, error) {
	if isCommitSHA(gitConfig.Revision) {
		return strings.TrimSpace(gitConfig.Revision), nil
	}

	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitConfig.Uri},
	})
	refs, err := remote.List(&git.ListOptions{
		ClientOptions: clientOptions,
		Timeout:       options.ListTimeoutSeconds,
	})
	if err != nil {
		return "", err
	}

	if references := normalizeRefs(gitConfig.Revision); len(references) > 0 {
		for _, reference := range references {
			for _, r := range refs {
				if r.Name() == plumbing.ReferenceName(reference) {
					return r.Hash().String(), nil
				}
			}
		}
		return "", fmt.Errorf("reference not found: %q", strings.Join(references, ", "))
	}

	for _, r := range refs {
		if r.Name() == plumbing.HEAD {
			return r.Hash().String(), nil
		}
	}
	for _, r := range refs {
		if r.Name().IsBranch() {
			return r.Hash().String(), nil
		}
	}

	return "", errors.New("unable to determine remote HEAD")
}

func cloneOptions(gitConfig *testkube.TestTriggerContentGit, options Options) (*git.CloneOptions, error) {
	references := normalizeRefs(gitConfig.Revision)
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
	references := normalizeRefs(gitConfig.Revision)
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
	case token != "" && (authType == string(testkube.HEADER_ContentGitAuthType) || authType == "github"):
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

	if source.FileKeyRef != nil {
		return fmt.Errorf("unsupported %s source: fileKeyRef is not supported for git credentials, use secretKeyRef or configMapKeyRef", fieldName)
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
	if source.SecretKeyRef != nil {
		if resolved, ok := i.resolveSecretKeyRefValue(ctx, namespace, source.SecretKeyRef); ok {
			return resolved
		}
	}
	if source.ConfigMapKeyRef != nil {
		if resolved, ok := i.resolveConfigMapKeyRefValue(ctx, namespace, source.ConfigMapKeyRef); ok {
			return resolved
		}
	}
	return resolveCredentialValue(value, source)
}

func (i *Informer) resolveSecretKeyRefValue(ctx context.Context, namespace string, ref *testkube.EnvVarSourceSecretKeyRef) (string, bool) {
	if ref == nil || ref.Name == "" || ref.Key == "" {
		return "", false
	}
	secret, err := i.kubeClient.CoreV1().Secrets(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		if !(apierrors.IsNotFound(err) && ref.Optional != nil && *ref.Optional) {
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
		if !(apierrors.IsNotFound(err) && ref.Optional != nil && *ref.Optional) {
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
		gitConfig.Revision,
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
		fileKeyRefCacheKey(source.FileKeyRef),
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

func fileKeyRefCacheKey(ref *testkube.EnvVarSourceFileKeyRef) string {
	if ref == nil {
		return ""
	}
	return strings.Join([]string{ref.VolumeName, ref.Path, ref.Key, boolPtrString(ref.Optional)}, ":")
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
