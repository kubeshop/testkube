package testworkflowexecutor

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"io"
	"math"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
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
	Execute(ctx context.Context, environmentId string, req *cloud.ScheduleRequest) TestWorkflowExecutionStream
	Start(environmentId string, execution *testkube.TestWorkflowExecution, secrets map[string]map[string]string) error
}

type executor struct {
	grpcClient           cloud.TestKubeCloudAPIClient
	apiKey               string
	cdEventsTarget       string
	organizationId       string
	defaultEnvironmentId string
	agentId              string

	emitter                event.Interface
	metrics                v1.Metrics
	secretManager          secretmanager.SecretManager
	dashboardURI           string
	runner                 runner.RunnerExecute
	scheduler              Scheduler
	featureNewArchitecture bool
}

func New(
	grpcClient cloud.TestKubeCloudAPIClient,
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
	organizationSlug string,
	defaultEnvironmentId string,
	getEnvSlug func(string) string,
	agentId string,
	featureNewArchitecture bool) TestWorkflowExecutor {
	return &executor{
		agentId:                agentId,
		grpcClient:             grpcClient,
		apiKey:                 apiKey,
		cdEventsTarget:         cdEventsTarget,
		emitter:                emitter,
		metrics:                metrics,
		secretManager:          secretManager,
		dashboardURI:           dashboardURI,
		runner:                 runner,
		organizationId:         organizationId,
		defaultEnvironmentId:   defaultEnvironmentId,
		featureNewArchitecture: featureNewArchitecture,
		scheduler: NewScheduler(
			testWorkflowsClient,
			testWorkflowTemplatesClient,
			repository,
			output,
			func(_ string, _ *cloud.ExecutionTarget) ([]map[string]string, error) {
				return nil, nil
			},
			globalTemplateName,
			"",
			organizationId,
			organizationSlug,
			defaultEnvironmentId,
			getEnvSlug,
			agentId,
			grpcClient,
			apiKey,
			featureNewArchitecture,
		),
	}
}

func (e *executor) Execute(ctx context.Context, environmentId string, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	if environmentId == "" {
		environmentId = e.defaultEnvironmentId
	}
	if !e.featureNewArchitecture {
		// NOTE: only on the old arcitecture
		return e.executeDirect(ctx, environmentId, req)
	}
	return e.execute(ctx, environmentId, req)
}

func (e *executor) execute(ctx context.Context, environmentId string, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	ch := make(chan *testkube.TestWorkflowExecution)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	ctx = agentclient.AddAPIKeyMeta(ctx, e.apiKey)
	ctx = metadata.AppendToOutgoingContext(ctx, "environment-id", environmentId)
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

// TODO: what is executeDirect?
func (e *executor) executeDirect(ctx context.Context, environmentId string, req *cloud.ScheduleRequest) TestWorkflowExecutionStream {
	// Prepare dependencies
	sensitiveDataHandler := NewSecretHandler(e.secretManager)

	// Schedule execution
	ch, err := e.scheduler.Schedule(ctx, sensitiveDataHandler, environmentId, req)
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
			// TODO: canceled event is missing here
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
			execution.RunnerId = e.agentId
			if environmentId == "" {
				execution.RunnerId = "oss"
			}
			execution.AssignedAt = time.Now()

			// Start the execution
			err = e.Start(environmentId, execution, sensitiveDataHandler.Get(execution.Id))
			if err != nil {
				log2.DefaultLogger.Errorw("failed to start execution", "executionId", execution.Id, "error", err)
			}
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
	// TODO: and what is this runner exactly?
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
	// TODO: CriticalError should use Finish if possible
	if err != nil {
		// NOTE: this is being called by the agent, but the scheduler is passed in a grpc interface for the workflows repository so it makes calls to the
		err2 := e.scheduler.CriticalError(execution, "Failed to run execution", err)
		err = errors2.Join(err, err2)
		if err != nil {
			log2.DefaultLogger.Errorw("failed to run and update execution", "executionId", execution.Id, "error", err)
		}
		e.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
		return nil
	}

	// Apply the known d ata to temporary object.
	execution.Namespace = result.Namespace
	execution.Signature = result.Signature
	// TODO: Don't emit?
	if err = e.scheduler.Start(execution); err != nil {
		log2.DefaultLogger.Errorw("failed to mark execution as initialized", "executionId", execution.Id, "error", err)
	}
	return nil
}
