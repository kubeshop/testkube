// Package testworkflow provides a cronjob schedule watcher for
// TestWorkflows and TestWorkflowTemplates.
package testworkflow

import (
	"context"
	"slices"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/cronjob"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

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
	watcher := w.testWorkflowClient.WatchUpdates(ctx, w.environmentId, true)
	if watcher.Err() != nil {
		w.logger.Errorw("failed to watch TestWorkflows",
			"error", watcher.Err())
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case obj := <-watcher.Channel():
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

			events := obj.Resource.Spec.Events
			for _, template := range obj.Resource.Spec.Use {
				testWorkflowTemplate, err := w.testWorkflowTemplateClient.Get(ctx, w.environmentId, testworkflowresolver.GetInternalTemplateName(template.Name))
				if err != nil {
					w.logger.Errorw("failed to get template for scheduled workflow, ignoring this template and continuing processing of workflow schedule",
						"workflow", obj.Resource.Name,
						"template", template.Name,
						"error", err)
					continue
				}

				if testWorkflowTemplate.Spec == nil {
					continue
				}

				events = append(events, testWorkflowTemplate.Spec.Events...)
			}

			for _, event := range events {
				if event.Cronjob != nil {
					configChan <- cronjob.Config{
						WorkflowName: obj.Resource.Name,
						CronJob:      *event.Cronjob,
						Remove:       obj.Type == testworkflowclient.EventTypeDelete,
					}
				}
			}

			w.logger.Infow("seen workflow change",
				"workflow", obj.Resource.Name,
			)
		}
	}
}

func (w Watcher) WatchTestWorkflowTemplates(ctx context.Context, configChan chan<- cronjob.Config) {
	watcher := w.testWorkflowTemplateClient.WatchUpdates(ctx, w.environmentId, true)
	if watcher.Err() != nil {
		w.logger.Errorw("failed to watch TestWorkflowTemplates",
			"error", watcher.Err())
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case obj := <-watcher.Channel():
			if obj.Resource == nil || obj.Resource.Spec == nil || len(obj.Resource.Spec.Events) == 0 {
				// Not a schedulable template so no additional processing is required.
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
				if testWorkflow.Spec == nil {
					continue
				}

				events := testWorkflow.Spec.Events
				for _, template := range testWorkflow.Spec.Use {
					internalName := testworkflowresolver.GetInternalTemplateName(template.Name)
					if internalName != obj.Resource.Name {
						continue
					}
					testWorkflowTemplate, err := w.testWorkflowTemplateClient.Get(ctx, w.environmentId, internalName)
					if err != nil {
						w.logger.Errorw("failed to get template for scheduled workflow, ignoring this template and continuing processing of workflow schedule",
							"workflow", testWorkflow.Name,
							"template", template.Name,
							"error", err)
						continue
					}

					events = append(events, testWorkflowTemplate.Spec.Events...)
				}

				for _, event := range events {
					if event.Cronjob != nil {
						configChan <- cronjob.Config{
							WorkflowName: testWorkflow.Name,
							CronJob:      *event.Cronjob,
							Remove:       obj.Type == testworkflowtemplateclient.EventTypeDelete,
						}
					}
				}
			}

			w.logger.Infow("seen workflow template change",
				"template", obj.Resource.Name,
			)
		}
	}
}
