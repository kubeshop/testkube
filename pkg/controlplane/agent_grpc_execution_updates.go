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

func (s *Server) GetExecutionUpdates(ctx context.Context, req *executionv1.GetExecutionUpdatesRequest) (*executionv1.GetExecutionUpdatesResponse, error) {
	info := scheduling.RunnerInfo{
		Id:            common.StandaloneRunner,
		Name:          common.StandaloneRunnerName,
		EnvironmentId: common.StandaloneEnvironment,
	}

	// To ensure that the response goes out even if the update OR scheduling fails, then we call
	// separate functions that do not error (logging is doing a lot of heavy lifting here).
	log := log2.DefaultLogger.With("runner id", info.Id, "runner name", info.Name)
	update := s.getExecutionUpdates(ctx, log, info)
	start := s.getNextExecution(ctx, log, info)

	return &executionv1.GetExecutionUpdatesResponse{Update: update, Start: start}, nil
}

// TODO
func (s *Server) getExecutionUpdates(ctx context.Context, log *zap.SugaredLogger, info scheduling.RunnerInfo) []*executionv1.ExecutionStateTransition {
	return []*executionv1.ExecutionStateTransition{}
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

	// TODO Generate execution token
	// TODO Mark execution as starting

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
			//ExecutionToken:       common.Ptr(token), TODO
			AncestorExecutionIds: ancestorIds,
			WorkflowName:         common.Ptr(workflowName),
			VariableOverrides:    exe.Runtime.Variables,
		},
	}
}
