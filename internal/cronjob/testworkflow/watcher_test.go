package testworkflow_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	cronjobtestworkflow "github.com/kubeshop/testkube/internal/cronjob/testworkflow"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cronjob"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

func Test_ReconcileTestWorkflow(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflow := &testkube.TestWorkflow{
		Name: "testworkflow",
		Spec: &testkube.TestWorkflowSpec{
			Events: []testkube.TestWorkflowEvent{
				{
					Cronjob: &testkube.TestWorkflowCronJobConfig{
						Cron:     "* * * * *",
						Timezone: &testkube.BoxedString{Value: "America/New_York"},
					},
				},
			},
		},
	}

	result := channels.NewWatcher[testworkflowclient.Update]()
	go func() {
		result.Send(testworkflowclient.Update{
			Type:      testworkflowclient.EventTypeCreate,
			Timestamp: time.Now(),
			Resource:  mockTestWorkflow,
		})

		result.Send(testworkflowclient.Update{
			Type:      testworkflowclient.EventTypeUpdate,
			Timestamp: time.Now(),
			Resource:  mockTestWorkflow,
		})

		result.Send(testworkflowclient.Update{
			Type:      testworkflowclient.EventTypeDelete,
			Timestamp: time.Now(),
			Resource:  mockTestWorkflow,
		})

		result.Close(nil)
	}()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowsClient.EXPECT().WatchUpdates(gomock.Any(), "fooenv", gomock.Any()).Return(result).AnyTimes()
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowTemplatesClient.EXPECT().WatchUpdates(gomock.Any(), "fooenv", gomock.Any()).Return(channels.NewWatcher[testworkflowtemplateclient.Update]()).AnyTimes()
	watcher := cronjobtestworkflow.New(log.DefaultLogger, mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, "fooenv")

	mgr := cronjob.NewMockScheduleManager(mockCtrl)
	mgr.EXPECT().
		ReplaceWorkflowSchedules(gomock.Any(), cronjob.Workflow{Name: mockTestWorkflow.Name, EnvId: "fooenv"}, []testkube.TestWorkflowCronJobConfig{*mockTestWorkflow.Spec.Events[0].Cronjob}).
		Return(nil).
		Times(2)
	mgr.EXPECT().
		ReplaceWorkflowSchedules(gomock.Any(), cronjob.Workflow{Name: mockTestWorkflow.Name, EnvId: "fooenv"}, []testkube.TestWorkflowCronJobConfig(nil)).
		Return(nil).
		Times(1)

	svc := cronjob.NewService(
		log.DefaultLogger,
		mgr,
		watcher.WatchTestWorkflows,
		watcher.WatchTestWorkflowTemplates,
	)

	go svc.Run(t.Context())

	// Wait for the mock to be satified.
	// Will cause test to stall forever if the test is broken...thats probably fine, better than a sleep.
	for !mockCtrl.Satisfied() {
		continue
	}
}

func Test_WatchTestWorkflowTemplates_ReschedulesTheWorkflowsUsingTheTemplate(t *testing.T) {
	nightly := testkube.TestWorkflowCronJobConfig{Cron: "0 2 * * *"}
	own := testkube.TestWorkflowCronJobConfig{Cron: "*/30 * * * *"}
	template := &testkube.TestWorkflowTemplate{
		Name: "nightly",
		Spec: &testkube.TestWorkflowTemplateSpec{
			Events: []testkube.TestWorkflowEvent{{Cronjob: &nightly}},
		},
	}
	workflows := []testkube.TestWorkflow{
		{
			Name: "smoke",
			Spec: &testkube.TestWorkflowSpec{
				Use:    []testkube.TestWorkflowTemplateRef{{Name: "nightly"}},
				Events: []testkube.TestWorkflowEvent{{Cronjob: &own}},
			},
		},
		{
			Name: "api",
			Spec: &testkube.TestWorkflowSpec{Use: []testkube.TestWorkflowTemplateRef{{Name: "nightly"}}},
		},
		{
			Name: "unrelated",
			Spec: &testkube.TestWorkflowSpec{Events: []testkube.TestWorkflowEvent{{Cronjob: &own}}},
		},
	}

	tests := map[string]struct {
		event testworkflowtemplateclient.EventType
		want  []cronjob.Config
	}{
		"template updated": {
			event: testworkflowtemplateclient.EventTypeUpdate,
			want: []cronjob.Config{
				{Workflow: cronjob.Workflow{Name: "smoke", EnvId: "fooenv"}, Schedules: []testkube.TestWorkflowCronJobConfig{own, nightly}},
				{Workflow: cronjob.Workflow{Name: "api", EnvId: "fooenv"}, Schedules: []testkube.TestWorkflowCronJobConfig{nightly}},
			},
		},
		"template deleted": {
			event: testworkflowtemplateclient.EventTypeDelete,
			want: []cronjob.Config{
				{Workflow: cronjob.Workflow{Name: "smoke", EnvId: "fooenv"}, Schedules: []testkube.TestWorkflowCronJobConfig{own}},
				{Workflow: cronjob.Workflow{Name: "api", EnvId: "fooenv"}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)

			updates := channels.NewWatcher[testworkflowtemplateclient.Update]()
			go func() {
				updates.Send(testworkflowtemplateclient.Update{Type: tc.event, Timestamp: time.Now(), Resource: template})
				updates.Close(nil)
			}()

			templatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
			templatesClient.EXPECT().WatchUpdates(gomock.Any(), "fooenv", gomock.Any()).Return(updates).AnyTimes()
			templatesClient.EXPECT().Get(gomock.Any(), "fooenv", "nightly").Return(template, nil).AnyTimes()
			workflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
			workflowsClient.EXPECT().List(gomock.Any(), "fooenv", gomock.Any()).Return(workflows, nil).AnyTimes()
			watcher := cronjobtestworkflow.New(log.DefaultLogger, workflowsClient, templatesClient, "fooenv")

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			configs := make(chan cronjob.Config, len(workflows))
			done := make(chan struct{})
			go func() {
				defer close(done)
				watcher.WatchTestWorkflowTemplates(ctx, configs)
			}()

			var got []cronjob.Config
			for len(got) < len(tc.want) {
				select {
				case config := <-configs:
					got = append(got, config)
				case <-time.After(5 * time.Second):
					require.FailNow(t, "timed out waiting for the workflow schedules", "got: %+v", got)
				}
			}
			cancel()
			<-done

			assert.Equal(t, tc.want, got)
			assert.Empty(t, configs, "only the workflows that use the template are rescheduled")
		})
	}
}
