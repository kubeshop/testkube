package testworkflowexecutor_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

var (
	forbid = testkube.FORBID_TestWorkflowConcurrencyPolicy
)

func resolveWorkflow(t *testing.T, w testkube.TestWorkflow) []byte {
	t.Helper()

	ret, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	return ret
}

func TestSchedulerConcurrencyPolicy(t *testing.T) {
	tests := map[string]struct {
		req             *cloud.ScheduleRequest
		envID           string
		expectScheduled bool
		prepare         func(*testing.T, *testworkflowclient.MockTestWorkflowClient, *testworkflow.MockRepository, *testworkflowexecutor.MockSensitiveDataHandler)
	}{
		string(testworkflowsv1.AllowConcurrent): {
			req: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{
					{
						Selector: &cloud.ScheduleResourceSelector{
							Name: "foo",
						},
					},
				},
				ResolvedWorkflow: resolveWorkflow(t, testkube.TestWorkflow{
					Name: "bar",
				}),
			},
			envID:           "env",
			expectScheduled: true,
			prepare: func(t *testing.T, twc *testworkflowclient.MockTestWorkflowClient, repo *testworkflow.MockRepository, secrets *testworkflowexecutor.MockSensitiveDataHandler) {
				t.Helper()

				twc.EXPECT().
					Get(gomock.Any(), "env", "foo").
					Return(&testkube.TestWorkflow{
						Spec: &testkube.TestWorkflowSpec{
							Execution: &testkube.TestWorkflowTagSchema{},
						},
					}, nil)

				repo.EXPECT().
					GetNextExecutionNumber(gomock.Any(), "bar").
					Return(int32(123), nil)
				repo.EXPECT().
					GetByNameAndTestWorkflow(gomock.Any(), "bar-123", "bar")

				secrets.EXPECT().
					Process(gomock.Any())
				repo.EXPECT().
					Insert(gomock.Any(), gomock.Any())
			},
		},
		string(testworkflowsv1.ForbidConcurrent) + " running": {
			req: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{
					{
						Selector: &cloud.ScheduleResourceSelector{
							Name: "foo",
						},
					},
				},
				ResolvedWorkflow: resolveWorkflow(t, testkube.TestWorkflow{
					Name: "bar",
					Spec: &testkube.TestWorkflowSpec{
						ConcurrencyPolicy: &forbid,
					},
				}),
			},
			envID:           "env",
			expectScheduled: false,
			prepare: func(t *testing.T, twc *testworkflowclient.MockTestWorkflowClient, repo *testworkflow.MockRepository, secrets *testworkflowexecutor.MockSensitiveDataHandler) {
				t.Helper()

				repo.EXPECT().
					GetRunning(gomock.Any()).
					Return([]testkube.TestWorkflowExecution{
						{
							Name: "bar-123",
							Workflow: &testkube.TestWorkflow{
								Name: "bar",
							},
						},
					}, nil)
			},
		},
		string(testworkflowsv1.ForbidConcurrent) + " none": {
			req: &cloud.ScheduleRequest{
				Executions: []*cloud.ScheduleExecution{
					{
						Selector: &cloud.ScheduleResourceSelector{
							Name: "foo",
						},
					},
				},
				ResolvedWorkflow: resolveWorkflow(t, testkube.TestWorkflow{
					Name: "bar",
					Spec: &testkube.TestWorkflowSpec{
						ConcurrencyPolicy: &forbid,
					},
				}),
			},
			envID:           "env",
			expectScheduled: true,
			prepare: func(t *testing.T, twc *testworkflowclient.MockTestWorkflowClient, repo *testworkflow.MockRepository, secrets *testworkflowexecutor.MockSensitiveDataHandler) {
				t.Helper()

				repo.EXPECT().
					GetRunning(gomock.Any()).
					Return([]testkube.TestWorkflowExecution{}, nil)

				twc.EXPECT().
					Get(gomock.Any(), "env", "foo").
					Return(&testkube.TestWorkflow{
						Spec: &testkube.TestWorkflowSpec{
							Execution: &testkube.TestWorkflowTagSchema{},
						},
					}, nil)

				repo.EXPECT().
					GetNextExecutionNumber(gomock.Any(), "bar").
					Return(int32(123), nil)
				repo.EXPECT().
					GetByNameAndTestWorkflow(gomock.Any(), "bar-123", "bar")

				secrets.EXPECT().
					Process(gomock.Any())
				repo.EXPECT().
					Insert(gomock.Any(), gomock.Any())
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			var (
				grpcClient cloud.TestKubeCloudAPIClient

				testWorkflowsClient         = testworkflowclient.NewMockTestWorkflowClient(ctrl)
				testWorkflowTemplatesClient = testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(ctrl)
				resultsRepository           = testworkflow.NewMockRepository(ctrl)
				outputRepository            = testworkflow.NewMockOutputRepository(ctrl)

				getRunners = func(environmentId string, target *cloud.ExecutionTarget) ([]map[string]string, error) {
					return make([]map[string]string, 0), nil
				}
				getEnvSlug = func(s string) string { return s }

				globalTemplateName       = ""
				globalTemplateInlineYaml = ""
				organizationId           = ""
				organizationSlug         = ""
				defaultEnvironmentId     = ""
				agentId                  = ""
				grpcApiToken             = ""
				newArchitectureEnabled   = true

				secrets = testworkflowexecutor.NewMockSensitiveDataHandler(ctrl)
			)

			if test.prepare != nil {
				test.prepare(t, testWorkflowsClient, resultsRepository, secrets)
			}

			scheduler := testworkflowexecutor.NewScheduler(
				testWorkflowsClient,
				testWorkflowTemplatesClient,
				resultsRepository,
				outputRepository,
				getRunners,
				globalTemplateName,
				globalTemplateInlineYaml,
				organizationId,
				organizationSlug,
				defaultEnvironmentId,
				getEnvSlug,
				agentId,
				grpcClient,
				grpcApiToken,
				newArchitectureEnabled,
			)

			ch, err := scheduler.Schedule(context.Background(), secrets, test.envID, test.req)
			if err != nil {
				if !test.expectScheduled {
					// Success!
					return
				}
				t.Fatal(err)
			}
			for {
				select {
				case _, ok := <-ch:
					assert.Equal(t, test.expectScheduled, ok)
					return
				}
			}
		})
	}
}
