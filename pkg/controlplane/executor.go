package controlplane

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/internal/common"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event"
	log2 "github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type Executor struct {
	direct         *bool
	directMu       sync.Mutex
	scheduler      *scheduler
	eventEmitter   event.Interface
	runner         runner.Runner
	client         cloud.TestKubeCloudAPIClient
	secretManager  secretmanager.SecretManager
	apiKey         string
	dashboardURI   string
	cdEventsTarget string
}

func NewExecutor(
	scheduler *scheduler,
	client cloud.TestKubeCloudAPIClient,
	eventEmitter event.Interface,
	runner runner.Runner,
	secretManager secretmanager.SecretManager,
	apiKey string,
	dashboardURI string,
	cdEventsTarget string,
) *Executor {
	return &Executor{
		scheduler:      scheduler,
		client:         client,
		apiKey:         apiKey,
		eventEmitter:   eventEmitter,
		secretManager:  secretManager,
		runner:         runner,
		dashboardURI:   dashboardURI,
		cdEventsTarget: cdEventsTarget,
	}
}

func (e *Executor) isDirect() bool {
	e.directMu.Lock()
	defer e.directMu.Unlock()
	if e.direct == nil {
		if e.client == nil {
			e.direct = common.Ptr(true)
			return true
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ctx = agentclient.AddAPIKeyMeta(ctx, e.apiKey)
		proContext, _ := e.client.GetProContext(ctx, &emptypb.Empty{})
		if proContext != nil {
			e.direct = common.Ptr(!capabilities.Enabled(proContext.Capabilities, capabilities.CapabilityNewExecutions))
		}
	}
	return *e.direct
}

func (e *Executor) Execute(ctx context.Context, req *cloud.ScheduleRequest) (<-chan *testkube.TestWorkflowExecution, error) {
	if e.isDirect() {
		return e.executeDirect(ctx, req)
	}
	return e.execute(ctx, req)
}

func (e *Executor) execute(ctx context.Context, req *cloud.ScheduleRequest) (<-chan *testkube.TestWorkflowExecution, error) {
	ch := make(chan *testkube.TestWorkflowExecution)
	resp, err := e.client.ScheduleExecution(ctx, req)
	if err != nil {
		close(ch)
		return ch, err
	}
	go func() {
		errs := make([]error, 0)
		var item *cloud.ScheduleResponse
		for {
			item, err = resp.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errs = append(errs, err)
				}
				break
			}
			var r testkube.TestWorkflowExecution
			err = json.Unmarshal(item.Execution, &r)
			if err != nil {
				errs = append(errs, err)
				break
			}
			ch <- &r
		}
	}()
	return ch, nil
}

func (e *Executor) executeDirect(ctx context.Context, req *cloud.ScheduleRequest) (<-chan *testkube.TestWorkflowExecution, error) {
	// Prepare dependencies
	sensitiveDataHandler := NewSecretHandler(e.secretManager)

	// Schedule execution
	ch, err := e.scheduler.Schedule(ctx, sensitiveDataHandler, req)
	if err != nil {
		return ch, err
	}

	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		DashboardUrl:   e.dashboardURI,
		CDEventsTarget: e.cdEventsTarget,
	}

	ch2 := make(chan *testkube.TestWorkflowExecution, 1)
	go func() {
		defer close(ch2)
		for execution := range ch {
			e.eventEmitter.Notify(testkube.NewEventQueueTestWorkflow(execution))

			// Send the data
			ch2 <- execution.Clone()

			// Finish early if it's immediately known to finish
			if execution.Result.IsFinished() {
				e.eventEmitter.Notify(testkube.NewEventStartTestWorkflow(execution))
				if execution.Result.IsAborted() {
					e.eventEmitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
				} else if execution.Result.IsFailed() {
					e.eventEmitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
				} else {
					e.eventEmitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
				}
				continue
			}

			// Start the execution
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
					OrganizationId:  "",
					EnvironmentId:   "",
					ParentIds:       parentIds,
				},
				Secrets:      sensitiveDataHandler.Get(execution.Id),
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
				e.eventEmitter.Notify(testkube.NewEventStartTestWorkflow(execution))
				e.eventEmitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
				continue
			}

			// Inform about execution start
			e.eventEmitter.Notify(testkube.NewEventStartTestWorkflow(execution))

			// Apply the known data to temporary object.
			execution.Namespace = result.Namespace
			execution.Signature = result.Signature
			execution.RunnerId = ""
			if err = e.scheduler.Start(execution); err != nil {
				log2.DefaultLogger.Errorw("failed to mark execution as initialized", "executionId", execution.Id, "error", err)
			}
		}
	}()

	return ch2, nil
}
