package testworkflowexecutor

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"io"
	"math"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event"
	log2 "github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

const (
	ConfigSizeLimit = 3 * 1024 * 1024
)

type TestWorkflowExecutionStream Stream[*testkube.TestWorkflowExecution]

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
	Execute(ctx context.Context, req *cloud.ScheduleRequest) TestWorkflowExecutionStream
	Start(environmentId string, execution *testkube.TestWorkflowExecution, secrets map[string]map[string]string) error
}

type executor struct {
	grpcClient           cloud.TestKubeCloudAPIClient
	apiKey               string
	cdEventsTarget       string
	organizationId       string
	defaultEnvironmentId string

	emitter              event.Interface
	metrics              v1.Metrics
	secretManager        secretmanager.SecretManager
	dashboardURI         string
	runner               runner.RunnerExecute
	proContext           *config.ProContext
	scheduler            Scheduler
	featureNewExecutions bool
}

func New(
	grpClient cloud.TestKubeCloudAPIClient,
	apiKey string,
	cdEventsTarget string,
	emitter event.Interface,
	runner runner.RunnerExecute,
	repository testworkflow.Repository,
	output testworkflow.OutputRepository,
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	metrics v1.Metrics,
	secretManager secretmanager.SecretManager,
	globalTemplateName string,
	dashboardURI string,
	organizationId string,
	defaultEnvironmentId string,
	featureNewExecutions bool) TestWorkflowExecutor {
	return &executor{
		grpcClient:           grpClient,
		apiKey:               apiKey,
		cdEventsTarget:       cdEventsTarget,
		emitter:              emitter,
		metrics:              metrics,
		secretManager:        secretManager,
		dashboardURI:         dashboardURI,
		runner:               runner,
		organizationId:       organizationId,
		defaultEnvironmentId: defaultEnvironmentId,
		featureNewExecutions: featureNewExecutions,
		scheduler: NewScheduler(
			testWorkflowsClient,
			testWorkflowTemplatesClient,
			repository,
			output,
			globalTemplateName,
			organizationId,
			defaultEnvironmentId,
		),
	}
}

func (e *executor) isDirect() bool {
	return e.proContext == nil || !e.proContext.NewExecutions
}

func (e *executor) Execute(ctx context.Context, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	if req != nil {
		req = common.Ptr(*req) // nolint:govet
		if req.EnvironmentId == "" {
			req.EnvironmentId = e.defaultEnvironmentId
		}
	}
	if e.isDirect() {
		return e.executeDirect(ctx, req)
	}
	return e.execute(ctx, req)
}

func (e *executor) execute(ctx context.Context, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	ch := make(chan *testkube.TestWorkflowExecution)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	ctx = agentclient.AddAPIKeyMeta(ctx, e.apiKey)
	resp, err := e.grpcClient.ScheduleExecution(ctx, req, opts...)
	resultStream := NewStream(ch)
	if err != nil {
		close(ch)
		resultStream.addError(err)
		return resultStream
	}
	go func() {
		defer close(ch)
		var item *cloud.ScheduleResponse
		for {
			item, err = resp.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					resultStream.addError(err)
				}
				break
			}
			var r testkube.TestWorkflowExecution
			err = json.Unmarshal(item.Execution, &r)
			if err != nil {
				resultStream.addError(err)
				break
			}
			ch <- &r
		}
	}()
	return resultStream
}

func (e *executor) executeDirect(ctx context.Context, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	// Prepare dependencies
	sensitiveDataHandler := NewSecretHandler(e.secretManager)

	// Schedule execution
	ch, err := e.scheduler.Schedule(ctx, sensitiveDataHandler, req)
	if err != nil {
		resultStream := NewStream(ch)
		resultStream.addError(err)
		return resultStream
	}

	ch2 := make(chan *testkube.TestWorkflowExecution, 1)
	resultStream := NewStream(ch2)
	go func() {
		defer close(ch2)
		for execution := range ch {
			e.emitter.Notify(testkube.NewEventQueueTestWorkflow(execution))

			// Send the data
			ch2 <- execution.Clone()

			// Send information about start
			e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution))

			// Finish early if it's immediately known to finish
			if execution.Result.IsFinished() {
				if execution.Result.IsAborted() {
					e.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
				} else if execution.Result.IsFailed() {
					e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
				} else {
					e.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
				}
				continue
			}

			// Set the runner execution to environment ID as it's a legacy Agent
			execution.RunnerId = req.EnvironmentId
			if req.EnvironmentId == "" {
				execution.RunnerId = "oss"
			}

			// Start the execution
			_ = e.Start(req.EnvironmentId, execution, sensitiveDataHandler.Get(execution.Id))
		}
	}()

	return resultStream
}

// TODO: Delete?
func (e *executor) Start(environmentId string, execution *testkube.TestWorkflowExecution, secrets map[string]map[string]string) error {
	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		DashboardUrl:   e.dashboardURI,
		CDEventsTarget: e.cdEventsTarget,
	}

	parentIds := ""
	if execution.RunningContext != nil && execution.RunningContext.Actor != nil {
		parentIds = execution.RunningContext.Actor.ExecutionPath
	}
	result, err := e.runner.Execute(executionworkertypes.ExecuteRequest{
		Execution: testworkflowconfig.ExecutionConfig{
			Id:              execution.Id,
			GroupId:         execution.GroupId,
			Name:            execution.Name,
			Number:          execution.Number,
			ScheduledAt:     execution.ScheduledAt,
			DisableWebhooks: execution.DisableWebhooks,
			Debug:           false,
			OrganizationId:  e.organizationId,
			EnvironmentId:   environmentId,
			ParentIds:       parentIds,
		},
		Secrets:      secrets,
		Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*execution.ResolvedWorkflow),
		ControlPlane: controlPlaneConfig,
	})

	// TODO: define "revoke" error by runner (?)
	if err != nil {
		err2 := e.scheduler.CriticalError(execution, "Failed to run execution", err)
		err = errors2.Join(err, err2)
		if err != nil {
			log2.DefaultLogger.Errorw("failed to run and update execution", "executionId", execution.Id, "error", err)
		}
		// TODO: Don't emit?
		//e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution)) // TODO: delete - sent from Cloud
		e.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
		return nil
	}

	// Inform about execution start
	//e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution)) // TODO: delete - sent from Cloud

	// Apply the known d ata to temporary object.
	execution.Namespace = result.Namespace
	execution.Signature = result.Signature
	// TODO: Don't emit?
	if err = e.scheduler.Start(execution); err != nil {
		log2.DefaultLogger.Errorw("failed to mark execution as initialized", "executionId", execution.Id, "error", err)
	}
	return nil
}
