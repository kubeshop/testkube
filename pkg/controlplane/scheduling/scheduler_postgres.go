package scheduling

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
)

type PostgresScheduler struct {
	db *database.DB
}

func NewPostgresScheduler(db *database.DB) Scheduler {
	return &PostgresScheduler{db: db}
}

func (s *PostgresScheduler) ScheduleExecution(ctx context.Context, info RunnerInfo) (testkube.TestWorkflowExecution, bool, error) {
	// Note: Standalone Control Plane does not support policies.
	// Note: Standalone Control Plane does not support label matches, excludes, etc. It always targets the sole standalone runner.

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	db := s.db.WithTx(tx)

	execRow, err := db.GetNextExecution(ctx)
	if err != nil {
		if err == pgx.ErrNoRows {
			return testkube.TestWorkflowExecution{}, false, nil
		}
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to fetch next execution: %w", err)
	}
	exec := mapPgTestWorkflowExecutionPartial(execRow.TestWorkflowExecution, execRow.TestWorkflowResult)

	if exec.Result.Status != nil && *exec.Result.Status != testkube.QUEUED_TestWorkflowStatus {
		fullExec, err := s.getFullExecution(ctx, execRow.TestWorkflowExecution, execRow.TestWorkflowResult)
		if err != nil {
			return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to fetch execution details: %w", err)
		}
		err = tx.Commit(ctx)
		if err != nil {
			return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to commit transaction: %w", err)
		}
		return fullExec, true, nil
	}

	now := time.Now()
	updatedRoot, err := db.AssignExecutionRoot(ctx, sqlc.AssignExecutionRootParams{
		RunnerID:    info.Id,
		Ts:          pgtype.Timestamptz{Time: now},
		ExecutionID: exec.Id,
	})
	if err != nil {
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to update execution root: %w", err)
	}
	updatedResult, err := db.AssignExecutionResult(ctx, sqlc.AssignExecutionResultParams{
		Ts:          pgtype.Timestamptz{Time: now},
		ExecutionID: exec.Id,
	})
	if err != nil {
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to update execution result: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fullExec, err := s.getFullExecution(ctx, updatedRoot, updatedResult)
	if err != nil {
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("failed to fetch execution details: %w", err)
	}

	return fullExec, true, nil
}

// We need everything because the event dispatcher and listeners expect the full execution.
func (s *PostgresScheduler) getFullExecution(ctx context.Context, exec sqlc.TestWorkflowExecution, result sqlc.TestWorkflowResult) (testkube.TestWorkflowExecution, error) {
	// TODO fetch in parallel.
	// TODO better handle NoRows found.
	workflow, err := s.db.GetExecutionWorkflow(ctx, exec.ID)
	if err != nil && err != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution workflow: %w", err)
	}
	resolvedWorkflow, err := s.db.GetExecutionResolvedWorkflow(ctx, exec.ID)
	if err != nil && err != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution resolved workflow: %w", err)
	}
	outputs, err := s.db.GetExecutionOutputs(ctx, exec.ID)
	if err != nil && err != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution outputs: %w", err)
	}
	signatures, err := s.db.GetExecutionSignatures(ctx, exec.ID)
	if err != nil && err != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution signatures: %w", err)
	}
	reports, err := s.db.GetExecutionReports(ctx, exec.ID)
	if err != nil && err != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution reports: %w", err)
	}
	aggregation, err := s.db.GetExecutionAggregation(ctx, exec.ID)
	if err != nil && err != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution aggregation: %w", err)
	}

	return mapPgTestWorkflowExecution(exec, result, workflow, resolvedWorkflow, signatures, reports, outputs, aggregation), nil
}
