package informer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/storer"

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
		}
	}
}

func isGitContentTrigger(trigger testkube.TestTrigger) bool {
	return !trigger.Disabled &&
		trigger.Resource != nil && *trigger.Resource == testkube.CONTENT_TestTriggerResources &&
		trigger.ContentSelector != nil &&
		trigger.ContentSelector.Git != nil &&
		trigger.ContentSelector.Git.Uri != ""
}

func (i *Informer) hasNewMatchingCommit(trigger testkube.TestTrigger) (bool, error) {
	repoDir := filepath.Join(os.TempDir(), "testkube-git-trigger", trigger.Namespace, trigger.Name)
	_ = os.RemoveAll(repoDir)
	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return false, err
	}

	cloneOpts := &git.CloneOptions{
		URL:          trigger.ContentSelector.Git.Uri,
		SingleBranch: true,
	}
	if ref := normalizeRef(trigger.ContentSelector.Git.Revision); ref != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(ref)
	}

	repo, err := git.PlainClone(repoDir, cloneOpts)
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

	paths := trigger.ContentSelector.Git.Paths
	if len(paths) == 0 {
		return true, nil
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

func pathMatches(paths []string, file string) bool {
	for _, p := range paths {
		p = strings.Trim(strings.TrimSpace(p), "/")
		if p == "" {
			continue
		}
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

func triggerKey(namespace, name string) string {
	return namespace + "/" + name
}
