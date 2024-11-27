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
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

const (
	SaveResultRetryMaxAttempts = 100
	SaveResultRetryBaseDelay   = 300 * time.Millisecond

	ConfigSizeLimit = 3 * 1024 * 1024
)

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
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
	scheduler                    *ExecutionScheduler
}

func New(emitter *event.Emitter,
	rnr runner.Runner,
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
		runner:                       rnr,
		proContext:                   proContext,
		scheduler: NewExecutionScheduler(
			testWorkflowsClient,
			testWorkflowTemplatesClient,
			testWorkflowExecutionsClient,
			secretManager,
			repository,
			output,
			func() runner.Runner { return rnr },
			globalTemplateName,
			func() *event.Emitter { return emitter },
		),
	}
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
		go func(id string) {
			// TODO: Use OpenAPI objects only
			err = e.runner.Monitor(context.Background(), id)
			if err != nil {
				// TODO: Handle fatal error
				//e.handleFatalError(execution, err, time.Time{})
				return
			}
		}(executions[i].Id)
	}

	if len(executions) > 0 {
		return executions[0], nil
	}
	return testkube.TestWorkflowExecution{}, errors.New("failed to build the execution")
}
