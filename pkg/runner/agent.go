package runner

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	errors2 "github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

const (
	apiKeyMeta  = "api-key"
	agentIdMeta = "agent-id"
	orgIdMeta   = "organization-id"
	envIdMeta   = "environment-id"
)

const (
	saveResultRetryMaxAttempts = 100
	saveResultRetryBaseDelay   = 300 * time.Millisecond
)

type agentLoop struct {
	runner              Runner
	worker              executionworkertypes.Worker
	logger              *zap.SugaredLogger
	emitter             event.Interface
	client              cloud.TestKubeCloudAPIClient
	grpcApiToken        string
	runnerId            string
	organizationId      string
	legacyEnvironmentId string

	newExecutionsEnabled bool
}

type AgentLoop interface {
	Start(ctx context.Context) error
}

func newAgentLoop(
	runner Runner,
	worker executionworkertypes.Worker,
	logger *zap.SugaredLogger,
	emitter event.Interface,
	grpcClient cloud.TestKubeCloudAPIClient,
	grpcApiToken string,
	runnerId string,
	organizationId string,
	legacyEnvironmentId string,
	newExecutionsEnabled bool,
) AgentLoop {
	return &agentLoop{
		runner:               runner,
		worker:               worker,
		logger:               logger,
		emitter:              emitter,
		client:               grpcClient,
		grpcApiToken:         grpcApiToken,
		runnerId:             runnerId,
		organizationId:       organizationId,
		legacyEnvironmentId:  legacyEnvironmentId,
		newExecutionsEnabled: newExecutionsEnabled,
	}
}

func (a *agentLoop) Start(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := a.run(ctx)

		a.logger.Errorw("runner agent connection failed, reconnecting", "error", err)

		// TODO: some smart back off strategy?
		time.Sleep(5 * time.Second)
	}
}

func (a *agentLoop) buildContext(ctx context.Context, environmentId string) context.Context {
	md := metadata.MD{}
	if a.runnerId != "" {
		md[agentIdMeta] = []string{a.runnerId}
	}
	if a.grpcApiToken != "" {
		md[apiKeyMeta] = []string{a.grpcApiToken}
	}
	if a.organizationId != "" {
		md[orgIdMeta] = []string{a.organizationId}
	}
	if a.legacyEnvironmentId != "" {
		// TODO: delete, as [1] the runner is decoupled out of it, and [2] the Control Plane has this information anyway
		md[envIdMeta] = []string{a.legacyEnvironmentId}
	}
	if environmentId != "" {
		md[envIdMeta] = []string{environmentId}
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func (a *agentLoop) getExecution(ctx context.Context, environmentId, id string) (*testkube.TestWorkflowExecution, error) {
	if !a.newExecutionsEnabled {
		return a.getExecutionLegacy(ctx, environmentId, id)
	}
	ctx = a.buildContext(ctx, environmentId)
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	req := cloud.GetExecutionRequest{EnvironmentId: environmentId, Id: id}
	response, err := a.client.GetExecution(ctx, &req, opts...)
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

func (a *agentLoop) getExecutionLegacy(ctx context.Context, environmentId, id string) (*testkube.TestWorkflowExecution, error) {
	ctx = a.buildContext(ctx, environmentId)
	jsonPayload, err := json.Marshal(testworkflow.ExecutionGetRequest{ID: id})
	if err != nil {
		return nil, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return nil, err
	}
	req := cloud.CommandRequest{
		Command: string(testworkflow.CmdTestWorkflowExecutionGet),
		Payload: &s,
	}
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	cmdResponse, err := a.client.Call(ctx, &req, opts...)
	if err != nil {
		return nil, err
	}
	var response testworkflow.ExecutionGetResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	return &response.WorkflowExecution, err
}

func (a *agentLoop) presignLogs(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) (string, error) {
	if !a.newExecutionsEnabled {
		return a.presignLogsLegacy(ctx, environmentId, execution)
	}

	md := metadata.New(map[string]string{apiKeyMeta: a.grpcApiToken, orgIdMeta: a.organizationId, agentIdMeta: a.runnerId})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	res, err := a.client.SaveExecutionLogsPresigned(metadata.NewOutgoingContext(ctx, md), &cloud.SaveExecutionLogsPresignedRequest{
		EnvironmentId: environmentId,
		Id:            execution.Id,
	}, opts...)
	if err != nil {
		return "", err
	}
	return res.Url, nil
}

func (a *agentLoop) presignLogsLegacy(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) (string, error) {
	// Extracting the test workflow name
	workflowName := ""
	if execution.Workflow != nil {
		workflowName = execution.Workflow.Name
	}

	jsonPayload, err := json.Marshal(testworkflow.OutputPresignSaveLogRequest{ID: execution.Id, WorkflowName: workflowName})
	if err != nil {
		return "", err
	}
	s := structpb.Struct{}
	if err = s.UnmarshalJSON(jsonPayload); err != nil {
		return "", err
	}
	cmdReq := cloud.CommandRequest{
		Command: string(testworkflow.CmdTestWorkflowOutputPresignSaveLog),
		Payload: &s,
	}
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	cmdResponse, err := a.client.Call(ctx, &cmdReq, opts...)
	if err != nil {
		return "", err
	}
	var response testworkflow.OutputPresignSaveLogResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	if err != nil {
		return "", err
	}
	return response.URL, nil
}

// TODO: Add proper gRPC method for that
func (a *agentLoop) _saveEmptyLogs(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	ctx = a.buildContext(ctx, environmentId)

	// Presigning the log
	url, err := a.presignLogs(ctx, environmentId, execution)
	if err != nil {
		return err
	}

	// Saving empty logs
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/octet-stream")
	httpClient := http.DefaultClient
	httpClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	res, err := httpClient.Do(req)
	if err != nil {
		return errors2.Wrap(err, "failed to save empty logs in the storage")
	}
	if res.StatusCode != http.StatusOK {
		return errors2.Errorf("error saving file with presigned url: expected 200 OK response code, got %d", res.StatusCode)
	}

	return err
}

func (a *agentLoop) saveEmptyLogs(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() error {
		return a._saveEmptyLogs(ctx, environmentId, execution)
	})
	if err != nil {
		a.logger.Errorw("failed to save empty log", "executionId", execution.Id, "error", err)
	}
	return err
}

// TODO: Add proper gRPC method for that
func (a *agentLoop) _updateExecution(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	ctx = a.buildContext(ctx, environmentId)

	jsonPayload, err := json.Marshal(testworkflow.ExecutionUpdateRequest{WorkflowExecution: *execution})
	if err != nil {
		return err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return err
	}
	cmdReq := cloud.CommandRequest{
		Command: string(testworkflow.CmdTestWorkflowExecutionUpdate),
		Payload: &s,
	}
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	_, err = a.client.Call(ctx, &cmdReq, opts...)
	return err
}

func (a *agentLoop) updateExecution(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() error {
		err := a._updateExecution(ctx, environmentId, execution)
		if err != nil {
			a.logger.Warnw("failed to update the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		a.logger.Errorw("failed to update the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (a *agentLoop) _finishExecution(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	ctx = a.buildContext(ctx, environmentId)

	resultBytes, err := json.Marshal(execution.Result)
	if err != nil {
		return err
	}

	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	_, err = a.client.FinishExecution(ctx, &cloud.FinishExecutionRequest{
		EnvironmentId: environmentId,
		Id:            execution.Id,
		Result:        resultBytes,
	}, opts...)
	return err
}

func (a *agentLoop) finishExecution(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	if !a.newExecutionsEnabled {
		err := a.updateExecution(ctx, environmentId, execution)
		if err != nil {
			return err
		}

		// Emit events locally if the Control Plane doesn't support that
		//a.emitter.Notify(testkube.NewEventStartTestWorkflow(execution)) // TODO: delete - sent from Cloud
		if execution.Result.IsPassed() {
			a.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
		} else if execution.Result.IsAborted() {
			a.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
		} else {
			a.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
		}
		return nil
	}

	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() error {
		err := a._finishExecution(ctx, environmentId, execution)
		if err != nil {
			a.logger.Warnw("failed to finish the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		a.logger.Errorw("failed to finish the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (a *agentLoop) init(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	// TODO: Make it non-conflict
	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() error {
		prevExecution, err := a.getExecution(ctx, environmentId, execution.Id)
		if err != nil {
			return err
		}
		prevExecution.RunnerId = a.runnerId
		prevExecution.Namespace = execution.Namespace
		prevExecution.Signature = execution.Signature
		err = a._updateExecution(ctx, environmentId, prevExecution)
		if err != nil {
			a.logger.Warnw("failed to initialize the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		a.logger.Errorw("failed to initialize the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (a *agentLoop) run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	ctx = a.buildContext(ctx, a.legacyEnvironmentId)

	// Handle the new mechanism for runners
	if a.newExecutionsEnabled {
		g.Go(func() error {
			return errors2.Wrap(a.loopRunnerRequests(ctx), "runners loop")
		})
	}

	// Handle Test Workflow notifications of all kinds
	g.Go(func() error {
		return errors2.Wrap(a.loopNotifications(ctx), "notifications loop")
	})
	g.Go(func() error {
		return errors2.Wrap(a.loopServiceNotifications(ctx), "service notifications loop")
	})
	g.Go(func() error {
		return errors2.Wrap(a.loopParallelStepNotifications(ctx), "parallel steps notifications loop")
	})

	return g.Wait()
}

func (a *agentLoop) loopNotifications(ctx context.Context) error {
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := a.client.GetTestWorkflowNotificationsStream(ctx, opts...)

	g, ctx := errgroup.WithContext(ctx)
	responses := make(chan *cloud.TestWorkflowNotificationsResponse, 5)

	// Send responses in sequence
	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g.Go(func() error {
		for msg := range responses {
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
		return nil
	})

	// Process the requests
	g.Go(func() error {
		defer close(responses)
		for {
			// Take the context error if possible
			if err == nil && ctx.Err() != nil {
				err = ctx.Err()
			}

			// Handle the error
			if err != nil {
				return err
			}

			// Get the next request
			var req *cloud.TestWorkflowNotificationsRequest
			req, err = stream.Recv()
			if err != nil {
				continue
			}

			// Send PONG to the PING message
			if req.RequestType == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
				responses <- &cloud.TestWorkflowNotificationsResponse{StreamId: req.StreamId, SeqNo: 0}
				continue
			}

			// Start reading the notifications
			g.Go(func(req *cloud.TestWorkflowNotificationsRequest) func() error {
				seqNo := uint32(0)
				return func() error {
					// Read the initial status TODO: consider getting from the database
					status, err := a.worker.Summary(ctx, req.ExecutionId, executionworkertypes.GetOptions{})
					if err != nil {
						responses <- buildCloudError(req.StreamId, fmt.Sprintf("failed to fetch real-time notifications: failed to read execution summary: %s", err.Error()))
						return nil
					}

					// Fail fast when it's already finished
					if status.EstimatedResult.IsFinished() {
						responses <- buildCloudError(req.StreamId, fmt.Sprintf("failed to fetch real-time notifications: execution is already finished"))
						return nil
					}

					// Start reading the notifications
					// TODO: allow stopping that - it will require different gRPC API though
					notifications := a.worker.Notifications(ctx, status.Resource.Id, executionworkertypes.NotificationsOptions{
						Hints: executionworkertypes.Hints{
							Namespace:   status.Namespace,
							Signature:   status.Signature,
							ScheduledAt: common.Ptr(status.Execution.ScheduledAt),
						},
					})

					// Process the notifications
					for notification := range notifications.Channel() {
						responses <- buildCloudNotification(req.StreamId, seqNo, notification)
						seqNo++
					}
					if notifications.Err() != nil {
						responses <- buildCloudError(req.StreamId, fmt.Sprintf("failed to fetch real-time notifications: %s", err.Error()))
					}
					return nil
				}
			}(req))
		}
	})

	return g.Wait()
}

func (a *agentLoop) loopServiceNotifications(ctx context.Context) error {
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := a.client.GetTestWorkflowServiceNotificationsStream(ctx, opts...)

	g, ctx := errgroup.WithContext(ctx)
	responses := make(chan *cloud.TestWorkflowServiceNotificationsResponse, 5)

	// Send responses in sequence
	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g.Go(func() error {
		for msg := range responses {
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
		return nil
	})

	// Process the requests
	g.Go(func() error {
		defer close(responses)
		for {
			// Take the context error if possible
			if err == nil && ctx.Err() != nil {
				err = ctx.Err()
			}

			// Handle the error
			if err != nil {
				return err
			}

			// Get the next request
			var req *cloud.TestWorkflowServiceNotificationsRequest
			req, err = stream.Recv()
			if err != nil {
				continue
			}

			// Send PONG to the PING message
			if req.RequestType == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
				responses <- &cloud.TestWorkflowServiceNotificationsResponse{StreamId: req.StreamId, SeqNo: 0}
				continue
			}

			// Start reading the notifications
			g.Go(func(req *cloud.TestWorkflowServiceNotificationsRequest) func() error {
				seqNo := uint32(0)
				return func() error {
					// Build the internal resource name
					resourceId := fmt.Sprintf("%s-%s-%d", req.ExecutionId, req.ServiceName, req.ServiceIndex)

					// Start reading the notifications
					// TODO: allow stopping that - it will require different gRPC API though
					notifications := a.worker.Notifications(ctx, resourceId, executionworkertypes.NotificationsOptions{
						Hints: executionworkertypes.Hints{},
					})

					// Process the notifications
					for notification := range notifications.Channel() {
						responses <- buildServiceCloudNotification(req.StreamId, seqNo, notification)
						seqNo++
					}
					if notifications.Err() != nil {
						responses <- buildServiceCloudError(req.StreamId, fmt.Sprintf("failed to fetch real-time notifications: %s", err.Error()))
					}
					return nil
				}
			}(req))
		}
	})

	return g.Wait()
}

func (a *agentLoop) loopParallelStepNotifications(ctx context.Context) error {
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := a.client.GetTestWorkflowParallelStepNotificationsStream(ctx, opts...)

	g, ctx := errgroup.WithContext(ctx)
	responses := make(chan *cloud.TestWorkflowParallelStepNotificationsResponse, 5)

	// Send responses in sequence
	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g.Go(func() error {
		for msg := range responses {
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
		return nil
	})

	// Process the requests
	g.Go(func() error {
		defer close(responses)
		for {
			// Take the context error if possible
			if err == nil && ctx.Err() != nil {
				err = ctx.Err()
			}

			// Handle the error
			if err != nil {
				return err
			}

			// Get the next request
			var req *cloud.TestWorkflowParallelStepNotificationsRequest
			req, err = stream.Recv()
			if err != nil {
				continue
			}

			// Send PONG to the PING message
			if req.RequestType == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
				responses <- &cloud.TestWorkflowParallelStepNotificationsResponse{StreamId: req.StreamId, SeqNo: 0}
				continue
			}

			// Start reading the notifications
			g.Go(func(req *cloud.TestWorkflowParallelStepNotificationsRequest) func() error {
				seqNo := uint32(0)
				return func() error {
					// Build the internal resource name
					resourceId := fmt.Sprintf("%s-%s-%d", req.ExecutionId, req.Ref, req.WorkerIndex)

					// Start reading the notifications
					// TODO: allow stopping that - it will require different gRPC API though
					notifications := a.worker.Notifications(ctx, resourceId, executionworkertypes.NotificationsOptions{
						Hints: executionworkertypes.Hints{},
					})

					// Process the notifications
					for notification := range notifications.Channel() {
						responses <- buildParallelStepCloudNotification(req.StreamId, seqNo, notification)
						seqNo++
					}
					if notifications.Err() != nil {
						responses <- buildParallelStepCloudError(req.StreamId, fmt.Sprintf("failed to fetch real-time notifications: %s", err.Error()))
					}
					return nil
				}
			}(req))
		}
	})

	return g.Wait()
}

func (a *agentLoop) loopRunnerRequests(ctx context.Context) error {
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := a.client.GetRunnerRequests(ctx, opts...)
	for {
		// Ignore if it's not implemented in the Control Plane
		if getGrpcErrorCode(err) == codes.Unimplemented {
			return nil
		}

		// Take the context error if possible
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
		}

		// Handle the error
		if err != nil {
			return err
		}

		// Get the next runner request
		var req *cloud.RunnerRequest
		req, err = stream.Recv()
		if err != nil {
			continue
		}

		// Lock the execution for itself
		var resp *cloud.ObtainExecutionResponse
		resp, err = a.client.ObtainExecution(ctx, &cloud.ObtainExecutionRequest{Id: req.Id, EnvironmentId: req.EnvironmentId}, opts...)
		if err != nil {
			a.logger.Errorf("failed to obtain execution '%s/%s', from Control Plane: %v", req.EnvironmentId, req.Id, err)
			continue
		}

		// Ignore if the resource has been locked before
		if !resp.Success {
			continue
		}

		// Continue
		err = a.runTestWorkflow(req.EnvironmentId, req.Id)
		if err != nil {
			a.logger.Errorf("failed to run execution '%s/%s' from Control Plane: %v", req.EnvironmentId, req.Id, err)
			continue
		}
	}

	return nil
}

func (a *agentLoop) runTestWorkflow(environmentId string, executionId string) error {
	ctx := context.Background()
	logger := a.logger.With("environmentId", environmentId, "executionId", executionId)

	// Get the execution details
	execution, err := a.getExecution(ctx, environmentId, executionId)
	if err != nil {
		return errors2.Wrapf(err, "failed to get execution details '%s/%s' from Control Plane", environmentId, executionId)
	}

	// TODO: Pass it there?
	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		// TODO
		//DashboardUrl:   e.dashboardURI,
		//CDEventsTarget: e.cdEventsTarget,
	}

	parentIds := ""
	if execution.RunningContext != nil && execution.RunningContext.Actor != nil {
		parentIds = execution.RunningContext.Actor.ExecutionPath
	}
	result, err := a.runner.Execute(executionworkertypes.ExecuteRequest{
		Execution: testworkflowconfig.ExecutionConfig{
			Id:              execution.Id,
			GroupId:         execution.GroupId,
			Name:            execution.Name,
			Number:          execution.Number,
			ScheduledAt:     execution.ScheduledAt,
			DisableWebhooks: execution.DisableWebhooks,
			Debug:           false,
			OrganizationId:  a.organizationId,
			EnvironmentId:   environmentId,
			ParentIds:       parentIds,
		},
		Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*execution.ResolvedWorkflow),
		ControlPlane: controlPlaneConfig,
	})

	// TODO: define "revoke" error by runner (?)
	if err != nil {
		execution.InitializationError("Failed to run execution", err)
		_ = a.saveEmptyLogs(context.Background(), environmentId, execution)
		err2 := a.finishExecution(context.Background(), environmentId, execution)
		err = errors.Join(err, err2)
		if err != nil {
			logger.Errorw("failed to run and update execution", "error", err)
		}
		return nil
	}

	// Inform about execution start
	//e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution)) // TODO: delete - sent from Cloud

	// Apply the known data to temporary object.
	execution.Namespace = result.Namespace
	execution.Signature = result.Signature
	execution.RunnerId = a.runnerId
	if err = a.init(context.Background(), environmentId, execution); err != nil {
		logger.Errorw("failed to mark execution as initialized", "error", err)
	}

	return nil
}
