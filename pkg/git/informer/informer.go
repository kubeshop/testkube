package informer

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/triggers"
)

const (
	reconcileInterval = time.Minute
)

// NewInformer returns new informer instance
func NewInformer(testTriggerClient testtriggerclient.TestTriggerClient, matcher triggers.Matcher,
	namespace, environmentID string) *Informer {
	return &Informer{
		testTriggerClient: testTriggerClient,
		matcher:           matcher,
		triggers:          make(map[string]testkube.TestTrigger),
		commits:           make(map[string]map[string]struct{}),
		namespace:         namespace,
		environmentID:     environmentID,
	}
}

// Informer handles events emitting for git repo
type Informer struct {
	mutex             sync.RWMutex
	testTriggerClient testtriggerclient.TestTriggerClient
	triggers          map[string]testkube.TestTrigger
	commits           map[string]map[string]struct{}
	matcher           triggers.Matcher
	namespace         string
	environmentID     string
}

// UpdateListeners updates repositories
func (i *Informer) UpdateRepositories(ctx context.Context) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	list, err := i.testTriggerClient.List(ctx, i.environmentID, testtriggerclient.ListOptions{}, i.namespace)
	if err != nil {
		log.DefaultLogger.Errorf("informer service: error listing cloud test triggers: %v", err)
		return
	}

	for _, item := range list {
		if item.Resource != nil && *item.Resource == testkube.TestTriggerResources(testkube.CONTENT_TestTriggerResources) &&
			item.ContentSelector != nil && item.ContentSelector.Git != nil {
			i.triggers[item.Name] = item
			if _, ok := i.commits[item.Name]; !ok {
				i.commits[item.Name] = make(map[string]struct{})
			}
		} else {
			delete(i.triggers, item.Name)
			delete(i.commits, item.Name)
		}
	}

	for _, trigger := range i.triggers {
		if trigger.Disabled {
			continue
		}

		directory := "/tmp/" + trigger.Name
		r, err := git.PlainClone(directory, &git.CloneOptions{
			URL:           trigger.ContentSelector.Git.Uri,
			ReferenceName: plumbing.ReferenceName(trigger.ContentSelector.Git.Revision),
			SingleBranch:  true,
		})
		if err != nil {
			log.DefaultLogger.Errorf("informer service: error git clonning: %v", err)
			continue
		}

		ref, err := r.Head()
		if err != nil {
			log.DefaultLogger.Errorf("informer service: error pointing to head: %v", err)
			continue
		}

		// ... retrieves the commit history
		cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			log.DefaultLogger.Errorf("informer service: error retrieving commit history: %v", err)
			continue
		}

		matched := false
		// ... just iterates over the commits, printing it
		if err = cIter.ForEach(func(c *object.Commit) error {
			if _, ok := i.commits[trigger.Name][c.Hash.String()]; ok {
				return nil
			}

			i.commits[trigger.Name][c.Hash.String()] = struct{}{}
			// ... retrieve the tree from the commit
			tree, err := c.Tree()
			if err != nil {
				log.DefaultLogger.Errorf("informer service: error getting tree: %v", err)
				return err
			}

			// ... get the files iterator and print the file
			tree.Files().ForEach(func(f *object.File) error {
				for _, path := range trigger.ContentSelector.Git.Paths {
					if f.Name == path || strings.HasPrefix(f.Name, path+"/") {
						matched = true
						return nil
					}
				}
				return nil
			})

			return nil
		}); err != nil {
			log.DefaultLogger.Errorf("informer service: error printing commit: %v", err)
		}

		if matched {
			i.Notify(ctx, trigger.Name)
		}
	}
}

// Notify notifies informer
func (i *Informer) Notify(ctx context.Context, triggerName string) {
	log.DefaultLogger.Info("informer service: publishing event")
	event := triggers.NewWatcherEvent(testtrigger.EventModified, &metav1.ObjectMeta{}, nil,
		testtrigger.ResourceType(testkube.CONTENT_TestTriggerResources), triggers.WithTrigger(triggerName))
	if err := i.matcher.Match(ctx, event); err != nil {
		log.DefaultLogger.Errorf("informer service: error matching event: %v", err)
	}
}

// Reconcile reloads listeners from all registered reconcilers
func (i *Informer) Reconcile(ctx context.Context) {
	log.DefaultLogger.Info("informer service: starting reconciler")
	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Info("informer service: stopping reconciler")
			return
		default:
			i.UpdateRepositories(ctx)
			time.Sleep(reconcileInterval)
		}
	}
}
