package cronjob

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

// ReconcileTestWorklows is watching for test workflow change and schedule test workflow cron jobs
func (s *Scheduler) ReconcileTestWorkflows(ctx context.Context) error {
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
					testWorkflowTemplate, err := s.testWorkflowTemplateClient.Get(ctx, s.getEnvironmentId(), testworkflowresolver.GetInternalTemplateName(template.Name))
					if err != nil {
						s.logger.Errorw("cron job scheduler: reconciler component: failed to get TestWorkflowTemplate", "name", template.Name, "error", err)
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

				if err != nil {
					s.logger.Errorw("cron job scheduler: reconciler component: failed to watch TestWorkflows",
						"error", err,
						"resource", obj.Resource.Name,
						"event", obj.Type,
					)
					continue
				}

				s.logger.Infow("cron job scheduler: reconciler component: scheduled TestWorkflow to cron jobs", "name", obj.Resource.Name)
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
					s.logger.Errorw("cron job scheduler: reconciler component: failed to get TestWorkflows", "error", err)
					continue
				}

				for _, testWorkflow := range testWorkflows {
					if testWorkflow.Spec == nil {
						continue
					}

					found := false
					for _, template := range testWorkflow.Spec.Use {
						if testworkflowresolver.GetInternalTemplateName(template.Name) == obj.Resource.Name {
							found = true
							break
						}
					}

					if !found {
						continue
					}

					events := testWorkflow.Spec.Events
					for _, template := range testWorkflow.Spec.Use {
						testWorkflowTemplate, err := s.testWorkflowTemplateClient.Get(ctx, s.getEnvironmentId(), testworkflowresolver.GetInternalTemplateName(template.Name))
						if err != nil {
							s.logger.Errorw("cron job scheduler: reconciler component: failed to get TestWorkflowTemplate", "name", template.Name, "error", err)
							continue
						}

						events = append(events, testWorkflowTemplate.Spec.Events...)
					}

					if err = s.changeTestWorkflowCronJobs(ctx, testWorkflow.Name, events); err != nil {
						break
					}
				}

				if err == nil {
					s.logger.Infow("cron job scheduler: reconciler component: scheduled TestWorkflowTemplate to cron jobs", "name", obj.Resource.Name)
				} else {
					s.logger.Errorw("cron job scheduler: reconciler omponent: failed to watch TestWorkflowTemplates", "error", err)
				}
			}

			if watcher.Err() != nil {
				s.logger.Errorw("cron job scheduler: reconciler component: failed to watch TestWorkflowTemplates", "error", watcher.Err())
			}

			time.Sleep(watcherDelay)
		}
	}

}

func (s *Scheduler) addTestWorkflowCronJobs(ctx context.Context, testWorkflowName string, events []testkube.TestWorkflowEvent) error {
	for _, event := range events {
		if event.Cronjob != nil {
			var cronJobName string
			cronJobName, err := getTestWorkflowHashedMetadataName(event.Cronjob)
			if err != nil {
				return err
			}

			if err = s.addTestWorkflowCronJob(ctx, testWorkflowName, cronJobName, event.Cronjob); err != nil {
				return fmt.Errorf("adding new cron job %q for workflow %q: %w", cronJobName, testWorkflowName, err)
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
		cronName := cronJob.Cron
		if cronJob.Timezone != nil {
			cronName = fmt.Sprintf("CRON_TZ=%s %s", cronJob.Timezone.Value, cronJob.Cron)
		}

		entryID, err := s.cronService.AddJob(cronName,
			cron.FuncJob(func() { s.executeTestWorkflow(ctx, testWorkflowName, cronJob) }))
		if err != nil {
			return fmt.Errorf("adding cron %q for workflow %q to service: %w", cronJobName, testWorkflowName, err)
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
			cronJobName, err := getTestWorkflowHashedMetadataName(event.Cronjob)
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
					return fmt.Errorf("add missing cron job %q for workflow %q: %w", cronJobName, testWorkflowName, err)
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

type configKeyValue struct {
	Key   string
	Value string
}

type configKeyValues []configKeyValue

// getTestWorkflowHashedMetadataName returns cron job hashed metadata name
func getTestWorkflowHashedMetadataName(cronJob *testkube.TestWorkflowCronJobConfig) (string, error) {
	var slice configKeyValues
	for key, value := range cronJob.Config {
		slice = append(slice, configKeyValue{Key: key, Value: value})
	}

	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Key < slice[j].Key
	})

	data, err := json.Marshal(slice)
	if err != nil {
		return "", err
	}

	cronName := cronJob.Cron
	if cronJob.Timezone != nil {
		cronName = fmt.Sprintf("%s %s", cronJob.Timezone.Value, cronJob.Cron)
	}

	return fmt.Sprintf("%s-%x", cronName, sha256.Sum256(data)), nil
}
