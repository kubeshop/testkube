package scheduling

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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

// Transitions returns every execution awaiting a pause, resume or stop.
func (a *PostgresExecutionQuerier) Transitions(ctx context.Context) ([]ExecutionTransition, error) {
	rows, err := a.db.GetExecutionTransitions(ctx, []string{
		string(testkube.PAUSING_TestWorkflowStatus),
		string(testkube.RESUMING_TestWorkflowStatus),
		string(testkube.STOPPING_TestWorkflowStatus),
	})
	if err != nil {
		return nil, fmt.Errorf("find executions awaiting a transition: %w", err)
	}

	transitions := make([]ExecutionTransition, 0, len(rows))
	for _, row := range rows {
		transitions = append(transitions, ExecutionTransition{
			Id:              row.ExecutionID,
			Status:          testkube.TestWorkflowStatus(row.Status.String),
			PredictedStatus: testkube.TestWorkflowStatus(row.PredictedStatus.String),
		})
	}
	return transitions, nil
}

// ToStart returns at most limit executions to dispatch, oldest scheduled first.
func (a *PostgresExecutionQuerier) ToStart(ctx context.Context, limit int, redispatchBefore time.Time) ([]testkube.TestWorkflowExecution, error) {
	rows, err := a.db.GetExecutionsToStart(ctx, sqlc.GetExecutionsToStartParams{
		RedispatchBefore: pgtype.Timestamptz{Time: redispatchBefore, Valid: true},
		BatchSize:        int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("find executions to start: %w", err)
	}

	executions := make([]testkube.TestWorkflowExecution, 0, len(rows))
	for _, row := range rows {
		executions = append(executions, mapPgTestWorkflowExecutionForDispatch(row.TestWorkflowExecution))
	}
	return executions, nil
}

// StaleStarting returns the ids of executions the runner never acknowledged.
func (a *PostgresExecutionQuerier) StaleStarting(ctx context.Context, staleBefore time.Time, limit int) ([]string, error) {
	ids, err := a.db.GetStaleStartingExecutions(ctx, sqlc.GetStaleStartingExecutionsParams{
		StaleBefore: pgtype.Timestamptz{Time: staleBefore, Valid: true},
		BatchSize:   int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("find stale starting executions: %w", err)
	}
	return ids, nil
}
