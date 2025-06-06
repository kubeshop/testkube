// interfaces.go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
)

// QueriesInterface defines the interface for sqlc generated queries
type QueriesInterface interface {
	// Transaction methods
	WithTx(tx pgx.Tx) QueriesInterface

	// TestWorkflowExecution queries
	GetTestWorkflowExecution(ctx context.Context, id string) (sqlc.GetTestWorkflowExecutionRow, error)
	GetTestWorkflowExecutionByNameAndTestWorkflow(ctx context.Context, arg sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowParams) (sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowRow, error)
	GetLatestTestWorkflowExecutionByTestWorkflow(ctx context.Context, workflowName pgtype.Text) (sqlc.GetLatestTestWorkflowExecutionByTestWorkflowRow, error)
	GetLatestTestWorkflowExecutionsByTestWorkflows(ctx context.Context, workflowNames []pgtype.Text) ([]sqlc.GetLatestTestWorkflowExecutionsByTestWorkflowsRow, error)
	GetRunningTestWorkflowExecutions(ctx context.Context) ([]sqlc.GetRunningTestWorkflowExecutionsRow, error)
	GetTestWorkflowExecutionsTotals(ctx context.Context, arg sqlc.GetTestWorkflowExecutionsTotalsParams) ([]sqlc.GetTestWorkflowExecutionsTotalsRow, error)
	GetTestWorkflowExecutions(ctx context.Context, arg sqlc.GetTestWorkflowExecutionsParams) ([]sqlc.GetTestWorkflowExecutionsRow, error)
	GetTestWorkflowExecutionsSummary(ctx context.Context, arg sqlc.GetTestWorkflowExecutionsSummaryParams) ([]sqlc.GetTestWorkflowExecutionsSummaryRow, error)
	GetFinishedTestWorkflowExecutions(ctx context.Context, arg sqlc.GetFinishedTestWorkflowExecutionsParams) ([]sqlc.GetFinishedTestWorkflowExecutionsRow, error)
	GetUnassignedTestWorkflowExecutions(ctx context.Context) ([]sqlc.GetUnassignedTestWorkflowExecutionsRow, error)

	// Insert operations
	InsertTestWorkflowExecution(ctx context.Context, arg sqlc.InsertTestWorkflowExecutionParams) error
	InsertTestWorkflowResult(ctx context.Context, arg sqlc.InsertTestWorkflowResultParams) error
	InsertTestWorkflowSignature(ctx context.Context, arg sqlc.InsertTestWorkflowSignatureParams) error
	InsertTestWorkflowOutput(ctx context.Context, arg sqlc.InsertTestWorkflowOutputParams) error
	InsertTestWorkflowReport(ctx context.Context, arg sqlc.InsertTestWorkflowReportParams) error
	InsertTestWorkflowResourceAggregations(ctx context.Context, arg sqlc.InsertTestWorkflowResourceAggregationsParams) error
	InsertTestWorkflow(ctx context.Context, arg sqlc.InsertTestWorkflowParams) error

	// Update operations
	UpdateTestWorkflowExecution(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionParams) error
	UpdateTestWorkflowExecutionResult(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionResultParams) error
	UpdateExecutionStatusAt(ctx context.Context, arg sqlc.UpdateExecutionStatusAtParams) error
	UpdateTestWorkflowExecutionReport(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionReportParams) error
	UpdateTestWorkflowExecutionResourceAggregations(ctx context.Context, arg sqlc.UpdateTestWorkflowExecutionResourceAggregationsParams) error

	// Delete operations
	DeleteTestWorkflowOutputs(ctx context.Context, executionID string) error
	DeleteTestWorkflowExecutionsByTestWorkflow(ctx context.Context, workflowName pgtype.Text) error
	DeleteAllTestWorkflowExecutions(ctx context.Context) error
	DeleteTestWorkflowExecutionsByTestWorkflows(ctx context.Context, workflowNames []pgtype.Text) error

	// Related data queries
	GetTestWorkflowSignatures(ctx context.Context, executionID string) ([]sqlc.TestWorkflowSignature, error)
	GetTestWorkflowOutputs(ctx context.Context, executionID string) ([]sqlc.TestWorkflowOutput, error)
	GetTestWorkflowReports(ctx context.Context, executionID string) ([]sqlc.TestWorkflowReport, error)
	GetTestWorkflowResourceAggregations(ctx context.Context, executionID string) (sqlc.TestWorkflowResourceAggregation, error)

	// Metrics and analytics
	GetTestWorkflowMetrics(ctx context.Context, arg sqlc.GetTestWorkflowMetricsParams) ([]sqlc.GetTestWorkflowMetricsRow, error)
	GetPreviousFinishedState(ctx context.Context, arg sqlc.GetPreviousFinishedStateParams) (pgtype.Text, error)
	GetTestWorkflowExecutionTags(ctx context.Context, workflowName string) ([]sqlc.GetTestWorkflowExecutionTagsRow, error)

	// Execution management
	InitTestWorkflowExecution(ctx context.Context, arg sqlc.InitTestWorkflowExecutionParams) error
	AssignTestWorkflowExecution(ctx context.Context, arg sqlc.AssignTestWorkflowExecutionParams) (string, error)
	AbortTestWorkflowExecutionIfQueued(ctx context.Context, arg sqlc.AbortTestWorkflowExecutionIfQueuedParams) (string, error)
	AbortTestWorkflowResultIfQueued(ctx context.Context, arg sqlc.AbortTestWorkflowResultIfQueuedParams) error

	// Sequence operations
	GetNextExecutionNumber(ctx context.Context, workflowName pgtype.Text) (int64, error)
}

// Ensure sqlc.Queries implements QueriesInterface
var _ QueriesInterface = (*SQLCQueriesWrapper)(nil)

// SQLCQueriesWrapper wraps sqlc.Queries to implement QueriesInterface
type SQLCQueriesWrapper struct {
	*sqlc.Queries
}

// NewSQLCQueriesWrapper creates a new wrapper for sqlc.Queries
func NewSQLCQueriesWrapper(queries *sqlc.Queries) QueriesInterface {
	return &SQLCQueriesWrapper{Queries: queries}
}

// WithTx returns a new QueriesInterface with transaction
func (w *SQLCQueriesWrapper) WithTx(tx pgx.Tx) QueriesInterface {
	return &SQLCQueriesWrapper{Queries: w.Queries.WithTx(tx)}
}

// DatabaseInterface defines the interface for database operations
type DatabaseInterface interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PgxPoolWrapper wraps pgxpool.Pool to implement DatabaseInterface
type PgxPoolWrapper struct {
	*pgxpool.Pool
}

func (w *PgxPoolWrapper) Begin(ctx context.Context) (pgx.Tx, error) {
	return w.Pool.Begin(ctx)
}
