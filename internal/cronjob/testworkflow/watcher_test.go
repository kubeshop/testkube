package testworkflow_test

import (
	"testing"
	"time"

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
