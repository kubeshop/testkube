package cronjob

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

const (
	watcherDelay = 200 * time.Millisecond
)

//go:generate mockgen -destination=./mock_scheduler.go -package=cronjob "github.com/kubeshop/testkube/pkg/cronjob" Interface
type Interface interface {
	Reconcile(ctx context.Context) error
}

// Scheduler provide methods to schedule cronjobs
type Scheduler struct {
	testWorkflowClient          testworkflowclient.TestWorkflowClient
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient
	testWorkflowExecutor        testworkflowexecutor.TestWorkflowExecutor
	logger                      *zap.SugaredLogger
	proContext                  *intconfig.ProContext
	cronService                 *cron.Cron
	testWorklows                map[string]map[string]cron.EntryID
}

// New is a method to create new cronjob scheduler
func New(testWorkflowClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	logger *zap.SugaredLogger) *Scheduler {
	return &Scheduler{
		testWorkflowClient:          testWorkflowClient,
		testWorkflowTemplatesClient: testWorkflowTemplatesClient,
		testWorkflowExecutor:        testWorkflowExecutor,
		logger:                      logger,
		cronService:                 cron.New(),
		testWorklows:                make(map[string]map[string]cron.EntryID),
	}
}

type Option func(*Scheduler)

func WithProContext(proContext *intconfig.ProContext) Option {
	return func(s *Scheduler) {
		s.proContext = proContext
	}
}

// Reconcile is watching for test workflow and test worklow template change and schedule test workflow cron jobs
func (s *Scheduler) Reconcile(ctx context.Context) error {
	s.cronService.Start()
	defer s.cronService.Stop()

	includeInitialData := true
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			watcher := s.testWorkflowClient.WatchUpdates(ctx, s.getEnvironmentId(), includeInitialData)
			for obj := range watcher.Channel() {
				var err error
				switch obj.Type {
				case testworkflowclient.EventTypeCreate:
					if obj.Resource == nil {
						continue
					}

					for _, event := range obj.Resource.Spec.Events {
						if event.Cronjob != nil {
							var name string
							name, err = getHashedMetadataName(event.Cronjob.Cron, event.Cronjob.Config)
							if err != nil {
								break
							}

							if _, ok := s.testWorklows[obj.Resource.Name]; !ok {
								s.testWorklows[obj.Resource.Name] = make(map[string]cron.EntryID, 0)
							}

							if _, ok := s.testWorklows[obj.Resource.Name][name]; !ok {
								var entryID cron.EntryID
								entryID, err = s.cronService.AddJob(event.Cronjob.Cron,
									cron.FuncJob(func() { s.execute(ctx, obj.Resource.Name, event.Cronjob) }))
								if err != nil {
									break
								}

								s.testWorklows[obj.Resource.Name][name] = entryID
							}

						}
					}
				case testworkflowclient.EventTypeDelete:
					if obj.Resource == nil {
						continue
					}

					if entryIDs, ok := s.testWorklows[obj.Resource.Name]; ok {
						for _, entryID := range entryIDs {
							s.cronService.Remove(entryID)
							delete(s.testWorklows, obj.Resource.Name)
						}
					}
				default:
					err = errors.New("unknown event type")
				}

				if err == nil {
					s.logger.Infow("cron job schedduler: reconciler component: scheduled TestWorkflow to cron jobs", "name", obj.Resource.Name, "error", err)
				} else {
					s.logger.Errorw("cron job schedduler: reconciler component: failed to watch TestWorkflows", "error", err)
				}
			}
			if watcher.Err() != nil {
				s.logger.Errorw("cron job schedduler: reconciler component: failed to watch TestWorkflows", "error", watcher.Err())
			} else {
				includeInitialData = false
			}

			time.Sleep(watcherDelay)
		}
	}

}
func (s *Scheduler) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}

	return ""
}

// getHashedMetadataName returns cron job hashed metadata name
func getHashedMetadataName(schedule string, config map[string]string) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%x", schedule, sha256.Sum256(data)), nil
}
