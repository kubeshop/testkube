package scheduling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
)

type PostgresExecutionController struct {
	db *database.DB
}

func NewPostgresExecutionController(db *database.DB) *PostgresExecutionController {
	return &PostgresExecutionController{db: db}
}

func (a PostgresExecutionController) StartExecution(ctx context.Context, executionId string) error {
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"assigned"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:     "starting",
		ExecutionID:  executionId,
		FromStatuses: []string{"assigned"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

func (a PostgresExecutionController) PauseExecution(ctx context.Context, executionId string) error {
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"running"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:     "pausing",
		ExecutionID:  executionId,
		FromStatuses: []string{"running"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

func (a PostgresExecutionController) ResumeExecution(ctx context.Context, executionId string) error {
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"paused"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:     "resuming",
		ExecutionID:  executionId,
		FromStatuses: []string{"paused"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

func (a PostgresExecutionController) AbortExecution(ctx context.Context, executionId string) error {
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"running", "resuming"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update running execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:        "stopping",
		PredictedStatus: "aborted",
		ExecutionID:     executionId,
		FromStatuses:    []string{"running", "resuming"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update running execution result status: %w", err)
	}

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"queued"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update queued execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:     "aborted",
		FinishedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"queued"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update queued execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

func (a PostgresExecutionController) CancelExecution(ctx context.Context, executionId string) error {
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"running", "resuming"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update running execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:        "stopping",
		PredictedStatus: "canceled",
		ExecutionID:     executionId,
		FromStatuses:    []string{"running", "resuming"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update running execution result status: %w", err)
	}

	err = qtx.TransitionExecutionStatusAt(ctx, sqlc.TransitionExecutionStatusAtParams{
		StatusAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"queued"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update queued execution status_at: %w", err)
	}

	err = qtx.TransitionExecutionResultStatus(ctx, sqlc.TransitionExecutionResultStatusParams{
		ToStatus:     "canceled",
		FinishedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID:  executionId,
		FromStatuses: []string{"queued"},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update queued execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

func (a *PostgresExecutionController) ForceCancelExecution(ctx context.Context, executionId string) error {
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	_, err = qtx.GetExecutionForceCancel(ctx, executionId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("execution not found or not in cancellable state")
		}
		return fmt.Errorf("failed to verify execution: %w", err)
	}

	err = qtx.ForceCancelExecution(ctx, sqlc.ForceCancelExecutionParams{
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID: executionId,
	})
	if err != nil {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	err = qtx.ForceCancelExecutionResult(ctx, sqlc.ForceCancelExecutionResultParams{
		FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID: executionId,
	})
	if err != nil {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	err = qtx.ForceCancelExecutionSteps(ctx, sqlc.ForceCancelExecutionStepsParams{
		FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID: executionId,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to cancel execution steps: %w", err)
	}

	err = qtx.ForceCancelExecutionInitialization(ctx, sqlc.ForceCancelExecutionInitializationParams{
		FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		ExecutionID: executionId,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to cancel execution initialization: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
