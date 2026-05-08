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

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
)

const reconcileInterval = 2 * time.Minute
const defaultGitUsername = "git"

// envVarNameSanitizer normalizes Secret/ConfigMap name+key into env-var-safe tokens.
var envVarNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)
var gitCommitSHAPattern = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

type Options struct {
	RepoDepth          int
	ListTimeoutSeconds int
	MaxCommitsScan     int
	PullRetries        int
	PullRetryDelay     time.Duration
}

func normalizeOptions(opts Options) Options {
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
}

// Informer polls git repositories referenced by content triggers and fires
// events when matching commits are detected.
type Informer struct {
	mu                sync.Mutex
	testTriggerClient testtriggerclient.TestTriggerClient
	matcher           Matcher
	commits           map[string]string // key -> last seen head hash
	namespace         string
	environmentID     string
	options           Options
}

// NewInformer returns a new git content informer.
func NewInformer(
	testTriggerClient testtriggerclient.TestTriggerClient,
	matcher Matcher,
	namespace string,
	environmentID string,
	options Options,
) *Informer {
	return &Informer{
		testTriggerClient: testTriggerClient,
		matcher:           matcher,
		commits:           make(map[string]string),
		namespace:         namespace,
		environmentID:     environmentID,
		options:           normalizeOptions(options),
	}
}

// Reconcile periodically polls git repositories and emits trigger events.
func (i *Informer) Reconcile(ctx context.Context) {
	log.DefaultLogger.Info("git informer: starting reconciler")

	i.updateRepositories(ctx)

	ticker := time.NewTicker(reconcileInterval)
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

	list, err := i.testTriggerClient.List(ctx, i.environmentID, testtriggerclient.ListOptions{}, i.namespace)
	if err != nil {
		log.DefaultLogger.Errorf("git informer: error listing triggers: %v", err)
		return
	}

	active := make(map[string]struct{}, len(list))
	for idx := range list {
		trigger := list[idx]
		if !isGitContentTrigger(trigger) {
			continue
		}
		key := triggerKey(trigger.Namespace, trigger.Name)
		active[key] = struct{}{}

		changed, err := i.hasNewMatchingCommit(trigger)
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

	// Clean up commits for removed triggers
	for k := range i.commits {
		if _, ok := active[k]; !ok {
			delete(i.commits, k)
			_ = os.RemoveAll(triggerRepositoryPathFromKey(k))
		}
	}
}

func isGitContentTrigger(trigger testkube.TestTrigger) bool {
	return !trigger.Disabled &&
		isContentResource(trigger) &&
		trigger.ContentSelector != nil &&
		trigger.ContentSelector.Git != nil &&
		trigger.ContentSelector.Git.Uri != ""
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

func (i *Informer) hasNewMatchingCommit(trigger testkube.TestTrigger) (bool, error) {
	paths := normalizePaths(trigger.ContentSelector.Git.Paths)
	if len(paths) == 0 {
		return i.hasNewHeadCommit(trigger)
	}

	key := triggerKey(trigger.Namespace, trigger.Name)
	if isCommitSHA(trigger.ContentSelector.Git.Revision) {
		// Commit SHA is immutable; there is no moving ref to watch for path changes.
		i.commits[key] = strings.TrimSpace(trigger.ContentSelector.Git.Revision)
		return false, nil
	}

	repo, err := i.openOrUpdateRepository(trigger)
	if err != nil {
		return false, err
	}

	head, err := repo.Head()
	if err != nil {
		return false, err
	}

	headHash := head.Hash().String()
	prevHash, hasPrev := i.commits[key]

	if !hasPrev {
		i.commits[key] = headHash
		return false, nil
	}
	if prevHash == headHash {
		return false, nil
	}

	// Walk commits from head back to previous to check if any changed paths match
	iter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return false, err
	}
	defer iter.Close()

	foundPrev := false
	matched := false
	scanned := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if i.options.MaxCommitsScan > 0 && scanned >= i.options.MaxCommitsScan {
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
			if pathMatchesNormalized(paths, stat.Name) {
				matched = true
				return storer.ErrStop
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		if shouldAdvanceBaselineOnScanError(err) {
			i.commits[key] = headHash
		}
		return false, err
	}

	i.commits[key] = headHash

	if matched {
		return true, nil
	}
	if !foundPrev {
		log.DefaultLogger.Warnf(
			"git informer: history boundary reached before previous commit for trigger %s/%s (repo depth/max scan limit); advancing baseline without firing",
			trigger.Namespace,
			trigger.Name,
		)
		return false, nil
	}
	return false, nil
}

func (i *Informer) hasNewHeadCommit(trigger testkube.TestTrigger) (bool, error) {
	headHash, err := remoteHeadHash(trigger.ContentSelector.Git, i.options)
	if err != nil {
		return false, err
	}

	key := triggerKey(trigger.Namespace, trigger.Name)
	prevHash, hasPrev := i.commits[key]
	i.commits[key] = headHash

	return hasPrev && prevHash != headHash, nil
}

func (i *Informer) openOrUpdateRepository(trigger testkube.TestTrigger) (*git.Repository, error) {
	repoDir := triggerRepositoryPath(trigger.Namespace, trigger.Name)
	gitConfig := trigger.ContentSelector.Git
	references := normalizeRefs(gitConfig.Revision)
	if len(references) == 0 {
		references = []string{""}
	}

	repo, err := git.PlainOpen(repoDir)
	if err == nil {
		worktree, wtErr := repo.Worktree()
		if wtErr == nil {
			var pullErr error
			for _, reference := range references {
				pullOpts, err := pullOptionsForRef(gitConfig, i.options, reference)
				if err != nil {
					return nil, err
				}
				for attempt := 0; attempt <= i.options.PullRetries; attempt++ {
					pullErr = worktree.Pull(pullOpts)
					if pullErr == nil || errors.Is(pullErr, git.NoErrAlreadyUpToDate) {
						return repo, nil
					}
					if attempt < i.options.PullRetries && i.options.PullRetryDelay > 0 {
						time.Sleep(i.options.PullRetryDelay)
					}
				}
			}
			if pullErr != nil {
				log.DefaultLogger.Warnf("git informer: pull failed for %s/%s, recreating local clone: %v", trigger.Namespace, trigger.Name, pullErr)
			}
		}
	}

	_ = os.RemoveAll(repoDir)
	if err = os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return nil, err
	}

	var cloneErr error
	for _, reference := range references {
		_ = os.RemoveAll(repoDir)
		cloneOpts, err := cloneOptionsForRef(gitConfig, i.options, reference)
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

func remoteHeadHash(gitConfig *testkube.TestTriggerContentGit, options Options) (string, error) {
	if isCommitSHA(gitConfig.Revision) {
		return strings.TrimSpace(gitConfig.Revision), nil
	}

	clientOptions, err := authClientOptions(gitConfig)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("reference not found: %q", strings.Join(references, "\", \""))
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
	username := resolveCredentialValue(gitConfig.Username, gitConfig.UsernameFrom)
	token := resolveCredentialValue(gitConfig.Token, gitConfig.TokenFrom)
	sshKey := resolveCredentialValue(gitConfig.SshKey, gitConfig.SshKeyFrom)

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

func triggerKey(namespace, name string) string {
	return namespace + "/" + name
}

func triggerRepositoryPath(namespace, name string) string {
	return filepath.Join(os.TempDir(), "testkube-git-trigger", triggerKey(namespace, name))
}

func triggerRepositoryPathFromKey(key string) string {
	return filepath.Join(os.TempDir(), "testkube-git-trigger", key)
}
