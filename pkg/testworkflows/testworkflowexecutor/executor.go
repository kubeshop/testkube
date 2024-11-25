package testworkflowexecutor

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

const (
	SaveResultRetryMaxAttempts = 100
	SaveResultRetryBaseDelay   = 300 * time.Millisecond

	SaveLogsRetryMaxAttempts = 10

	ConfigSizeLimit = 3 * 1024 * 1024
)

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
	Control(ctx context.Context, testWorkflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) error
	Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (
		execution testkube.TestWorkflowExecution, err error)
}

type executor struct {
	emitter                      *event.Emitter
	clientSet                    kubernetes.Interface
	repository                   testworkflow.Repository
	output                       testworkflow.OutputRepository
	configMap                    configRepo.Repository
	testWorkflowTemplatesClient  testworkflowsclientv1.TestWorkflowTemplatesInterface
	testWorkflowExecutionsClient testworkflowsclientv1.TestWorkflowExecutionsInterface
	testWorkflowsClient          testworkflowsclientv1.Interface
	metrics                      v1.Metrics
	secretManager                secretmanager.SecretManager
	globalTemplateName           string
	dashboardURI                 string
	runner                       runner.Runner
	proContext                   *config.ProContext
	scheduler                    *scheduler
}

func New(emitter *event.Emitter,
	runner runner.Runner,
	clientSet kubernetes.Interface,
	repository testworkflow.Repository,
	output testworkflow.OutputRepository,
	configMap configRepo.Repository,
	testWorkflowTemplatesClient testworkflowsclientv1.TestWorkflowTemplatesInterface,
	testWorkflowExecutionsClient testworkflowsclientv1.TestWorkflowExecutionsInterface,
	testWorkflowsClient testworkflowsclientv1.Interface,
	metrics v1.Metrics,
	secretManager secretmanager.SecretManager,
	globalTemplateName string,
	dashboardURI string,
	proContext *config.ProContext) TestWorkflowExecutor {
	return &executor{
		emitter:                      emitter,
		clientSet:                    clientSet,
		repository:                   repository,
		output:                       output,
		configMap:                    configMap,
		testWorkflowTemplatesClient:  testWorkflowTemplatesClient,
		testWorkflowExecutionsClient: testWorkflowExecutionsClient,
		testWorkflowsClient:          testWorkflowsClient,
		metrics:                      metrics,
		secretManager:                secretManager,
		globalTemplateName:           globalTemplateName,
		dashboardURI:                 dashboardURI,
		runner:                       runner,
		proContext:                   proContext,
		scheduler: newScheduler(
			testWorkflowsClient,
			testWorkflowTemplatesClient,
			testWorkflowExecutionsClient,
			secretManager,
			repository,
			output,
			runner,
			globalTemplateName,
			emitter,
		),
	}
}

func (e *executor) Control(ctx context.Context, testWorkflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) error {
	return e.runner.Monitor(ctx, execution.Id)
}

func (e *executor) Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (
	testkube.TestWorkflowExecution, error) {
	// Determine the organization/environment
	cloudApiKey := common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY"))
	environmentId := common.GetOr(os.Getenv("TESTKUBE_PRO_ENV_ID"), os.Getenv("TESTKUBE_CLOUD_ENV_ID"))
	organizationId := common.GetOr(os.Getenv("TESTKUBE_PRO_ORG_ID"), os.Getenv("TESTKUBE_CLOUD_ORG_ID"))
	if cloudApiKey == "" {
		organizationId = ""
		environmentId = ""
	}

	executions, err := e.scheduler.Do(ctx, e.dashboardURI, organizationId, environmentId, ScheduleRequest{
		Name:                            workflow.Name,
		Config:                          request.Config,
		ExecutionName:                   request.Name,
		Tags:                            request.Tags,
		DisableWebhooks:                 request.DisableWebhooks,
		TestWorkflowExecutionObjectName: request.TestWorkflowExecutionName,
		RunningContext:                  request.RunningContext,
		ParentExecutionIds:              request.ParentExecutionIds,
	})

	for i := range executions {
		// Start to control the results
		go func() {
			// TODO: Use OpenAPI objects only
			err = e.Control(context.Background(), testworkflowmappers.MapAPIToKube(executions[i].Workflow), &executions[i])
			if err != nil {
				// TODO: Handle fatal error
				//e.handleFatalError(execution, err, time.Time{})
				return
			}
		}()
	}

	if len(executions) > 0 {
		return executions[0], nil
	}
	return testkube.TestWorkflowExecution{}, errors.New("failed to build the execution")
}
