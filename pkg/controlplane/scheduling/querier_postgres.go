package scheduling

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"

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

// Assigned yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be started by the runner.
func (a *PostgresExecutionQuerier) Assigned(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, testkube.ASSIGNED_TestWorkflowStatus, "")
}

// Assigned yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be started by the runner.
func (a *PostgresExecutionQuerier) Starting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, testkube.STARTING_TestWorkflowStatus, "")
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

// ByStatus yields an iterator returning all executions that match one of the given statuses.
func (a PostgresExecutionQuerier) ByStatus(ctx context.Context, statuses []testkube.TestWorkflowStatus) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIteratorByStatus(ctx, statuses)
}

func (a *PostgresExecutionQuerier) executionIteratorByStatus(ctx context.Context, statuses []testkube.TestWorkflowStatus) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return func(yield func(testkube.TestWorkflowExecution, error) bool) {
		var status []string
		for _, s := range statuses {
			status = append(status, string(s))
		}

		executions, err := a.db.GetExecutionsByStatuses(ctx, status)
		if err != nil {
			yield(testkube.TestWorkflowExecution{}, fmt.Errorf("find executions with ExecutionQuerier statuses: %w", err))
			return
		}
		for _, row := range executions {

			exec, err := a.getFullExecution(ctx, row.TestWorkflowExecution, row.TestWorkflowResult)
			if err != nil {
				if !yield(exec, fmt.Errorf("decode test workflow execution: %w", err)) {
					return
				}
				return
			}

			if err != nil {
				if !yield(exec, fmt.Errorf("decode test workflow execution: %w", err)) {
					return
				}
				continue
			}
			if !yield(exec, nil) {
				return
			}
		}
	}
}

// We need everything because the event dispatcher and listeners expect the full execution.
func (s *PostgresExecutionQuerier) getFullExecution(ctx context.Context, exec sqlc.TestWorkflowExecution, result sqlc.TestWorkflowResult) (testkube.TestWorkflowExecution, error) {
	var workflow sqlc.TestWorkflow
	var resolvedWorkflow sqlc.TestWorkflow
	var outputs []sqlc.TestWorkflowOutput
	var signatures []sqlc.TestWorkflowSignature
	var reports []sqlc.TestWorkflowReport
	var aggregation sqlc.TestWorkflowResourceAggregation

	var workflowErr, resolvedWorkflowErr, outputsErr, signaturesErr, reportsErr, aggregationErr error

	var wg sync.WaitGroup
	wg.Add(6)

	go func() {
		defer wg.Done()
		workflow, workflowErr = s.db.GetExecutionWorkflow(ctx, exec.ID)
	}()

	go func() {
		defer wg.Done()
		resolvedWorkflow, resolvedWorkflowErr = s.db.GetExecutionResolvedWorkflow(ctx, exec.ID)
	}()

	go func() {
		defer wg.Done()
		outputs, outputsErr = s.db.GetExecutionOutputs(ctx, exec.ID)
	}()

	go func() {
		defer wg.Done()
		signatures, signaturesErr = s.db.GetExecutionSignatures(ctx, exec.ID)
	}()

	go func() {
		defer wg.Done()
		reports, reportsErr = s.db.GetExecutionReports(ctx, exec.ID)
	}()

	go func() {
		defer wg.Done()
		aggregation, aggregationErr = s.db.GetExecutionAggregation(ctx, exec.ID)
	}()

	wg.Wait()

	if workflowErr != nil && workflowErr != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution workflow: %w", workflowErr)
	}
	if resolvedWorkflowErr != nil && resolvedWorkflowErr != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution resolved workflow: %w", resolvedWorkflowErr)
	}
	if outputsErr != nil && outputsErr != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution outputs: %w", outputsErr)
	}
	if signaturesErr != nil && signaturesErr != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution signatures: %w", signaturesErr)
	}
	if reportsErr != nil && reportsErr != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution reports: %w", reportsErr)
	}
	if aggregationErr != nil && aggregationErr != pgx.ErrNoRows {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("cannot fetch execution aggregation: %w", aggregationErr)
	}

	return mapPgTestWorkflowExecution(exec, result, workflow, resolvedWorkflow, signatures, reports, outputs, aggregation), nil
}
