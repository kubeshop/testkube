package runner

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	testworkflow2 "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
)

const (
	GetNotificationsRetryCount = 10
	GetNotificationsRetryDelay = 500 * time.Millisecond

	SaveEndResultRetryCount     = 100
	SaveEndResultRetryBaseDelay = 500 * time.Millisecond
)

type RunnerExecute interface {
	Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
}

//go:generate mockgen -destination=./mock_runner.go -package=runner "github.com/kubeshop/testkube/pkg/runner" Runner
type Runner interface {
	RunnerExecute
	Monitor(ctx context.Context, organizationId, environmentId, id string) error
	Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher
	Pause(id string) error
	Resume(id string) error
	Abort(id string) error
}

type runner struct {
	id                   string
	worker               executionworkertypes.Worker
	outputRepository     testworkflow.OutputRepository
	executionsRepository testworkflow.Repository
	grpcClient           cloud.TestKubeCloudAPIClient
	grpcApiToken         string
	configRepository     configRepo.Repository
	emitter              event.Interface
	metrics              metrics.Metrics
	proContext           config.ProContext // TODO: Include Agent ID in pro context
	dashboardURI         string
	storageSkipVerify    bool
	newExecutionsEnabled bool // TODO: ag.featureNewExecutions && ag.proContext.NewExecutions

	watching sync.Map
}

// TODO: ABORT/RESUME/PAUSE ETC CALLS
func New(
	id string,
	worker executionworkertypes.Worker,
	outputRepository testworkflow.OutputRepository,
	executionsRepository testworkflow.Repository,
	configRepository configRepo.Repository,
	grpcClient cloud.TestKubeCloudAPIClient,
	grpcApiToken string,
	emitter event.Interface,
	metrics metrics.Metrics,
	proContext config.ProContext,
	dashboardURI string,
	storageSkipVerify bool,
	newExecutionsEnabled bool,
) Runner {
	return &runner{
		id:                   id,
		worker:               worker,
		outputRepository:     outputRepository,
		executionsRepository: executionsRepository,
		configRepository:     configRepository,
		grpcClient:           grpcClient,
		grpcApiToken:         grpcApiToken,
		emitter:              emitter,
		metrics:              metrics,
		proContext:           proContext,
		dashboardURI:         dashboardURI,
		storageSkipVerify:    storageSkipVerify,
		newExecutionsEnabled: newExecutionsEnabled,
	}
}

func (r *runner) getLogPresigner(environmentId string) LogPresigner {
	if r.newExecutionsEnabled {
		return &newLogPresigner{
			organizationId: r.proContext.OrgID,
			environmentId:  environmentId,
			agentId:        r.id,
			grpcClient:     r.grpcClient,
			grpcApiToken:   r.grpcApiToken,
		}
	}
	return r.outputRepository
}

func (r *runner) monitor(ctx context.Context, organizationId string, environmentId string, execution testkube.TestWorkflowExecution) error {
	defer r.watching.Delete(execution.Id)

	var notifications executionworkertypes.NotificationsWatcher
	for i := 0; i < GetNotificationsRetryCount; i++ {
		notifications = r.worker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{})
		if notifications.Err() == nil {
			break
		}
		if errors.Is(notifications.Err(), registry.ErrResourceNotFound) {
			// TODO: should it mark as job was aborted then?
			return registry.ErrResourceNotFound
		}
		time.Sleep(GetNotificationsRetryDelay)
	}
	if notifications.Err() != nil {
		return errors.Wrapf(notifications.Err(), "failed to listen for '%s' execution notifications", execution.Id)
	}

	logs, err := NewExecutionLogsWriter(r.getLogPresigner(environmentId), execution.Id, execution.Workflow.Name, r.storageSkipVerify)
	if err != nil {
		return err
	}
	saver, err := NewExecutionSaver(ctx, r.executionsRepository, r.grpcClient, r.grpcApiToken, execution.Id, organizationId, environmentId, r.id, logs, r.newExecutionsEnabled)
	if err != nil {
		return err
	}
	defer logs.Cleanup()

	currentRef := ""
	var lastResult *testkube.TestWorkflowResult
	for n := range notifications.Channel() {
		if n.Temporary {
			continue
		}

		if n.Output != nil {
			saver.AppendOutput(*n.Output)
		} else if n.Result != nil {
			lastResult = n.Result
			// Don't send final result until everything is finished
			if n.Result.IsFinished() {
				continue
			}
			saver.UpdateResult(*n.Result)
		} else {
			if n.Ref != currentRef && n.Ref != "" {
				currentRef = n.Ref
				err = logs.WriteStart(n.Ref)
				if err != nil {
					log.DefaultLogger.Errorw("failed to write start logs", "id", execution.Id, "ref", n.Ref)
					continue
				}
			}
			_, err = logs.Write([]byte(n.Log))
			if err != nil {
				log.DefaultLogger.Errorw("failed to write logs", "id", execution.Id, "ref", n.Ref)
				continue
			}
		}
	}

	// Ignore further monitoring if it has been canceled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if notifications.Err() != nil {
		log.DefaultLogger.Errorw("error from TestWorkflow watcher", "id", execution.Id, "error", notifications.Err())
	}

	if lastResult == nil || !lastResult.IsFinished() {
		log.DefaultLogger.Errorw("not finished TestWorkflow result received, trying to recover...", "id", execution.Id)
		watcher := r.worker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			NoFollow: true,
		})
		if watcher.Err() == nil {
			for n := range watcher.Channel() {
				if n.Result != nil {
					lastResult = n.Result
				}
			}
		}

		if lastResult == nil {
			lastResult = execution.Result
		}
		if !lastResult.IsFinished() {
			log.DefaultLogger.Errorw("failed to recover TestWorkflow result, marking as fatal error...", "id", execution.Id)
			lastResult.Fatal(err, true, time.Now())
		}
	}

	for i := 0; i < SaveEndResultRetryCount; i++ {
		err = saver.End(ctx, *lastResult)
		if err == nil {
			break
		}
		log.DefaultLogger.Warnw("failed to save execution data", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i/10) * SaveEndResultRetryBaseDelay)
	}

	// Handle fatal error
	if err != nil {
		log.DefaultLogger.Errorw("failed to save execution data", "id", execution.Id, "error", err)
		return errors.Wrapf(err, "failed to save execution '%s' data", execution.Id)
	}

	// Try to substitute execution data
	execution.Output = nil
	execution.Result = lastResult
	execution.StatusAt = lastResult.FinishedAt

	// Emit data, if the Control Plane doesn't support informing about status by itself
	if !r.newExecutionsEnabled {
		if lastResult.IsPassed() {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(&execution))
		} else if lastResult.IsAborted() {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&execution))
		} else {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
		}
	}

	err = r.worker.Destroy(context.Background(), execution.Id, executionworkertypes.DestroyOptions{})
	if err != nil {
		// TODO: what to do on error?
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}

	return nil
}
func (r *runner) getExecution(ctx context.Context, environmentId, id string) (*testkube.TestWorkflowExecution, error) {
	if !r.newExecutionsEnabled {
		return r.getExecutionLegacy(ctx, environmentId, id)
	}
	md := metadata.New(map[string]string{apiKeyMeta: r.grpcApiToken, orgIdMeta: r.proContext.OrgID, agentIdMeta: r.id})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	req := cloud.GetExecutionRequest{EnvironmentId: environmentId, Id: id}
	response, err := r.grpcClient.GetExecution(metadata.NewOutgoingContext(ctx, md), &req, opts...)
	if err != nil {
		return nil, err
	}
	var execution testkube.TestWorkflowExecution
	err = json.Unmarshal(response.Execution, &execution)
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (r *runner) getExecutionLegacy(ctx context.Context, environmentId, id string) (*testkube.TestWorkflowExecution, error) {
	md := metadata.New(map[string]string{apiKeyMeta: r.grpcApiToken, orgIdMeta: r.proContext.OrgID, agentIdMeta: r.id, envIdMeta: environmentId})
	jsonPayload, err := json.Marshal(testworkflow2.ExecutionGetRequest{ID: id})
	if err != nil {
		return nil, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return nil, err
	}
	req := cloud.CommandRequest{
		Command: string(testworkflow2.CmdTestWorkflowExecutionGet),
		Payload: &s,
	}
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	cmdResponse, err := r.grpcClient.Call(metadata.NewOutgoingContext(ctx, md), &req, opts...)
	if err != nil {
		return nil, err
	}
	var response testworkflow2.ExecutionGetResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	return &response.WorkflowExecution, err
}

func (r *runner) Monitor(ctx context.Context, organizationId string, environmentId string, id string) error {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Check if there is any monitor attached already
	r.watching.LoadOrStore(id, false)
	if !r.watching.CompareAndSwap(id, false, true) {
		return nil
	}

	// Load the execution TODO: retry?
	execution, err := r.getExecution(ctx, environmentId, id)
	if err != nil {
		log.DefaultLogger.Errorw("failed to get execution", "id", id, "error", err)
		return err
	}

	return r.monitor(ctx, organizationId, environmentId, *execution)
}

func (r *runner) Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher {
	return r.worker.Notifications(ctx, id, executionworkertypes.NotificationsOptions{})
}

func (r *runner) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	res, err := r.worker.Execute(context.Background(), request)
	if err == nil {
		// TODO: consider retry?
		go r.Monitor(context.Background(), request.Execution.OrganizationId, request.Execution.EnvironmentId, request.Execution.Id)
	}
	return res, err
}

func (r *runner) Pause(id string) error {
	return r.worker.Pause(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Resume(id string) error {
	return r.worker.Resume(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Abort(id string) error {
	return r.worker.Abort(context.Background(), id, executionworkertypes.DestroyOptions{})
}
