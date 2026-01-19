package controlplane

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
	log := log2.DefaultLogger.With("runner id", info.Id, "runner name", info.Name)

	var updates []*executionv1.ExecutionStateTransition
	var start []*executionv1.ExecutionStart

	statuses := []testkube.TestWorkflowStatus{
		// statuses required for updates
		testkube.PAUSING_TestWorkflowStatus,
		testkube.RESUMING_TestWorkflowStatus,
		testkube.STOPPING_TestWorkflowStatus,

		// statuses required for start
		testkube.STARTING_TestWorkflowStatus,
		testkube.ASSIGNED_TestWorkflowStatus,
	}

	for exe, err := range s.executionQuerier.ByStatus(ctx, statuses) {
		if err != nil {
			log.Errorw("Error retrieving executions", "err", err)
			continue
		}

		switch *exe.Result.Status {
		case testkube.PAUSING_TestWorkflowStatus:
			updates = append(updates, &executionv1.ExecutionStateTransition{
				ExecutionId:  common.Ptr(exe.Id),
				TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_PAUSED),
			})
		case testkube.RESUMING_TestWorkflowStatus:
			updates = append(updates, &executionv1.ExecutionStateTransition{
				ExecutionId:  common.Ptr(exe.Id),
				TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_RUNNING),
			})
		case testkube.STOPPING_TestWorkflowStatus:
			if *exe.Result.PredictedStatus == testkube.CANCELED_TestWorkflowStatus {
				updates = append(updates, &executionv1.ExecutionStateTransition{
					ExecutionId:  common.Ptr(exe.Id),
					TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_ABORTED),
				})
			} else {
				updates = append(updates, &executionv1.ExecutionStateTransition{
					ExecutionId:  common.Ptr(exe.Id),
					TransitionTo: common.Ptr(executionv1.ExecutionState_EXECUTION_STATE_CANCELLED),
				})
			}

		case testkube.STARTING_TestWorkflowStatus:
			executionStart := createExecutionStart(exe, info)
			start = append(start, &executionStart)
		case testkube.ASSIGNED_TestWorkflowStatus:
			executionStart := createExecutionStart(exe, info)
			start = append(start, &executionStart)

			// Mark the execution as starting
			if err := s.ExecutionController.StartExecution(ctx, exe.Id); err != nil {
				log.Warnw("error marking execution as starting", "err", err)
			}

			// Dispatch event for WebHooks and friends
			s.emitter.Notify(testkube.NewEventStartTestWorkflow(&exe))
		default:
			log.Warnw("unexpected state", "id", exe.Id, "status", *exe.Result.Status)
		}
	}

	return &executionv1.GetExecutionUpdatesResponse{Update: updates, Start: start}, nil
}

func createExecutionStart(exe testkube.TestWorkflowExecution, info scheduling.RunnerInfo) executionv1.ExecutionStart {
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

	variableOverrides := make(map[string]string)
	if exe.Runtime != nil {
		variableOverrides = exe.Runtime.Variables
	}

	return executionv1.ExecutionStart{
		ExecutionId:          common.Ptr(exe.Id),
		GroupId:              common.Ptr(exe.GroupId),
		Name:                 common.Ptr(exe.Name),
		Number:               common.Ptr(exe.Number),
		QueuedAt:             timestamppb.New(exe.ScheduledAt),
		DisableWebhooks:      common.Ptr(exe.DisableWebhooks),
		EnvironmentId:        common.Ptr(info.EnvironmentId),
		ExecutionToken:       common.Ptr(""), //TODO currently build-in control plane is insecure. Add auth and generate execution tokens.
		AncestorExecutionIds: ancestorIds,
		WorkflowName:         common.Ptr(workflowName),
		VariableOverrides:    variableOverrides,
	}
}
