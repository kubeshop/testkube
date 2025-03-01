package cronjob

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

const (
	watcherDelay = 200 * time.Millisecond
)

//go:generate mockgen -destination=./mock_scheduler.go -package=cronjob "github.com/kubeshop/testkube/pkg/cronjob" Interface
type Interface interface {
	Reconcile(ctx context.Context)
	ReconcileTestWorkflows(ctx context.Context) error
	ReconcileTestWorkflowTemplates(ctx context.Context) error
}

// Scheduler provide methods to schedule cron jobs
type Scheduler struct {
	testWorkflowClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient
	testWorkflowExecutor       testworkflowexecutor.TestWorkflowExecutor
	logger                     *zap.SugaredLogger
	proContext                 *intconfig.ProContext
	cronService                *cron.Cron
	testWorklows               map[string]map[string]cron.EntryID
	lock                       sync.RWMutex
}

// New is a method to create new cron job scheduler
func New(testWorkflowClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	logger *zap.SugaredLogger) *Scheduler {
	return &Scheduler{
		testWorkflowClient:         testWorkflowClient,
		testWorkflowTemplateClient: testWorkflowTemplateClient,
		testWorkflowExecutor:       testWorkflowExecutor,
		logger:                     logger,
		cronService:                cron.New(),
		testWorklows:               make(map[string]map[string]cron.EntryID),
	}
}

type Option func(*Scheduler)

func WithProContext(proContext *intconfig.ProContext) Option {
	return func(s *Scheduler) {
		s.proContext = proContext
	}
}

// Reconcile is reconciling cron jobs
func (s *Scheduler) Reconcile(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()

		if err := s.ReconcileTestWorkflows(ctx); err != nil {
			s.logger.Errorw("cron job scheduler: reconciler component: failed to reconcile TestWorkflows", "error", err)
		}
	}()

	go func() {
		defer wg.Done()

		if err := s.ReconcileTestWorkflowTemplates(ctx); err != nil {
			s.logger.Errorw("cron job scheduler: reconciler component: failed to reconcile TestWorkflowTemplates", "error", err)
		}
	}()

	wg.Wait()
}

// ReconcileTestWorklows is watching for test workflow and test worklow template change and schedule test workflow cron jobs
func (s *Scheduler) ReconcileTestWorkflows(ctx context.Context) error {
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
				if obj.Resource == nil || obj.Resource.Spec == nil {
					continue
				}

				events := obj.Resource.Spec.Events
				for _, template := range obj.Resource.Spec.Use {
					testWorkflowTemplate, err := s.testWorkflowTemplateClient.Get(ctx, s.getEnvironmentId(), template.Name)
					if err != nil {
						s.logger.Errorw("cron job schedduler: reconciler component: failed to get TestWorkflowTemplate", "namr", template.Name, "error", err)
						continue
					}

					if testWorkflowTemplate.Spec == nil {
						continue
					}

					events = append(events, testWorkflowTemplate.Spec.Events...)
				}

				var err error
				switch obj.Type {
				case testworkflowclient.EventTypeCreate:
					err = s.addTestWorkflowCronJobs(ctx, obj.Resource.Name, events)
				case testworkflowclient.EventTypeUpdate:
					err = s.changeTestWorkflowCronJobs(ctx, obj.Resource.Name, events)
				case testworkflowclient.EventTypeDelete:
					s.removeTestWorkflowCronJobs(obj.Resource.Name)
				}

				if err == nil {
					s.logger.Infow("cron job scheduler: reconciler component: scheduled TestWorkflow to cron jobs", "name", obj.Resource.Name, "error", err)
				} else {
					s.logger.Errorw("cron job scheduler: reconciler component: failed to watch TestWorkflows", "error", err)
				}
			}
			if watcher.Err() != nil {
				s.logger.Errorw("cron job scheduler: reconciler component: failed to watch TestWorkflows", "error", watcher.Err())
			} else {
				includeInitialData = false
			}

			time.Sleep(watcherDelay)
		}
	}
}

// ReconcileTestWorklowTemplatess is watching for test worklow template change and schedule test workflow cron jobs
func (s *Scheduler) ReconcileTestWorkflowTemplates(ctx context.Context) error {
	s.cronService.Start()
	defer s.cronService.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			watcher := s.testWorkflowTemplateClient.WatchUpdates(ctx, s.getEnvironmentId(), false)
			for obj := range watcher.Channel() {
				if obj.Resource == nil || obj.Resource.Spec == nil {
					continue
				}

				if obj.Type != testworkflowtemplateclient.EventTypeCreate &&
					obj.Type != testworkflowtemplateclient.EventTypeUpdate &&
					obj.Type != testworkflowtemplateclient.EventTypeDelete {
					continue
				}

				testWorkflows, err := s.testWorkflowClient.List(ctx, s.getEnvironmentId(), testworkflowclient.ListOptions{})
				if err != nil {
					s.logger.Errorw("cron job schedduler: reconciler component: failed to get TestWorkflows", "error", err)
					continue
				}

				for _, testWorkflow := range testWorkflows {
					if testWorkflow.Spec == nil {
						continue
					}

					found := false
					for _, template := range testWorkflow.Spec.Use {
						if template.Name == obj.Resource.Name {
							found = true
							break
						}
					}

					if !found {
						continue
					}

					events := testWorkflow.Spec.Events
					for _, template := range testWorkflow.Spec.Use {
						testWorkflowTemplate, err := s.testWorkflowTemplateClient.Get(ctx, s.getEnvironmentId(), template.Name)
						if err != nil {
							s.logger.Errorw("cron job schedduler: reconciler component: failed to get TestWorkflowTemplate", "name", template.Name, "error", err)
							continue
						}

						events = append(events, testWorkflowTemplate.Spec.Events...)
					}

					if err = s.changeTestWorkflowCronJobs(ctx, testWorkflow.Name, events); err != nil {
						break
					}
				}

				if err == nil {
					s.logger.Infow("cron job schedduler: reconciler component: scheduled TestWorkflowTemplate to cron jobs", "name", obj.Resource.Name, "error", err)
				} else {
					s.logger.Errorw("cron job schedduler: reconciler omponent: failed to watch TestWorkflowTemplates", "error", err)
				}
			}
			if watcher.Err() != nil {
				s.logger.Errorw("cron job schedduler: reconciler component: failed to watch TestWorkflowTemplates", "error", watcher.Err())
			}

			time.Sleep(watcherDelay)
		}
	}

}

func (s *Scheduler) addTestWorkflowCronJobs(ctx context.Context, testWorkflowName string, events []testkube.TestWorkflowEvent) error {
	for _, event := range events {
		if event.Cronjob != nil {
			var cronJobName string
			cronJobName, err := getHashedMetadataName(event.Cronjob.Cron, event.Cronjob.Config)
			if err != nil {
				return err
			}

			if err = s.addTestWorkflowCronJob(ctx, testWorkflowName, cronJobName, event.Cronjob); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Scheduler) addTestWorkflowCronJob(ctx context.Context, testWorkflowName, cronJobName string,
	cronJob *testkube.TestWorkflowCronJobConfig) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.testWorklows[testWorkflowName]; !ok {
		s.testWorklows[testWorkflowName] = make(map[string]cron.EntryID, 0)
	}

	if _, ok := s.testWorklows[testWorkflowName][cronJobName]; !ok {
		entryID, err := s.cronService.AddJob(cronJob.Cron,
			cron.FuncJob(func() { s.execute(ctx, testWorkflowName, cronJob) }))
		if err != nil {
			return err
		}

		s.testWorklows[testWorkflowName][cronJobName] = entryID
	}

	return nil
}

func (s *Scheduler) changeTestWorkflowCronJobs(ctx context.Context, testWorkflowName string, events []testkube.TestWorkflowEvent) error {
	hasCronJob := false
	currentCronJobNames := make(map[string]struct{})
	for _, event := range events {
		if event.Cronjob != nil {
			hasCronJob = true

			var cronJobName string
			cronJobName, err := getHashedMetadataName(event.Cronjob.Cron, event.Cronjob.Config)
			if err != nil {
				return err
			}

			s.lock.RLock()
			found := false
			if cronJobNames, ok := s.testWorklows[testWorkflowName]; ok {
				if _, ok = cronJobNames[cronJobName]; ok {
					found = true
				}
			}
			s.lock.RUnlock()

			if !found {
				if err = s.addTestWorkflowCronJob(ctx, testWorkflowName, cronJobName, event.Cronjob); err != nil {
					return err
				}
			}

			currentCronJobNames[cronJobName] = struct{}{}
		}
	}

	if !hasCronJob {
		s.removeTestWorkflowCronJobs(testWorkflowName)
		return nil
	}

	s.lock.Lock()
	if cronJobNames, ok := s.testWorklows[testWorkflowName]; ok {
		for cronJobName, entryID := range cronJobNames {
			if _, ok := currentCronJobNames[cronJobName]; !ok {
				s.cronService.Remove(entryID)
				delete(s.testWorklows[testWorkflowName], cronJobName)
			}
		}
	}
	s.lock.Unlock()

	return nil
}

func (s *Scheduler) removeTestWorkflowCronJobs(testWorkflowName string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if cronJobNames, ok := s.testWorklows[testWorkflowName]; ok {
		for _, entryID := range cronJobNames {
			s.cronService.Remove(entryID)
		}

		delete(s.testWorklows, testWorkflowName)
	}
}

func (s *Scheduler) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}

	return ""
}

type configKeyValue struct {
	Key   string
	Value string
}

type configKeyValues []configKeyValue

// getHashedMetadataName returns cron job hashed metadata name
func getHashedMetadataName(schedule string, config map[string]string) (string, error) {
	var slice configKeyValues
	for key, value := range config {
		slice = append(slice, configKeyValue{Key: key, Value: value})
	}

	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Value < slice[j].Value
	})

	data, err := json.Marshal(slice)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%x", schedule, sha256.Sum256(data)), nil
}
