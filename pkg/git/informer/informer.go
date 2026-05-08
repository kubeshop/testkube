package informer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
}

// NewInformer returns a new git content informer.
func NewInformer(
	testTriggerClient testtriggerclient.TestTriggerClient,
	matcher Matcher,
	namespace string,
	environmentID string,
) *Informer {
	return &Informer{
		testTriggerClient: testTriggerClient,
		matcher:           matcher,
		commits:           make(map[string]string),
		namespace:         namespace,
		environmentID:     environmentID,
	}
}

// Reconcile periodically polls git repositories and emits trigger events.
func (i *Informer) Reconcile(ctx context.Context) {
	log.DefaultLogger.Info("git informer: starting reconciler")
	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Info("git informer: stopping reconciler")
			return
		default:
			i.updateRepositories(ctx)
			time.Sleep(reconcileInterval)
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
			_ = os.RemoveAll(filepath.Join(os.TempDir(), "testkube-git-trigger", k))
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

	repo, err := i.openOrUpdateRepository(trigger)
	if err != nil {
		return false, err
	}

	head, err := repo.Head()
	if err != nil {
		return false, err
	}

	key := triggerKey(trigger.Namespace, trigger.Name)
	headHash := head.Hash().String()
	prevHash, hasPrev := i.commits[key]
	i.commits[key] = headHash

	if !hasPrev || prevHash == headHash {
		return false, nil
	}

	// Walk commits from head back to previous to check if any changed paths match
	iter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return false, err
	}

	foundPrev := false
	matched := false
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == prevHash {
			foundPrev = true
			return storer.ErrStop
		}
		stats, err := c.Stats()
		if err != nil {
			return err
		}
		for _, stat := range stats {
			if pathMatches(paths, stat.Name) {
				matched = true
				return storer.ErrStop
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		return false, err
	}

	if !foundPrev {
		return true, nil
	}
	return matched, nil
}

func (i *Informer) hasNewHeadCommit(trigger testkube.TestTrigger) (bool, error) {
	headHash, err := remoteHeadHash(trigger.ContentSelector.Git)
	if err != nil {
		return false, err
	}

	key := triggerKey(trigger.Namespace, trigger.Name)
	prevHash, hasPrev := i.commits[key]
	i.commits[key] = headHash

	return hasPrev && prevHash != headHash, nil
}

func (i *Informer) openOrUpdateRepository(trigger testkube.TestTrigger) (*git.Repository, error) {
	repoDir := filepath.Join(os.TempDir(), "testkube-git-trigger", triggerKey(trigger.Namespace, trigger.Name))
	gitConfig := trigger.ContentSelector.Git

	repo, err := git.PlainOpen(repoDir)
	if err == nil {
		worktree, wtErr := repo.Worktree()
		if wtErr == nil {
			pullOpts, pullErr := pullOptions(gitConfig)
			if pullErr != nil {
				return nil, pullErr
			}
			if err = worktree.Pull(pullOpts); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				return nil, err
			}
			return repo, nil
		}
	}

	_ = os.RemoveAll(repoDir)
	if err = os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return nil, err
	}

	cloneOpts, err := cloneOptions(gitConfig)
	if err != nil {
		return nil, err
	}

	return git.PlainClone(repoDir, cloneOpts)
}

func remoteHeadHash(gitConfig *testkube.TestTriggerContentGit) (string, error) {
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
	})
	if err != nil {
		return "", err
	}

	if ref := normalizeRef(gitConfig.Revision); ref != "" {
		for _, r := range refs {
			if r.Name() == plumbing.ReferenceName(ref) {
				return r.Hash().String(), nil
			}
		}
		return "", fmt.Errorf("reference %q not found", ref)
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

func cloneOptions(gitConfig *testkube.TestTriggerContentGit) (*git.CloneOptions, error) {
	clientOptions, err := authClientOptions(gitConfig)
	if err != nil {
		return nil, err
	}

	cloneOpts := &git.CloneOptions{
		URL:           gitConfig.Uri,
		SingleBranch:  true,
		ClientOptions: clientOptions,
	}
	if ref := normalizeRef(gitConfig.Revision); ref != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(ref)
	}

	return cloneOpts, nil
}

func pullOptions(gitConfig *testkube.TestTriggerContentGit) (*git.PullOptions, error) {
	clientOptions, err := authClientOptions(gitConfig)
	if err != nil {
		return nil, err
	}

	pullOpts := &git.PullOptions{
		RemoteName:    "origin",
		SingleBranch:  true,
		Force:         true,
		ClientOptions: clientOptions,
	}
	if ref := normalizeRef(gitConfig.Revision); ref != "" {
		pullOpts.ReferenceName = plumbing.ReferenceName(ref)
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
			user = "git"
		}
		publicKeys, err := ssh.NewPublicKeys(user, []byte(sshKey), "")
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithSSHAuth(publicKeys))
	case token != "" && (authType == string(testkube.HEADER_ContentGitAuthType) || authType == "github"):
		opts = append(opts, client.WithHTTPAuth(&http.TokenAuth{Token: token}))
	case token != "" || username != "":
		if username == "" {
			username = "git"
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
	if source.SecretKeyRef != nil && source.SecretKeyRef.Key != "" {
		return os.Getenv(source.SecretKeyRef.Key)
	}
	if source.ConfigMapKeyRef != nil && source.ConfigMapKeyRef.Key != "" {
		return os.Getenv(source.ConfigMapKeyRef.Key)
	}
	return ""
}

func normalizeRef(revision string) string {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		return ""
	}
	if strings.HasPrefix(revision, "refs/") {
		return revision
	}
	return "refs/heads/" + revision
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
	for _, p := range normalizePaths(paths) {
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

func triggerKey(namespace, name string) string {
	return namespace + "/" + name
}
