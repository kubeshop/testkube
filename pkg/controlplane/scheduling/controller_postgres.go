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

// StartExecution marks an execution that is currently assigned that it should be started.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, then no error will be emitted and no action will have been taken.
func (a PostgresExecutionController) StartExecution(ctx context.Context, executionId string) error {
	// Start a transaction for atomic operations
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	// Update execution status_at
	err = qtx.StartExecution(ctx, sqlc.StartExecutionParams{
		ExecutionID: executionId,
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	// Update result status
	err = qtx.StartExecutionResult(ctx, executionId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

// PauseExecution marks an execution that is currently running that it should be paused.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is currently running, then no error will be emitted and no action will
// have been taken.
func (a PostgresExecutionController) PauseExecution(ctx context.Context, executionId string) error {
	// Start a transaction for atomic operations
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	// Update execution status_at
	err = qtx.PauseExecution(ctx, sqlc.PauseExecutionParams{
		ExecutionID: executionId,
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	// Update result status
	err = qtx.PauseExecutionResult(ctx, executionId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

// ResumeExecution marks an execution that is currently paused that it should be resumed.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is currently paused, then no error will be emitted and no action will
// have been taken.
func (a PostgresExecutionController) ResumeExecution(ctx context.Context, executionId string) error {
	// Start a transaction for atomic operations
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	// Update execution status_at
	err = qtx.ResumeExecution(ctx, sqlc.ResumeExecutionParams{
		ExecutionID: executionId,
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	// Update result status
	err = qtx.ResumeExecutionResult(ctx, executionId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	return tx.Commit(ctx)
}

// AbortExecution marks an execution that is currently in an executing state that it
// should be aborted.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can only be aborted if they are currently in a Starting, Running, Paused, or
// Resuming state.
func (a PostgresExecutionController) AbortExecution(ctx context.Context, executionId string) error {
	// Start a transaction for atomic operations
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	// Try to abort running execution first
	err = qtx.AbortExecutionRunning(ctx, sqlc.AbortExecutionRunningParams{
		ExecutionID: executionId,
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})

	var runningUpdated bool
	if err == nil {
		// Update result status for running execution
		err = qtx.AbortExecutionRunningResult(ctx, executionId)
		if err == nil {
			runningUpdated = true
		}
	}

	// If no running execution was found, try to abort queued execution
	if !runningUpdated {
		err = qtx.AbortExecutionQueued(ctx, sqlc.AbortExecutionQueuedParams{
			ExecutionID: executionId,
			StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to update queued execution status_at: %w", err)
		}

		// Update result status for queued execution
		err = qtx.AbortExecutionQueuedResult(ctx, sqlc.AbortExecutionQueuedResultParams{
			ExecutionID: executionId,
			FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to update queued execution result status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// CancelExecution marks an execution that is currently in an executing state that it
// should be cancelled.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can only be cancelled if they are currently in a Starting, Running, Paused, or
// Resuming state.
func (a PostgresExecutionController) CancelExecution(ctx context.Context, executionId string) error {
	// Start a transaction for atomic operations
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	// Try to cancel running execution first
	err = qtx.CancelExecutionRunning(ctx, sqlc.CancelExecutionRunningParams{
		ExecutionID: executionId,
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})

	var runningUpdated bool
	if err == nil {
		// Update result status for running execution
		err = qtx.CancelExecutionRunningResult(ctx, executionId)
		if err == nil {
			runningUpdated = true
		}
	}

	// If no running execution was found, try to cancel queued execution
	if !runningUpdated {
		err = qtx.CancelExecutionQueued(ctx, sqlc.CancelExecutionQueuedParams{
			ExecutionID: executionId,
			StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to update queued execution status_at: %w", err)
		}

		// Update result status for queued execution
		err = qtx.CancelExecutionQueuedResult(ctx, sqlc.CancelExecutionQueuedResultParams{
			ExecutionID: executionId,
			FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to update queued execution result status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ForceCancelExecution cancels an execution and all its steps
// This mirrors the MongoDB implementation's behavior:
// 1. Updates the main execution status_at timestamp
// 2. Sets the result status to 'canceled'
// 3. Cancels all non-terminated steps (preserves passed/failed steps)
// 4. Sets missing timestamps (queuedat, startedat, finishedat) to current time
// 5. Cancels the initialization step if present
func (a *PostgresExecutionController) ForceCancelExecution(ctx context.Context, executionId string) error {
	// Start a transaction for atomic operations
	tx, err := a.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := a.db.WithTx(tx)
	now := time.Now()

	// First, verify the execution exists and is in a cancellable state
	_, err = qtx.GetExecutionForceCancel(ctx, executionId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("execution not found or not in cancellable state")
		}
		return fmt.Errorf("failed to verify execution: %w", err)
	}

	// Step 1: Update execution status_at timestamp
	err = qtx.ForceCancelExecution(ctx, sqlc.ForceCancelExecutionParams{
		ExecutionID: executionId,
		StatusAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update execution status_at: %w", err)
	}

	// Step 2: Update main result status to 'canceled'
	err = qtx.ForceCancelExecutionResult(ctx, sqlc.ForceCancelExecutionResultParams{
		ExecutionID: executionId,
		FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update execution result status: %w", err)
	}

	// Step 3: Cancel all steps (preserving passed/failed steps)
	// This updates:
	// - status to 'canceled' for non-terminated steps
	// - queuedat, startedat, finishedat to 'now' if they are missing/empty
	err = qtx.ForceCancelExecutionSteps(ctx, sqlc.ForceCancelExecutionStepsParams{
		ExecutionID: executionId,
		FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to cancel execution steps: %w", err)
	}

	// Step 4: Cancel initialization step if present
	err = qtx.ForceCancelExecutionInitialization(ctx, sqlc.ForceCancelExecutionInitializationParams{
		ExecutionID: executionId,
		FinishedAt:  pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to cancel execution initialization: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
