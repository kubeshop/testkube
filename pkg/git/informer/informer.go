package informer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/triggers"
)

const (
	reconcileInterval = time.Hour
)

// NewInformer returns new informer instance
func NewInformer(testTriggerClient testtriggerclient.TestTriggerClient, service *triggers.Service,
	namespace, environmentID string) *Informer {
	return &Informer{
		testTriggerClient: testTriggerClient,
		service:           service,
		triggers:          make(map[string]testkube.TestTrigger),
		namespace:         namespace,
		environmentID:     environmentID,
	}
}

// Informer handles events emitting for git repo
type Informer struct {
	mutex             sync.RWMutex
	testTriggerClient testtriggerclient.TestTriggerClient
	triggers          map[string]testkube.TestTrigger
	service           *triggers.Service
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
		} else {
			delete(i.triggers, item.Name)
		}
	}

	for _, trigger := range i.triggers {
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
		log.DefaultLogger.Infow("informer service: step 1", "trigger", trigger)

		ref, err := r.Head()
		if err != nil {
			log.DefaultLogger.Errorf("informer service: error pointing to head: %v", err)
			continue
		}
		log.DefaultLogger.Infow("informer service: step 2", "trigger", trigger)
		// ... retrieves the commit history
		cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			log.DefaultLogger.Errorf("informer service: error retrieving commit history: %v", err)
			continue
		}
		log.DefaultLogger.Infow("informer service: step 3", "trigger", trigger)
		// ... just iterates over the commits, printing it
		if err = cIter.ForEach(func(c *object.Commit) error {
			// ... retrieve the tree from the commit
			tree, err := c.Tree()
			if err != nil {
				log.DefaultLogger.Errorf("informer service: error getting tree: %v", err)
				return err
			}

			// ... get the files iterator and print the file
			tree.Files().ForEach(func(f *object.File) error {
				fmt.Printf("100644 blob %s    %s\n", f.Hash, f.Name)
				return nil
			})

			return nil
		}); err != nil {
			log.DefaultLogger.Errorf("informer service: error printing commit: %v", err)
		}
	}
}

// Notify notifies informer
func (i *Informer) Notify(event testkube.Event) {

	log.DefaultLogger.Debugw("informer service: event published", event.Log()...)
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
