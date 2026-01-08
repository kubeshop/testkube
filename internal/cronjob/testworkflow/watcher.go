// Package testworkflow provides a cronjob schedule watcher for
// TestWorkflows and TestWorkflowTemplates.
package testworkflow

import (
	"context"
	"slices"
	"time"

	"go.uber.org/zap"

	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cronjob"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const watcherDelay = 200 * time.Millisecond

type Watcher struct {
	environmentId              string
	logger                     *zap.SugaredLogger
	testWorkflowClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient
}

func New(logger *zap.SugaredLogger, workflowClient testworkflowclient.TestWorkflowClient, templateClient testworkflowtemplateclient.TestWorkflowTemplateClient, environmentId string) Watcher {
	return Watcher{
		environmentId:              environmentId,
		logger:                     logger,
		testWorkflowClient:         workflowClient,
		testWorkflowTemplateClient: templateClient,
	}
}

func (w Watcher) WatchTestWorkflows(ctx context.Context, configChan chan<- cronjob.Config) {
	includeInitialData := true
	for {
		select {
		case <-ctx.Done():
			return
		default:
			watcher := w.testWorkflowClient.WatchUpdates(ctx, w.environmentId, includeInitialData)
			for obj := range watcher.Channel() {
				if obj.Resource == nil || obj.Resource.Spec == nil {
					continue
				}

				if !slices.Contains([]testworkflowclient.EventType{
					testworkflowclient.EventTypeCreate,
					testworkflowclient.EventTypeUpdate,
					testworkflowclient.EventTypeDelete,
				}, obj.Type) {
					continue
				}

				// In the event of deletion simply send an empty set of schedules.
				// This is expected to cause all schedules for this workflow to be
				// removed.
				if obj.Type == testworkflowclient.EventTypeDelete {
					configChan <- cronjob.Config{
						Workflow: cronjob.Workflow{
							Name:  obj.Resource.GetName(),
							EnvId: w.environmentId,
						},
					}
					continue
				}

				events := obj.Resource.Spec.Events
				for _, template := range obj.Resource.Spec.Use {
					testWorkflowTemplate, err := w.testWorkflowTemplateClient.Get(ctx, w.environmentId, testworkflowresolver.GetInternalTemplateName(template.Name))
					if err != nil {
						w.logger.Errorw("failed to get template for scheduled workflow, ignoring this template and continuing processing of workflow schedule",
							"workflow", obj.Resource.GetName(),
							"template", template.Name,
							"error", err)
						continue
					}

					if testWorkflowTemplate.Spec == nil {
						continue
					}

					events = append(events, testWorkflowTemplate.Spec.Events...)
				}

				var schedules []testkubev1.TestWorkflowCronJobConfig
				for _, event := range events {
					if event.Cronjob != nil {
						schedules = append(schedules, *event.Cronjob)
					}
				}
				configChan <- cronjob.Config{
					Workflow: cronjob.Workflow{
						Name:  obj.Resource.GetName(),
						EnvId: w.environmentId,
					},
					Schedules: schedules,
				}

				w.logger.Infow("seen workflow change",
					"workflow", obj.Resource.GetName(),
					"type", obj.Type,
				)
			}

			if watcher.Err() != nil {
				w.logger.Errorw("failed to watch TestWorkflows", "error", watcher.Err())
			} else {
				includeInitialData = false
			}

			time.Sleep(watcherDelay)
		}
	}
}

func (w Watcher) WatchTestWorkflowTemplates(ctx context.Context, configChan chan<- cronjob.Config) {
	includeInitialData := true
	for {
		select {
		case <-ctx.Done():
			return
		default:
			watcher := w.testWorkflowTemplateClient.WatchUpdates(ctx, w.environmentId, includeInitialData)
			for obj := range watcher.Channel() {
				if obj.Resource == nil || obj.Resource.Spec == nil {
					continue
				}

				if !slices.Contains([]testworkflowtemplateclient.EventType{
					testworkflowtemplateclient.EventTypeCreate,
					testworkflowtemplateclient.EventTypeUpdate,
					testworkflowtemplateclient.EventTypeDelete,
				}, obj.Type) {
					continue
				}

				testWorkflows, err := w.testWorkflowClient.List(ctx, w.environmentId, testworkflowclient.ListOptions{})
				if err != nil {
					w.logger.Errorw("failed to get all workflows to check for scheduled template changes", "error", err)
					continue
				}

				for _, testWorkflow := range testWorkflows {
					// Workflow spec is not processable. Something has gone wrong if this gets hit!
					if testWorkflow.Spec == nil {
						continue
					}
					// If the Workflow does not use the changed template then we can skip processing of it.
					if !slices.ContainsFunc(testWorkflow.Spec.Use, func(template testkubev1.TestWorkflowTemplateRef) bool {
						return testworkflowresolver.GetInternalTemplateName(template.Name) == obj.Resource.Name
					}) {
						continue
					}

					events := testWorkflow.Spec.Events
					for _, template := range testWorkflow.Spec.Use {
						internalName := testworkflowresolver.GetInternalTemplateName(template.Name)
						// If the workflow is using this template, but the template is being deleted,
						// then process the workflow as though this template is not being used.
						if obj.Type == testworkflowtemplateclient.EventTypeDelete && internalName == obj.Resource.Name {
							continue
						}
						testWorkflowTemplate, err := w.testWorkflowTemplateClient.Get(ctx, w.environmentId, internalName)
						if err != nil {
							w.logger.Errorw("failed to get template for scheduled workflow, ignoring this template and continuing processing of workflow schedule",
								"workflow", testWorkflow.GetName(),
								"template", template.Name,
								"error", err)
							continue
						}

						events = append(events, testWorkflowTemplate.Spec.Events...)
					}

					var schedules []testkubev1.TestWorkflowCronJobConfig
					for _, event := range events {
						if event.Cronjob != nil {
							schedules = append(schedules, *event.Cronjob)
						}
					}

					configChan <- cronjob.Config{
						Workflow: cronjob.Workflow{
							Name:  obj.Resource.GetName(),
							EnvId: w.environmentId,
						},
						Schedules: schedules,
					}
				}

				w.logger.Infow("seen workflow template change",
					"template", obj.Resource.GetName(),
					"type", obj.Type,
				)
			}

			if watcher.Err() != nil {
				w.logger.Errorw("failed to watch TestWorkflowTemplates", "error", watcher.Err())
			} else {
				includeInitialData = false
			}

			time.Sleep(watcherDelay)
		}
	}
}
