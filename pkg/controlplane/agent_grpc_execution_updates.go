package controlplane

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	log2 "github.com/kubeshop/testkube/pkg/log"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
)

func (s *Server) GetExecutionUpdates(ctx context.Context, _ *executionv1.GetExecutionUpdatesRequest) (*executionv1.GetExecutionUpdatesResponse, error) {
	info := scheduling.RunnerInfo{
		Id:            common.StandaloneRunner,
		Name:          common.StandaloneRunnerName,
		EnvironmentId: common.StandaloneEnvironment,
	}

	// To ensure that the response goes out even if the update OR scheduling fails, then we call
	// separate functions that do not error (logging is doing a lot of heavy lifting here).
	log := log2.DefaultLogger.With("runner id", info.Id, "runner name", info.Name)
	update := s.getExecutionUpdates(ctx, log)
	start := s.getNextExecution(ctx, log, info)

	return &executionv1.GetExecutionUpdatesResponse{Update: update, Start: start}, nil
}

func (s *Server) getExecutionUpdates(ctx context.Context, log *zap.SugaredLogger) []*executionv1.ExecutionStateTransition {
	var updates []*executionv1.ExecutionStateTransition

	for exe, err := range s.executionQuerier.Pausing(ctx) {
		if err != nil {
			log.Errorw("Error retrieving pausing executions",
				"err", err)
			continue
		}
		updates = append(updates, &executionv1.ExecutionStateTransition{
			ExecutionId:  common.Ptr(exe.Id),
			TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_PAUSED),
		})
	}
	for exe, err := range s.executionQuerier.Resuming(ctx) {
		if err != nil {
			log.Errorw("Error retrieving resuming executions",
				"err", err)
			continue
		}
		updates = append(updates, &executionv1.ExecutionStateTransition{
			ExecutionId:  common.Ptr(exe.Id),
			TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_RUNNING),
		})
	}
	for exe, err := range s.executionQuerier.Aborting(ctx) {
		if err != nil {
			log.Errorw("Error retrieving aborting executions",
				"err", err)
			continue
		}
		updates = append(updates, &executionv1.ExecutionStateTransition{
			ExecutionId:  common.Ptr(exe.Id),
			TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_ABORTED),
		})
	}
	for exe, err := range s.executionQuerier.Cancelling(ctx) {
		if err != nil {
			log.Errorw("Error retrieving cancelling executions",
				"err", err)
			continue
		}
		updates = append(updates, &executionv1.ExecutionStateTransition{
			ExecutionId:  common.Ptr(exe.Id),
			TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_CANCELLED),
		})
	}

	return updates
}

func (s *Server) getNextExecution(ctx context.Context, log *zap.SugaredLogger, info scheduling.RunnerInfo) []*executionv1.ExecutionStart {
	// Run THE schedule query that will atomically assign one or no execution to this runner.
	exe, ok, err := s.scheduler.ScheduleExecution(ctx, info)
	if err != nil {
		log.Warnw("error scheduling execution", "err", err)
		return nil
	}
	// No executions found, nothing to do.
	if !ok {
		return nil
	}

	// Mark the execution as starting
	if err := s.executionController.StartExecution(ctx, exe.Id); err != nil {
		log.Warnw("error marking execution as starting", "err", err)
	}

	// Populate some possibly missing values and avoid nil pointer issues.
	var workflowName string
	var ancestorIds []string
	if exe.Workflow != nil {
		workflowName = exe.Workflow.Name
	}
	if exe.RunningContext != nil && exe.RunningContext.Actor != nil {
		// For some reason we store ancestor execution IDs as a path rather than an array.
		ancestorIds = strings.Split(exe.RunningContext.Actor.ExecutionPath, "/")
	}
	return []*executionv1.ExecutionStart{
		{
			ExecutionId:     common.Ptr(exe.Id),
			GroupId:         common.Ptr(exe.GroupId),
			Name:            common.Ptr(exe.Name),
			Number:          common.Ptr(exe.Number),
			QueuedAt:        timestamppb.New(exe.ScheduledAt),
			DisableWebhooks: common.Ptr(exe.DisableWebhooks),
			EnvironmentId:   common.Ptr(info.EnvironmentId),
			//ExecutionToken:       common.Ptr(token), TODO currently build-in control plane is insecure. Add auth and generate execution tokens.
			AncestorExecutionIds: ancestorIds,
			WorkflowName:         common.Ptr(workflowName),
			VariableOverrides:    exe.Runtime.Variables,
		},
	}
}
