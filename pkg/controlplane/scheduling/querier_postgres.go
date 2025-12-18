package scheduling

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
)

type PostgresExecutionQuerier struct {
	db *database.DB
}

func NewPostgresExecutionQuerier(db *database.DB) *PostgresExecutionQuerier {
	return &PostgresExecutionQuerier{db: db}
}

// Pausing yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be paused by the runner.
func (a *PostgresExecutionQuerier) Pausing(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, testkube.PAUSING_TestWorkflowStatus, "")
}

// Resuming yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be resumed by the runner.
func (a *PostgresExecutionQuerier) Resuming(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, testkube.RESUMING_TestWorkflowStatus, "")
}

// Aborting yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be aborted by the runner.
func (a *PostgresExecutionQuerier) Aborting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, testkube.STOPPING_TestWorkflowStatus, testkube.ABORTED_TestWorkflowStatus)
}

// Cancelling yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be cancelled by the runner.
func (a *PostgresExecutionQuerier) Cancelling(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, testkube.STOPPING_TestWorkflowStatus, testkube.CANCELED_TestWorkflowStatus)
}

func (a *PostgresExecutionQuerier) executionIterator(ctx context.Context, status, predicted testkube.TestWorkflowStatus) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return func(yield func(testkube.TestWorkflowExecution, error) bool) {
		executions, err := a.db.GetExecutionsByStatus(ctx, sqlc.GetExecutionsByStatusParams{
			Status:          string(status),
			PredictedStatus: string(predicted),
		})
		if err != nil {
			yield(testkube.TestWorkflowExecution{}, fmt.Errorf("find executions with ExecutionQuerier statuses: %w", err))
			return
		}
		for _, row := range executions {
			exec, err := a.convertRowToObj(row)
			if err != nil {
				if !yield(*exec, fmt.Errorf("decode test workflow execution: %w", err)) {
					return
				}
				continue
			}
			if !yield(*exec, nil) {
				return
			}
		}
	}
}

// Current usage only needs id
func (a *PostgresExecutionQuerier) convertRowToObj(row sqlc.GetExecutionsByStatusRow) (*testkube.TestWorkflowExecution, error) {
	return &testkube.TestWorkflowExecution{Id: row.TestWorkflowExecution.ID}, nil
}
