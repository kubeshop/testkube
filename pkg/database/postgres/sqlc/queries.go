// interfaces.go
package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueriesInterface defines the interface for sqlc generated queries
type QueriesInterface interface {
	// Transaction methods
	WithTx(tx pgx.Tx) QueriesInterface

	// TestWorkflowExecution queries
	GetTestWorkflowExecution(ctx context.Context, id string) (GetTestWorkflowExecutionRow, error)
	GetTestWorkflowExecutionByNameAndTestWorkflow(ctx context.Context, arg GetTestWorkflowExecutionByNameAndTestWorkflowParams) (GetTestWorkflowExecutionByNameAndTestWorkflowRow, error)
	GetLatestTestWorkflowExecutionByTestWorkflow(ctx context.Context, workflowName string) (GetLatestTestWorkflowExecutionByTestWorkflowRow, error)
	GetLatestTestWorkflowExecutionsByTestWorkflows(ctx context.Context, workflowNames []string) ([]GetLatestTestWorkflowExecutionsByTestWorkflowsRow, error)
	GetRunningTestWorkflowExecutions(ctx context.Context) ([]GetRunningTestWorkflowExecutionsRow, error)
	GetTestWorkflowExecutionsTotals(ctx context.Context, arg GetTestWorkflowExecutionsTotalsParams) ([]GetTestWorkflowExecutionsTotalsRow, error)
	GetTestWorkflowExecutions(ctx context.Context, arg GetTestWorkflowExecutionsParams) ([]GetTestWorkflowExecutionsRow, error)
	GetTestWorkflowExecutionsSummary(ctx context.Context, arg GetTestWorkflowExecutionsSummaryParams) ([]GetTestWorkflowExecutionsSummaryRow, error)
	GetFinishedTestWorkflowExecutions(ctx context.Context, arg GetFinishedTestWorkflowExecutionsParams) ([]GetFinishedTestWorkflowExecutionsRow, error)
	GetUnassignedTestWorkflowExecutions(ctx context.Context) ([]GetUnassignedTestWorkflowExecutionsRow, error)

	// Insert operations
	InsertTestWorkflowExecution(ctx context.Context, arg InsertTestWorkflowExecutionParams) error
	InsertTestWorkflowResult(ctx context.Context, arg InsertTestWorkflowResultParams) error
	InsertTestWorkflowSignature(ctx context.Context, arg InsertTestWorkflowSignatureParams) error
	InsertTestWorkflowOutput(ctx context.Context, arg InsertTestWorkflowOutputParams) error
	InsertTestWorkflowReport(ctx context.Context, arg InsertTestWorkflowReportParams) error
	InsertTestWorkflowResourceAggregations(ctx context.Context, arg InsertTestWorkflowResourceAggregationsParams) error
	InsertTestWorkflow(ctx context.Context, arg InsertTestWorkflowParams) error

	// Update operations
	UpdateTestWorkflowExecution(ctx context.Context, arg UpdateTestWorkflowExecutionParams) error
	UpdateTestWorkflowExecutionResult(ctx context.Context, arg UpdateTestWorkflowExecutionResultParams) error
	UpdateExecutionStatusAt(ctx context.Context, arg UpdateExecutionStatusAtParams) error
	UpdateTestWorkflowExecutionReport(ctx context.Context, arg UpdateTestWorkflowExecutionReportParams) error
	UpdateTestWorkflowExecutionResourceAggregations(ctx context.Context, arg UpdateTestWorkflowExecutionResourceAggregationsParams) error

	// Delete operations
	DeleteTestWorkflowOutputs(ctx context.Context, executionID string) error
	DeleteTestWorkflowExecutionsByTestWorkflow(ctx context.Context, workflowName string) error
	DeleteAllTestWorkflowExecutions(ctx context.Context) error
	DeleteTestWorkflowExecutionsByTestWorkflows(ctx context.Context, workflowNames []string) error

	// Related data queries
	GetTestWorkflowSignatures(ctx context.Context, executionID string) ([]TestWorkflowSignature, error)
	GetTestWorkflowOutputs(ctx context.Context, executionID string) ([]TestWorkflowOutput, error)
	GetTestWorkflowReports(ctx context.Context, executionID string) ([]TestWorkflowReport, error)
	GetTestWorkflowResourceAggregations(ctx context.Context, executionID string) (TestWorkflowResourceAggregation, error)

	// Metrics and analytics
	GetTestWorkflowMetrics(ctx context.Context, arg GetTestWorkflowMetricsParams) ([]GetTestWorkflowMetricsRow, error)
	GetPreviousFinishedState(ctx context.Context, arg GetPreviousFinishedStateParams) (pgtype.Text, error)
	GetTestWorkflowExecutionTags(ctx context.Context, workflowName string) ([]GetTestWorkflowExecutionTagsRow, error)

	// Execution management
	InitTestWorkflowExecution(ctx context.Context, arg InitTestWorkflowExecutionParams) error
	AssignTestWorkflowExecution(ctx context.Context, arg AssignTestWorkflowExecutionParams) (string, error)
	AbortTestWorkflowExecutionIfQueued(ctx context.Context, arg AbortTestWorkflowExecutionIfQueuedParams) (string, error)
	AbortTestWorkflowResultIfQueued(ctx context.Context, arg AbortTestWorkflowResultIfQueuedParams) error

	// Sequence operations
	GetNextExecutionNumber(ctx context.Context, workflowName string) (int64, error)
}

// Ensure Queries implements QueriesInterface
var _ QueriesInterface = (*SQLCQueriesWrapper)(nil)

// SQLCQueriesWrapper wraps Queries to implement QueriesInterface
type SQLCQueriesWrapper struct {
	*Queries
}

// NewSQLCQueriesWrapper creates a new wrapper for Queries
func NewSQLCQueriesWrapper(queries *Queries) QueriesInterface {
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
