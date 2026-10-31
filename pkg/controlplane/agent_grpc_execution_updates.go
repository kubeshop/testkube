package controlplane

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	log2 "github.com/kubeshop/testkube/pkg/log"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
)

// claimTimeout bounds the bookkeeping writes that follow a dispatch. They run
// detached from the request context, so they need a deadline of their own.
const claimTimeout = 10 * time.Second

func (s *Server) GetExecutionUpdates(ctx context.Context, _ *executionv1.GetExecutionUpdatesRequest) (*executionv1.GetExecutionUpdatesResponse, error) {
	info := scheduling.RunnerInfo{
		Id:            common.StandaloneRunner,
		Name:          common.StandaloneRunnerName,
		EnvironmentId: common.StandaloneEnvironment,
	}
	log := log2.DefaultLogger.With("runner id", info.Id, "runner name", info.Name)

	// Control instructions. Unbounded, but cheap: an id and a target state per row.
	updates := s.collectTransitions(ctx, log)

	// One bounded page of work. The bound is the whole point of this handler.
	// It used to return every pending execution at once, each hydrated with six
	// extra queries, so a burst of workflows sharing a cron minute pushed the call
	// past the runner's 30s deadline and started a backoff the backlog could never
	// drain.
	batch, err := s.executionQuerier.ToStart(ctx, s.dispatchBatchSize(), time.Now().Add(-s.redispatchAfter()))
	if err != nil {
		log.Errorw("Error retrieving executions to start", "err", err)
		return &executionv1.GetExecutionUpdatesResponse{Update: updates}, nil
	}

	assigned := make([]string, 0, len(batch))
	redispatched := make([]string, 0, len(batch))
	for _, exe := range batch {
		if exe.Result != nil && exe.Result.Status != nil && *exe.Result.Status == testkube.ASSIGNED_TestWorkflowStatus {
			assigned = append(assigned, exe.Id)
			continue
		}
		redispatched = append(redispatched, exe.Id)
	}

	// Claim before dispatching, and dispatch only what the claim accepted.
	//
	// The read and the claim are separate statements, so an execution can be
	// aborted in between; the guarded write then skips it. Returning the row
	// anyway would have the runner create Kubernetes resources for an execution
	// that is already over. A claim that fails outright drops its rows from this
	// response too - they are still ASSIGNED, so the next poll picks them up.
	claimed, renewed := s.claimBatch(ctx, log, assigned, redispatched)

	dispatchable := make(map[string]struct{}, len(claimed)+len(renewed))
	for _, id := range claimed {
		dispatchable[id] = struct{}{}
	}
	for _, id := range renewed {
		dispatchable[id] = struct{}{}
	}

	start := make([]*executionv1.ExecutionStart, 0, len(dispatchable))
	for _, exe := range batch {
		if _, ok := dispatchable[exe.Id]; !ok {
			continue
		}
		executionStart := createExecutionStart(exe, info)
		start = append(start, &executionStart)
	}

	if dropped := len(batch) - len(start); dropped > 0 {
		log.Infow("dropped executions from the dispatch batch, they were no longer startable",
			"dropped", dropped, "dispatched", len(start))
	}

	// Hand the newly claimed executions to the dispatcher. Hydrating each one and
	// publishing its event inline cost a full execution read plus up to three
	// publish attempts with sleeps between them, per row, inside the request the
	// runner is blocked on.
	s.enqueueStartEvents(ctx, claimed)

	return &executionv1.GetExecutionUpdatesResponse{Update: updates, Start: start}, nil
}

// claimBatch records that the batch was handed out, and reports what it actually
// claimed and renewed.
//
// Detached from the request context on purpose: if the runner has already given
// up on this call, dropping the claim would offer the same rows again on the next
// poll as ASSIGNED and publish their start events a second time.
func (s *Server) claimBatch(ctx context.Context, log *zap.SugaredLogger, assigned, redispatched []string) (claimed, renewed []string) {
	if len(assigned) == 0 && len(redispatched) == 0 {
		return nil, nil
	}

	claimCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), claimTimeout)
	defer cancel()

	if len(assigned) > 0 {
		var err error
		claimed, err = s.ExecutionController.StartExecutions(claimCtx, assigned)
		if err != nil {
			// Nothing is dispatched from a failed claim. The rows stay ASSIGNED,
			// so the next poll retries them.
			log.Warnw("error marking executions as starting", "err", err, "count", len(assigned))
			claimed = nil
		}
	}
	if len(redispatched) > 0 {
		// Renew the lease so a retried row is not offered again on the next poll.
		var err error
		renewed, err = s.ExecutionController.RefreshStartingExecutions(claimCtx, redispatched)
		if err != nil {
			log.Warnw("error renewing dispatch lease", "err", err, "count", len(redispatched))
			renewed = nil
		}
	}
	return claimed, renewed
}

// collectTransitions turns pending pause/resume/stop records into runner
// instructions. A read failure is logged and yields none, because this call also
// carries the dispatch batch and must not fail over a control instruction.
func (s *Server) collectTransitions(ctx context.Context, log *zap.SugaredLogger) []*executionv1.ExecutionStateTransition {
	transitions, err := s.executionQuerier.Transitions(ctx)
	if err != nil {
		log.Errorw("Error retrieving execution transitions", "err", err)
		return nil
	}

	updates := make([]*executionv1.ExecutionStateTransition, 0, len(transitions))
	for _, transition := range transitions {
		var to executionv1.ExecutionState
		switch transition.Status {
		case testkube.PAUSING_TestWorkflowStatus:
			to = executionv1.ExecutionState_EXECUTION_STATE_PAUSED
		case testkube.RESUMING_TestWorkflowStatus:
			to = executionv1.ExecutionState_EXECUTION_STATE_RUNNING
		case testkube.STOPPING_TestWorkflowStatus:
			// CancelExecutionRunningResult writes predicted_status 'canceled' and
			// AbortExecutionRunningResult writes 'aborted'; the runner maps
			// CANCELLED to Cancel and ABORTED to Abort. These two were exchanged,
			// so a user cancel ran an abort and an abort ran a cancel.
			//
			// Only 'canceled' means cancel. Anything else, including an empty
			// predicted status, aborts - the stronger of the two.
			to = executionv1.ExecutionState_EXECUTION_STATE_ABORTED
			if transition.PredictedStatus == testkube.CANCELED_TestWorkflowStatus {
				to = executionv1.ExecutionState_EXECUTION_STATE_CANCELLED
			}
		default:
			log.Warnw("unexpected state", "id", transition.Id, "status", transition.Status)
			continue
		}

		updates = append(updates, &executionv1.ExecutionStateTransition{
			ExecutionId:  common.Ptr(transition.Id),
			TransitionTo: common.Ptr(to),
		})
	}
	return updates
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
		Tags:                 exe.Tags,
		Lineage:              lineageProtoOf(exe.Lineage),
	}
}

// lineageProtoOf carries the execution's lineage across to the runner.
//
// The scheduler records this on the execution rather than handing it to the
// runner, so every ExecutionStart writer has to read it back off the record.
// Omitting it here does not fail: the pod just cannot resolve
// execution("rerun"), and a workflow written against its own previous run
// silently loses the reference.
func lineageProtoOf(lineage *testkube.TestWorkflowExecutionLineage) *executionv1.ExecutionLineage {
	if lineage == nil {
		return nil
	}
	return &executionv1.ExecutionLineage{
		BaseExecutionId: common.Ptr(lineage.BaseId),
		RootExecutionId: common.Ptr(lineage.RootId),
		Attempt:         common.Ptr(lineage.Attempt),
	}
}
