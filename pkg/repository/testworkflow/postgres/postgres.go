package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/repository/common"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
	sequencepostgres "github.com/kubeshop/testkube/pkg/repository/sequence/postgres"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/utils"
)

var _ testworkflow.Repository = (*PostgresRepository)(nil)

const (
	configParamSizeLimit = 100
)

type PostgresRepository struct {
	db                 sqlc.DatabaseInterface
	queries            sqlc.TestWorkflowExecutionQueriesInterface
	sequenceRepository sequence.Repository
}

type PostgresRepositoryOpt func(*PostgresRepository)

func NewPostgresRepository(db *pgxpool.Pool, opts ...PostgresRepositoryOpt) *PostgresRepository {
	r := &PostgresRepository{
		db:                 &sqlc.PgxPoolWrapper{Pool: db},
		queries:            sqlc.NewSQLCTestWorkflowExecutionQueriesWrapper(sqlc.New(db)),
		sequenceRepository: sequencepostgres.NewPostgresRepository(db),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithQueriesInterface allows injecting a custom queries interface (useful for testing)
func WithQueriesInterface(queries sqlc.TestWorkflowExecutionQueriesInterface) PostgresRepositoryOpt {
	return func(r *PostgresRepository) {
		r.queries = queries
	}
}

// WithDatabaseInterface allows injecting a custom database interface (useful for testing)
func WithDatabaseInterface(db sqlc.DatabaseInterface) PostgresRepositoryOpt {
	return func(r *PostgresRepository) {
		r.db = db
	}
}

// Helper functions for type conversions
func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toPgBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

func toPgInt4(i int32) pgtype.Int4 {
	return pgtype.Int4{Int32: i, Valid: true}
}

func toPgTimestamp(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toJSONB(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func fromPgText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func fromPgBool(b pgtype.Bool) bool {
	if !b.Valid {
		return false
	}
	return b.Bool
}

func fromPgInt4(i pgtype.Int4) int32 {
	if !i.Valid {
		return 0
	}
	return i.Int32
}

func fromPgTimestamp(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func fromJSONB[T any](data []byte) (*T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var result T
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get method to use complete data
func (r *PostgresRepository) Get(ctx context.Context, id string) (testkube.TestWorkflowExecution, error) {
	// Get complete execution data with all related data in a single query
	row, err := r.queries.GetTestWorkflowExecution(ctx, id)
	if err != nil {
		return testkube.TestWorkflowExecution{}, err
	}

	// Convert the complete row to execution object
	execution, err := r.convertCompleteRowToExecutionWithRelated(row)
	if err != nil {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("failed to convert execution: %w", err)
	}

	return *execution, nil
}

func (r *PostgresRepository) GetWithRunner(ctx context.Context, id, runner string) (result testkube.TestWorkflowExecution, err error) {
	return testkube.TestWorkflowExecution{}, errors.New("not yet implemented")
}

// Helper method to convert complete row to TestWorkflowExecution
func (r *PostgresRepository) convertCompleteRowToExecutionWithRelated(row sqlc.GetTestWorkflowExecutionRow) (*testkube.TestWorkflowExecution, error) {
	var err error
	execution := &testkube.TestWorkflowExecution{
		Id:                        row.ID,
		GroupId:                   fromPgText(row.GroupID),
		RunnerId:                  fromPgText(row.RunnerID),
		Name:                      row.Name,
		Namespace:                 fromPgText(row.Namespace),
		Number:                    fromPgInt4(row.Number),
		ScheduledAt:               fromPgTimestamp(row.ScheduledAt),
		AssignedAt:                fromPgTimestamp(row.AssignedAt),
		StatusAt:                  fromPgTimestamp(row.StatusAt),
		TestWorkflowExecutionName: fromPgText(row.TestWorkflowExecutionName),
		DisableWebhooks:           fromPgBool(row.DisableWebhooks),
	}

	// Parse basic JSONB fields
	r.parseExecutionJSONFields(execution, row.RunnerTarget, row.RunnerOriginalTarget, row.Tags, row.RunningContext, row.ConfigParams)

	// Build result if exists
	if row.Status.Valid {
		execution.Result, err = r.buildResultFromRow(
			row.Status, row.PredictedStatus, row.QueuedAt, row.StartedAt, row.FinishedAt,
			row.Duration, row.TotalDuration, row.DurationMs, row.PausedMs, row.TotalDurationMs,
			row.Pauses, row.Initialization, row.Steps,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build result from row: %w", err)
		}
	}

	// Build workflow if exists
	if row.WorkflowName.Valid {
		execution.Workflow, err = r.buildWorkflowFromRow(
			row.WorkflowName, row.WorkflowNamespace, row.WorkflowDescription,
			row.WorkflowLabels, row.WorkflowAnnotations, row.WorkflowCreated,
			row.WorkflowUpdated, row.WorkflowSpec, row.WorkflowReadOnly, row.WorkflowStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build workflow from row: %w", err)
		}
	}

	// Build resolved workflow if exists
	if row.ResolvedWorkflowName.Valid {
		execution.ResolvedWorkflow, err = r.buildWorkflowFromRow(
			row.ResolvedWorkflowName, row.ResolvedWorkflowNamespace, row.ResolvedWorkflowDescription,
			row.ResolvedWorkflowLabels, row.ResolvedWorkflowAnnotations, row.ResolvedWorkflowCreated,
			row.ResolvedWorkflowUpdated, row.ResolvedWorkflowSpec, row.ResolvedWorkflowReadOnly, row.ResolvedWorkflowStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build resolved workflow from row: %w", err)
		}
	}

	// Parse signatures from JSON
	if len(row.SignaturesJson) > 0 {
		var signatures []map[string]interface{}
		if err := json.Unmarshal(row.SignaturesJson, &signatures); err != nil {
			return nil, fmt.Errorf("failed to parse signatures JSON: %w", err)
		}
		execution.Signature = r.buildSignatureTreeFromJSON(signatures)
	}

	// Parse outputs from JSON
	if len(row.OutputsJson) > 0 {
		var outputs []map[string]interface{}
		if err := json.Unmarshal(row.OutputsJson, &outputs); err != nil {
			return nil, fmt.Errorf("failed to parse outputs JSON: %w", err)
		}
		execution.Output = r.convertOutputsFromJSON(outputs)
	}

	// Parse reports from JSON
	if len(row.ReportsJson) > 0 {
		var reports []map[string]interface{}
		err := json.Unmarshal(row.ReportsJson, &reports)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reports JSON: %w", err)
		}
		execution.Reports, err = r.convertReportsFromJSON(reports)
		if err != nil {
			return nil, fmt.Errorf("failed to converts reports JSON: %w", err)
		}
	}

	// Parse resource aggregations
	if len(row.ResourceAggregationsGlobal) > 0 || len(row.ResourceAggregationsStep) > 0 {
		execution.ResourceAggregations = &testkube.TestWorkflowExecutionResourceAggregationsReport{}

		if len(row.ResourceAggregationsGlobal) > 0 {
			if err := json.Unmarshal(row.ResourceAggregationsGlobal, &execution.ResourceAggregations.Global); err != nil {
				return nil, fmt.Errorf("failed to parse resource aggregations global: %w", err)
			}
		}

		if len(row.ResourceAggregationsStep) > 0 {
			if err := json.Unmarshal(row.ResourceAggregationsStep, &execution.ResourceAggregations.Step); err != nil {
				return nil, fmt.Errorf("failed to parse resource aggregations step: %w", err)
			}
		}
	}

	// Populate config params if resolved workflow exists
	if execution.ResolvedWorkflow != nil && execution.ResolvedWorkflow.Spec != nil {
		execution.ConfigParams = populateConfigParams(execution.ResolvedWorkflow, execution.ConfigParams)
	}

	return execution.UnscapeDots(), nil
}

// Helper methods for converting JSON data
func (r *PostgresRepository) buildSignatureTreeFromJSON(signatures []map[string]interface{}) []testkube.TestWorkflowSignature {
	if len(signatures) == 0 {
		return nil
	}

	// Convert to map for easier processing
	sigMap := make(map[int32]*testkube.TestWorkflowSignature)
	parentChildMap := make(map[int32][]int32)

	for _, sig := range signatures {
		id := int32(sig["id"].(float64))

		twSig := &testkube.TestWorkflowSignature{
			Ref:      getStringFromMap(sig, "ref"),
			Name:     getStringFromMap(sig, "name"),
			Category: getStringFromMap(sig, "category"),
			Optional: getBoolFromMap(sig, "optional"),
			Negative: getBoolFromMap(sig, "negative"),
		}
		sigMap[id] = twSig

		if parentID, ok := sig["parent_id"]; ok && parentID != nil {
			parentInt := int32(parentID.(float64))
			parentChildMap[parentInt] = append(parentChildMap[parentInt], id)
		}
	}

	// Build tree structure
	var buildChildren func(parentId int32) []testkube.TestWorkflowSignature
	buildChildren = func(parentId int32) []testkube.TestWorkflowSignature {
		var children []testkube.TestWorkflowSignature
		for _, childId := range parentChildMap[parentId] {
			child := *sigMap[childId]
			child.Children = buildChildren(childId)
			children = append(children, child)
		}
		return children
	}

	// Find root signatures (those without parents)
	var roots []testkube.TestWorkflowSignature
	for _, sig := range signatures {
		id := int32(sig["id"].(float64))
		if _, hasParent := sig["parent_id"]; !hasParent || sig["parent_id"] == nil {
			root := *sigMap[id]
			root.Children = buildChildren(id)
			roots = append(roots, root)
		}
	}

	return roots
}
func (r *PostgresRepository) convertOutputsFromJSON(outputs []map[string]interface{}) []testkube.TestWorkflowOutput {
	result := make([]testkube.TestWorkflowOutput, len(outputs))
	for i, output := range outputs {
		result[i] = testkube.TestWorkflowOutput{
			Ref:  getStringFromMap(output, "ref"),
			Name: getStringFromMap(output, "name"),
		}
		if value, ok := output["value"]; ok && value != nil {
			result[i].Value = value.(map[string]interface{})
		}
	}
	return result
}

func (r *PostgresRepository) convertReportsFromJSON(reports []map[string]interface{}) ([]testkube.TestWorkflowReport, error) {
	result := make([]testkube.TestWorkflowReport, len(reports))
	for i, report := range reports {
		result[i] = testkube.TestWorkflowReport{
			Ref:  getStringFromMap(report, "ref"),
			Kind: getStringFromMap(report, "kind"),
			File: getStringFromMap(report, "file"),
		}
		if summary, ok := report["summary"]; ok && summary != nil {
			summaryBytes, err := json.Marshal(summary)
			if err != nil {
				return nil, err
			}

			summaryObj, err := fromJSONB[testkube.TestWorkflowReportSummary](summaryBytes)
			if err != nil {
				return nil, err
			}

			result[i].Summary = summaryObj
		}
	}
	return result, nil
}

// Helper functions for map access
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok && val != nil {
		return val.(string)
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok && val != nil {
		return val.(bool)
	}
	return false
}

func (r *PostgresRepository) executionToSummary(row testkube.TestWorkflowExecution) testkube.TestWorkflowExecutionSummary {
	return testkube.TestWorkflowExecutionSummary{
		Id:                   row.Id,
		GroupId:              row.GroupId,
		RunnerId:             row.RunnerId,
		Name:                 row.Name,
		Number:               row.Number,
		ScheduledAt:          row.ScheduledAt,
		StatusAt:             row.StatusAt,
		Result:               r.resultToSummary(row.Result),
		Workflow:             r.workflowToSummary(row.Workflow),
		Tags:                 row.Tags,
		RunningContext:       row.RunningContext,
		ConfigParams:         row.ConfigParams,
		Reports:              row.Reports,
		ResourceAggregations: row.ResourceAggregations,
	}
}

func (r *PostgresRepository) resultToSummary(row *testkube.TestWorkflowResult) *testkube.TestWorkflowResultSummary {
	if row == nil {
		return nil
	}
	return &testkube.TestWorkflowResultSummary{
		Status:          row.Status,
		PredictedStatus: row.PredictedStatus,
		QueuedAt:        row.QueuedAt,
		StartedAt:       row.StartedAt,
		FinishedAt:      row.FinishedAt,
		Duration:        row.Duration,
		TotalDuration:   row.TotalDuration,
		DurationMs:      row.DurationMs,
		TotalDurationMs: row.TotalDurationMs,
		PausedMs:        row.PausedMs,
	}
}

func (r *PostgresRepository) workflowToSummary(row *testkube.TestWorkflow) *testkube.TestWorkflowSummary {
	if row == nil {
		return nil
	}

	var health *testkube.TestWorkflowExecutionHealth
	if row.Status != nil {
		health = row.Status.Health
	}

	return &testkube.TestWorkflowSummary{
		Name:        row.Name,
		Namespace:   row.Namespace,
		Labels:      row.Labels,
		Annotations: row.Annotations,
		Health:      health,
	}
}

// GetByNameAndTestWorkflow returns execution by name and workflow name
func (r *PostgresRepository) GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (testkube.TestWorkflowExecution, error) {
	// Get complete execution data with all related data in a single query
	row, err := r.queries.GetTestWorkflowExecutionByNameAndTestWorkflow(ctx, sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowParams{
		Name:         name,
		WorkflowName: workflowName,
	})
	if err != nil {
		return testkube.TestWorkflowExecution{}, err
	}

	// Convert the complete row to execution object
	execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
	if err != nil {
		return testkube.TestWorkflowExecution{}, fmt.Errorf("failed to convert execution: %w", err)
	}

	return *execution, nil
}

// GetLatestByTestWorkflow returns latest execution for a workflow
func (r *PostgresRepository) GetLatestByTestWorkflow(ctx context.Context, workflowName string, sortBy testworkflow.LatestSortBy) (*testkube.TestWorkflowExecution, error) {
	// Get complete execution data with all related data in a single query
	row, err := r.queries.GetLatestTestWorkflowExecutionByTestWorkflow(ctx, sqlc.GetLatestTestWorkflowExecutionByTestWorkflowParams{
		WorkflowName: workflowName,
		SortByNumber: sortBy == testworkflow.LatestSortByNumber,
	})
	if err != nil {
		return nil, err
	}

	// Convert the complete row to execution object
	execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
	if err != nil {
		return nil, fmt.Errorf("failed to convert execution: %w", err)
	}

	return execution, nil
}

// GetLatestByTestWorkflows returns latest executions for multiple workflows
func (r *PostgresRepository) GetLatestByTestWorkflows(ctx context.Context, workflowNames []string) ([]testkube.TestWorkflowExecutionSummary, error) {
	if len(workflowNames) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetLatestTestWorkflowExecutionsByTestWorkflows(ctx, workflowNames)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecutionSummary, len(rows))
	for i, row := range rows {
		execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
		if err != nil {
			return nil, err
		}

		result[i] = r.executionToSummary(*execution)
	}

	return result, nil
}

// GetRunning returns running executions
func (r *PostgresRepository) GetRunning(ctx context.Context) ([]testkube.TestWorkflowExecution, error) {
	rows, err := r.queries.GetRunningTestWorkflowExecutions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecution, len(rows))
	for i, row := range rows {
		execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
		if err != nil {
			return nil, err
		}

		result[i] = *execution
	}

	return result, nil
}

// GetFinished returns finished executions with filter
func (r *PostgresRepository) GetFinished(ctx context.Context, filter testworkflow.Filter) ([]testkube.TestWorkflowExecution, error) {
	params, err := r.buildTestWorkflowExecutionParams(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetFinishedTestWorkflowExecutions(ctx, sqlc.GetFinishedTestWorkflowExecutionsParams(params))
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecution, len(rows))
	for i, row := range rows {
		execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
		if err != nil {
			return nil, err
		}

		result[i] = *execution
	}

	return result, nil
}

// GetExecutionsTotals returns execution totals with filter
func (r *PostgresRepository) GetExecutionsTotals(ctx context.Context, filter ...testworkflow.Filter) (testkube.ExecutionsTotals, error) {
	var params sqlc.GetTestWorkflowExecutionsTotalsParams
	var err error
	if len(filter) > 0 {
		params, err = r.buildTestWorkflowExecutionTotalParams(filter[0])
		if err != nil {
			return testkube.ExecutionsTotals{}, err
		}
	}

	rows, err := r.queries.GetTestWorkflowExecutionsTotals(ctx, params)
	if err != nil {
		return testkube.ExecutionsTotals{}, err
	}

	totals := testkube.ExecutionsTotals{}
	var sum int32

	for _, row := range rows {
		count := int32(row.Count)
		sum += count

		if !row.Status.Valid {
			continue
		}

		switch testkube.TestWorkflowStatus(row.Status.String) {
		case testkube.QUEUED_TestWorkflowStatus, testkube.PENDING_TestWorkflowStatus, testkube.STARTING_TestWorkflowStatus, testkube.SCHEDULING_TestWorkflowStatus:
			totals.Queued = count
		case testkube.RUNNING_TestWorkflowStatus, testkube.PAUSING_TestWorkflowStatus, testkube.PAUSED_TestWorkflowStatus, testkube.RESUMING_TestWorkflowStatus, testkube.STOPPING_TestWorkflowStatus:
			totals.Running = count
		case testkube.PASSED_TestWorkflowStatus:
			totals.Passed = count
		case testkube.FAILED_TestWorkflowStatus, testkube.ABORTED_TestWorkflowStatus, testkube.CANCELED_TestWorkflowStatus:
			totals.Failed = count
		}
	}
	totals.Results = sum

	return totals, nil
}

// GetExecutions returns executions with filter
func (r *PostgresRepository) GetExecutions(ctx context.Context, filter testworkflow.Filter) ([]testkube.TestWorkflowExecution, error) {
	params, err := r.buildTestWorkflowExecutionParams(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetTestWorkflowExecutions(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecution, len(rows))
	for i, row := range rows {
		execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
		if err != nil {
			return nil, err
		}

		result[i] = *execution
	}

	return result, nil
}

// GetExecutionsSummary method
func (r *PostgresRepository) GetExecutionsSummary(ctx context.Context, filter testworkflow.Filter) ([]testkube.TestWorkflowExecutionSummary, error) {
	params, err := r.buildTestWorkflowExecutionParams(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.queries.GetTestWorkflowExecutionsSummary(ctx, sqlc.GetTestWorkflowExecutionsSummaryParams(params))
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecutionSummary, len(rows))
	for i, row := range rows {
		execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
		if err != nil {
			return nil, err
		}

		result[i] = r.executionToSummary(*execution)
	}

	return result, nil
}

// Insert inserts new execution
func (r *PostgresRepository) Insert(ctx context.Context, result testkube.TestWorkflowExecution) error {
	execution := result.Clone()
	execution.EscapeDots()

	if execution.Reports == nil {
		execution.Reports = []testkube.TestWorkflowReport{}
	}

	return r.insertExecutionWithTransaction(ctx, execution)
}

func (r *PostgresRepository) insertExecutionWithTransaction(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Insert main execution
	err = r.insertMainExecution(ctx, qtx, execution)
	if err != nil {
		return err
	}

	// Insert related data
	if err = r.insertSignatures(ctx, qtx, execution.Id, execution.Signature, 0); err != nil {
		return err
	}

	if execution.Result != nil {
		if err = r.insertResult(ctx, qtx, execution.Id, execution.Result); err != nil {
			return err
		}
	}

	if err = r.insertOutputs(ctx, qtx, execution.Id, execution.Output); err != nil {
		return err
	}

	if err = r.insertReports(ctx, qtx, execution.Id, execution.Reports); err != nil {
		return err
	}

	if execution.ResourceAggregations != nil {
		if err = r.insertResourceAggregations(ctx, qtx, execution.Id, execution.ResourceAggregations); err != nil {
			return err
		}
	}

	if execution.Workflow != nil {
		if err = r.insertWorkflow(ctx, qtx, execution.Id, "workflow", execution.Workflow); err != nil {
			return err
		}
	}

	if execution.ResolvedWorkflow != nil {
		if err = r.insertWorkflow(ctx, qtx, execution.Id, "resolved_workflow", execution.ResolvedWorkflow); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) insertMainExecution(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, execution *testkube.TestWorkflowExecution) error {
	runnerTarget, err := toJSONB(execution.RunnerTarget)
	if err != nil {
		return err
	}

	runnerOriginalTarget, err := toJSONB(execution.RunnerOriginalTarget)
	if err != nil {
		return err
	}

	tags, err := toJSONB(execution.Tags)
	if err != nil {
		return err
	}

	runningContext, err := toJSONB(execution.RunningContext)
	if err != nil {
		return err
	}

	configParams, err := toJSONB(execution.ConfigParams)
	if err != nil {
		return err
	}

	return qtx.InsertTestWorkflowExecution(ctx, sqlc.InsertTestWorkflowExecutionParams{
		ID:                        execution.Id,
		GroupID:                   toPgText(execution.GroupId),
		RunnerID:                  toPgText(execution.RunnerId),
		RunnerTarget:              runnerTarget,
		RunnerOriginalTarget:      runnerOriginalTarget,
		Name:                      execution.Name,
		Namespace:                 toPgText(execution.Namespace),
		Number:                    toPgInt4(execution.Number),
		ScheduledAt:               toPgTimestamp(execution.ScheduledAt),
		AssignedAt:                toPgTimestamp(execution.AssignedAt),
		StatusAt:                  toPgTimestamp(execution.StatusAt),
		TestWorkflowExecutionName: toPgText(execution.TestWorkflowExecutionName),
		DisableWebhooks:           toPgBool(execution.DisableWebhooks),
		Tags:                      tags,
		RunningContext:            runningContext,
		ConfigParams:              configParams,
	})
}

func (r *PostgresRepository) insertSignatures(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string, signatures []testkube.TestWorkflowSignature, parentId int32) error {
	for _, sig := range signatures {
		var parentIdPg pgtype.Int4
		if parentId > 0 {
			parentIdPg = toPgInt4(parentId)
		}

		id, err := qtx.InsertTestWorkflowSignature(ctx, sqlc.InsertTestWorkflowSignatureParams{
			ExecutionID: executionId,
			Ref:         toPgText(sig.Ref),
			Name:        toPgText(sig.Name),
			Category:    toPgText(sig.Category),
			Optional:    toPgBool(sig.Optional),
			Negative:    toPgBool(sig.Negative),
			ParentID:    parentIdPg,
		})
		if err != nil {
			return err
		}

		//For children, we would need to get the inserted ID and use it as parentId
		// This requires modification to return the ID from the insert
		if len(sig.Children) > 0 {
			// TODO: Implement recursive insertion for children
			// This would require getting the ID of the just inserted signature
			if err = r.insertSignatures(ctx, qtx, executionId, sig.Children, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *PostgresRepository) insertResult(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string, result *testkube.TestWorkflowResult) error {
	pauses, err := toJSONB(result.Pauses)
	if err != nil {
		return err
	}

	initialization, err := toJSONB(result.Initialization)
	if err != nil {
		return err
	}

	steps, err := toJSONB(result.Steps)
	if err != nil {
		return err
	}

	var status, predictedStatus pgtype.Text
	if result.Status != nil {
		status = toPgText(string(*result.Status))
	}
	if result.PredictedStatus != nil {
		predictedStatus = toPgText(string(*result.PredictedStatus))
	}

	return qtx.InsertTestWorkflowResult(ctx, sqlc.InsertTestWorkflowResultParams{
		ExecutionID:     executionId,
		Status:          status,
		PredictedStatus: predictedStatus,
		QueuedAt:        toPgTimestamp(result.QueuedAt),
		StartedAt:       toPgTimestamp(result.StartedAt),
		FinishedAt:      toPgTimestamp(result.FinishedAt),
		Duration:        toPgText(result.Duration),
		TotalDuration:   toPgText(result.TotalDuration),
		DurationMs:      toPgInt4(result.DurationMs),
		PausedMs:        toPgInt4(result.PausedMs),
		TotalDurationMs: toPgInt4(result.TotalDurationMs),
		Pauses:          pauses,
		Initialization:  initialization,
		Steps:           steps,
	})
}

func (r *PostgresRepository) insertOutputs(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string, outputs []testkube.TestWorkflowOutput) error {
	for _, output := range outputs {
		value, err := toJSONB(output.Value)
		if err != nil {
			return err
		}

		err = qtx.InsertTestWorkflowOutput(ctx, sqlc.InsertTestWorkflowOutputParams{
			ExecutionID: executionId,
			Ref:         toPgText(output.Ref),
			Name:        toPgText(output.Name),
			Value:       value,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) deleteSignatures(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string) error {
	if err := qtx.DeleteTestWorkflowSignatures(ctx, executionId); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) deleteResult(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string) error {
	if err := qtx.DeleteTestWorkflowResult(ctx, executionId); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) deleteOutputs(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string) error {
	if err := qtx.DeleteTestWorkflowOutputs(ctx, executionId); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) deleteReports(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string) error {
	if err := qtx.DeleteTestWorkflowReports(ctx, executionId); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) DeleteResourceAggregations(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string) error {
	if err := qtx.DeleteTestWorkflowResourceAggregations(ctx, executionId); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) deleteTestWorkflow(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string, workflowType string) error {
	params := sqlc.DeleteTestWorkflowParams{
		ExecutionID:  executionId,
		WorkflowType: workflowType,
	}
	if err := qtx.DeleteTestWorkflow(ctx, params); err != nil {
		return err
	}

	return nil
}

func (r *PostgresRepository) insertReports(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string, reports []testkube.TestWorkflowReport) error {
	for _, report := range reports {
		summary, err := toJSONB(report.Summary)
		if err != nil {
			return err
		}

		err = qtx.InsertTestWorkflowReport(ctx, sqlc.InsertTestWorkflowReportParams{
			ExecutionID: executionId,
			Ref:         toPgText(report.Ref),
			Kind:        toPgText(report.Kind),
			File:        toPgText(report.File),
			Summary:     summary,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) insertResourceAggregations(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId string, agg *testkube.TestWorkflowExecutionResourceAggregationsReport) error {
	global, err := toJSONB(agg.Global)
	if err != nil {
		return err
	}

	step, err := toJSONB(agg.Step)
	if err != nil {
		return err
	}

	return qtx.InsertTestWorkflowResourceAggregations(ctx, sqlc.InsertTestWorkflowResourceAggregationsParams{
		ExecutionID: executionId,
		Global:      global,
		Step:        step,
	})
}

func (r *PostgresRepository) insertWorkflow(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, executionId, workflowType string, workflow *testkube.TestWorkflow) error {
	labels, err := toJSONB(workflow.Labels)
	if err != nil {
		return err
	}

	annotations, err := toJSONB(workflow.Annotations)
	if err != nil {
		return err
	}

	spec, err := toJSONB(workflow.Spec)
	if err != nil {
		return err
	}

	status, err := toJSONB(workflow.Status)
	if err != nil {
		return err
	}

	return qtx.InsertTestWorkflow(ctx, sqlc.InsertTestWorkflowParams{
		ExecutionID:  executionId,
		WorkflowType: workflowType,
		Name:         toPgText(workflow.Name),
		Namespace:    toPgText(workflow.Namespace),
		Description:  toPgText(workflow.Description),
		Labels:       labels,
		Annotations:  annotations,
		Created:      toPgTimestamp(workflow.Created),
		Updated:      toPgTimestamp(workflow.Updated),
		Spec:         spec,
		ReadOnly:     toPgBool(workflow.ReadOnly),
		Status:       status,
	})
}

// Update updates execution
func (r *PostgresRepository) Update(ctx context.Context, result testkube.TestWorkflowExecution) error {
	execution := result.Clone()
	execution.EscapeDots()

	if execution.Reports == nil {
		execution.Reports = []testkube.TestWorkflowReport{}
	}

	// For update, we need to delete and re-insert related data
	return r.updateExecutionWithTransaction(ctx, execution)
}

func (r *PostgresRepository) updateExecutionWithTransaction(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update main execution (we need to create this query)
	err = r.updateMainExecution(ctx, qtx, execution)
	if err != nil {
		return err
	}

	// Delete and re-insert related data
	// Delete existing signatures, outputs, reports
	// (These would need to be implemented as additional queries)
	if err = r.deleteSignatures(ctx, qtx, execution.Id); err != nil {
		return err
	}

	if err = r.deleteResult(ctx, qtx, execution.Id); err != nil {
		return err
	}

	if err = r.deleteOutputs(ctx, qtx, execution.Id); err != nil {
		return err
	}

	if err = r.deleteReports(ctx, qtx, execution.Id); err != nil {
		return err
	}

	if err = r.DeleteResourceAggregations(ctx, qtx, execution.Id); err != nil {
		return err
	}

	if err = r.deleteTestWorkflow(ctx, qtx, execution.Id, "workflow"); err != nil {
		return err
	}

	if err = r.deleteTestWorkflow(ctx, qtx, execution.Id, "resolved_workflow"); err != nil {
		return err
	}

	// Re-insert all related data
	if err = r.insertSignatures(ctx, qtx, execution.Id, execution.Signature, 0); err != nil {
		return err
	}

	if execution.Result != nil {
		if err = r.insertResult(ctx, qtx, execution.Id, execution.Result); err != nil {
			return err
		}
	}

	if err = r.insertOutputs(ctx, qtx, execution.Id, execution.Output); err != nil {
		return err
	}

	if err = r.insertReports(ctx, qtx, execution.Id, execution.Reports); err != nil {
		return err
	}

	if execution.ResourceAggregations != nil {
		if err = r.insertResourceAggregations(ctx, qtx, execution.Id, execution.ResourceAggregations); err != nil {
			return err
		}
	}

	if execution.Workflow != nil {
		if err = r.insertWorkflow(ctx, qtx, execution.Id, "workflow", execution.Workflow); err != nil {
			return err
		}
	}

	if execution.ResolvedWorkflow != nil {
		if err = r.insertWorkflow(ctx, qtx, execution.Id, "resolved_workflow", execution.ResolvedWorkflow); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) UpdateResultStrict(_ context.Context, _, _ string, _ *testkube.TestWorkflowResult) (updated bool, err error) {
	return false, errors.New("not yet implemented")
}

func (r *PostgresRepository) FinishResultStrict(_ context.Context, _, _ string, _ *testkube.TestWorkflowResult) (updated bool, err error) {
	return false, errors.New("not yet implemented")
}

// UpdateResult updates only the result
func (r *PostgresRepository) UpdateResult(ctx context.Context, id string, result *testkube.TestWorkflowResult) error {
	pauses, err := toJSONB(result.Pauses)
	if err != nil {
		return err
	}

	initialization, err := toJSONB(result.Initialization)
	if err != nil {
		return err
	}

	steps, err := toJSONB(result.Steps)
	if err != nil {
		return err
	}

	var status, predictedStatus pgtype.Text
	if result.Status != nil {
		status = toPgText(string(*result.Status))
	}
	if result.PredictedStatus != nil {
		predictedStatus = toPgText(string(*result.PredictedStatus))
	}

	err = r.queries.UpdateTestWorkflowExecutionResult(ctx, sqlc.UpdateTestWorkflowExecutionResultParams{
		ExecutionID:     id,
		Status:          status,
		PredictedStatus: predictedStatus,
		QueuedAt:        toPgTimestamp(result.QueuedAt),
		StartedAt:       toPgTimestamp(result.StartedAt),
		FinishedAt:      toPgTimestamp(result.FinishedAt),
		Duration:        toPgText(result.Duration),
		TotalDuration:   toPgText(result.TotalDuration),
		DurationMs:      toPgInt4(result.DurationMs),
		PausedMs:        toPgInt4(result.PausedMs),
		TotalDurationMs: toPgInt4(result.TotalDurationMs),
		Pauses:          pauses,
		Initialization:  initialization,
		Steps:           steps,
	})
	if err != nil {
		return err
	}

	// Update status_at if result has finished
	if !result.FinishedAt.IsZero() {
		return r.queries.UpdateExecutionStatusAt(ctx, sqlc.UpdateExecutionStatusAtParams{
			ExecutionID: id,
			StatusAt:    toPgTimestamp(result.FinishedAt),
		})
	}

	return nil
}

// UpdateReport adds a report
func (r *PostgresRepository) UpdateReport(ctx context.Context, id string, report *testkube.TestWorkflowReport) error {
	summary, err := toJSONB(report.Summary)
	if err != nil {
		return err
	}

	return r.queries.UpdateTestWorkflowExecutionReport(ctx, sqlc.UpdateTestWorkflowExecutionReportParams{
		ExecutionID: id,
		Ref:         toPgText(report.Ref),
		Kind:        toPgText(report.Kind),
		File:        toPgText(report.File),
		Summary:     summary,
	})
}

// UpdateOutput replaces all outputs
func (r *PostgresRepository) UpdateOutput(ctx context.Context, id string, refs []testkube.TestWorkflowOutput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Delete existing outputs
	err = qtx.DeleteTestWorkflowOutputs(ctx, id)
	if err != nil {
		return err
	}

	// Insert new outputs
	err = r.insertOutputs(ctx, qtx, id, refs)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateResourceAggregations updates resource aggregations
func (r *PostgresRepository) UpdateResourceAggregations(ctx context.Context, id string, resourceAggregations *testkube.TestWorkflowExecutionResourceAggregationsReport) error {
	global, err := toJSONB(resourceAggregations.Global)
	if err != nil {
		return err
	}

	step, err := toJSONB(resourceAggregations.Step)
	if err != nil {
		return err
	}

	return r.queries.UpdateTestWorkflowExecutionResourceAggregations(ctx, sqlc.UpdateTestWorkflowExecutionResourceAggregationsParams{
		ExecutionID: id,
		Global:      global,
		Step:        step,
	})
}

// DeleteByTestWorkflow deletes executions by workflow name
func (r *PostgresRepository) DeleteByTestWorkflow(ctx context.Context, workflowName string) error {
	if r.sequenceRepository != nil {
		err := r.sequenceRepository.DeleteExecutionNumber(ctx, workflowName, sequence.ExecutionTypeTestWorkflow)
		if err != nil {
			return err
		}
	}

	return r.queries.DeleteTestWorkflowExecutionsByTestWorkflow(ctx, workflowName)
}

// DeleteAll deletes all executions
func (r *PostgresRepository) DeleteAll(ctx context.Context) error {
	if r.sequenceRepository != nil {
		err := r.sequenceRepository.DeleteAllExecutionNumbers(ctx, sequence.ExecutionTypeTestWorkflow)
		if err != nil {
			return err
		}
	}

	return r.queries.DeleteAllTestWorkflowExecutions(ctx)
}

// DeleteByTestWorkflows deletes executions by workflow names
func (r *PostgresRepository) DeleteByTestWorkflows(ctx context.Context, workflowNames []string) error {
	if len(workflowNames) == 0 {
		return nil
	}

	if r.sequenceRepository != nil {
		err := r.sequenceRepository.DeleteExecutionNumbers(ctx, workflowNames, sequence.ExecutionTypeTestWorkflow)
		if err != nil {
			return err
		}
	}

	return r.queries.DeleteTestWorkflowExecutionsByTestWorkflows(ctx, workflowNames)
}

// GetNextExecutionNumber gets next execution number
func (r *PostgresRepository) GetNextExecutionNumber(ctx context.Context, name string) (int32, error) {
	if r.sequenceRepository == nil {
		return 0, errors.New("no sequence repository provided")
	}

	return r.sequenceRepository.GetNextExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)
}

// GetTestWorkflowMetrics returns metrics
func (r *PostgresRepository) GetTestWorkflowMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	metrics := testkube.ExecutionsMetrics{}

	var la int32
	if last < 0 || last > math.MaxInt32 {
		la = 0
	} else {
		la = int32(last)
	}

	var li int32
	if limit < 0 || limit > math.MaxInt32 {
		li = 0
	} else {
		li = int32(limit)
	}

	rows, err := r.queries.GetTestWorkflowMetrics(ctx, sqlc.GetTestWorkflowMetricsParams{
		WorkflowName: name,
		LastNDays:    la,
		Lmt:          int32(li),
	})
	if err != nil {
		return metrics, err
	}

	executions := make([]testkube.ExecutionsMetricsExecutions, len(rows))
	for i, row := range rows {
		executions[i] = testkube.ExecutionsMetricsExecutions{
			ExecutionId: row.ExecutionID,
			GroupId:     fromPgText(row.GroupID),
			Duration:    fromPgText(row.Duration),
			DurationMs:  fromPgInt4(row.DurationMs),
			Status:      fromPgText(row.Status),
			Name:        row.Name,
			StartTime:   fromPgTimestamp(row.StartTime),
			RunnerId:    fromPgText(row.RunnerID),
		}
	}

	metrics = common.CalculateMetrics(executions)
	if limit > 0 && limit < len(metrics.Executions) {
		metrics.Executions = metrics.Executions[:limit]
	}

	return metrics, nil
}

// GetPreviousFinishedState gets previous finished state
func (r *PostgresRepository) GetPreviousFinishedState(ctx context.Context, testWorkflowName string, date time.Time) (testkube.TestWorkflowStatus, error) {
	status, err := r.queries.GetPreviousFinishedState(ctx, sqlc.GetPreviousFinishedStateParams{
		WorkflowName: testWorkflowName,
		Date:         toPgTimestamp(date),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	return testkube.TestWorkflowStatus(fromPgText(status)), nil
}

// GetExecutionTags returns execution tags
func (r *PostgresRepository) GetExecutionTags(ctx context.Context, testWorkflowName string) (map[string][]string, error) {
	rows, err := r.queries.GetTestWorkflowExecutionTags(ctx, testWorkflowName)
	if err != nil {
		return nil, err
	}

	tags := make(map[string][]string)
	for _, row := range rows {
		tags[utils.UnescapeDots(row.TagKey)] = row.Values
	}

	return tags, nil
}

// Init initializes execution
func (r *PostgresRepository) Init(ctx context.Context, id string, data testworkflow.InitData) error {
	return r.queries.InitTestWorkflowExecution(ctx, sqlc.InitTestWorkflowExecutionParams{
		ID:        id,
		Namespace: toPgText(data.Namespace),
		RunnerID:  toPgText(data.RunnerID),
	})
}

// Assign assigns execution to runner
func (r *PostgresRepository) Assign(ctx context.Context, id string, prevRunnerId string, newRunnerId string, assignedAt *time.Time) (bool, error) {
	var assignedAtPg pgtype.Timestamptz
	if assignedAt != nil {
		assignedAtPg = toPgTimestamp(*assignedAt)
	}

	resultId, err := r.queries.AssignTestWorkflowExecution(ctx, sqlc.AssignTestWorkflowExecutionParams{
		ID:           id,
		PrevRunnerID: prevRunnerId,
		NewRunnerID:  newRunnerId,
		AssignedAt:   assignedAtPg,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return resultId != "", nil
}

// GetUnassigned returns unassigned executions
func (r *PostgresRepository) GetUnassigned(ctx context.Context) ([]testkube.TestWorkflowExecution, error) {
	rows, err := r.queries.GetUnassignedTestWorkflowExecutions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecution, len(rows))
	for i, row := range rows {
		execution, err := r.convertCompleteRowToExecutionWithRelated(sqlc.GetTestWorkflowExecutionRow(row))
		if err != nil {
			return nil, err
		}

		result[i] = *execution
	}

	return result, nil
}

// AbortIfQueued aborts execution if queued
func (r *PostgresRepository) AbortIfQueued(ctx context.Context, id string) (bool, error) {
	ts := time.Now()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Abort the execution
	resultId, err := qtx.AbortTestWorkflowExecutionIfQueued(ctx, sqlc.AbortTestWorkflowExecutionIfQueuedParams{
		ID:        id,
		AbortTime: toPgTimestamp(ts),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if resultId == "" {
		return false, nil
	}

	// Abort the result
	err = qtx.AbortTestWorkflowResultIfQueued(ctx, sqlc.AbortTestWorkflowResultIfQueuedParams{
		ID:        id,
		AbortTime: toPgTimestamp(ts),
	})
	if err != nil {
		return false, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *PostgresRepository) Count(ctx context.Context, filter testworkflow.Filter) (count int64, err error) {
	params, err := r.buildTestWorkflowExecutionParams(filter)
	if err != nil {
		return 0, err
	}

	return r.queries.CountTestWorkflowExecutions(ctx, sqlc.CountTestWorkflowExecutionsParams(params))
}

// Helper functions for building query parameters and converting rows
// These would need to be implemented based on the specific filter structure
// Enhanced buildTestWorkflowExecutionParams with full filter support
func (r *PostgresRepository) buildTestWorkflowExecutionParams(filter testworkflow.Filter) (sqlc.GetTestWorkflowExecutionsParams, error) {
	var err error
	params := sqlc.GetTestWorkflowExecutionsParams{
		Fst: int32(filter.Page() * filter.PageSize()),
		Lmt: int32(filter.PageSize()),
	}

	// Basic filters
	if filter.NameDefined() {
		params.WorkflowName = filter.Name()
	}

	if filter.NamesDefined() {
		params.WorkflowNames = filter.Names()
	}

	if filter.TextSearchDefined() {
		params.TextSearch = filter.TextSearch()
	}

	// Date filters
	if filter.StartDateDefined() {
		params.StartDate = toPgTimestamp(filter.StartDate())
	}

	if filter.EndDateDefined() {
		params.EndDate = toPgTimestamp(filter.EndDate())
	}

	if filter.LastNDaysDefined() {
		params.LastNDays = int32(filter.LastNDays())
	}

	// Status filters
	if filter.StatusesDefined() {
		statuses := filter.Statuses()
		pgStatuses := []string{}
		for _, status := range statuses {
			pgStatuses = append(pgStatuses, string(status))
		}
		params.Statuses = pgStatuses
	}

	// Runner filters
	if filter.RunnerIDDefined() {
		params.RunnerID = filter.RunnerID()
	}

	params.Assigned = pgtype.Bool{}
	if filter.AssignedDefined() {
		params.Assigned = toPgBool(filter.Assigned())
	}

	// Actor filters
	if filter.ActorNameDefined() {
		params.ActorName = filter.ActorName()
	}

	if filter.ActorTypeDefined() {
		params.ActorType = string(filter.ActorType())
	}

	// Group filter
	if filter.GroupIDDefined() {
		params.GroupID = filter.GroupID()
	}

	// Initialization filter
	params.Initialized = pgtype.Bool{}
	if filter.InitializedDefined() {
		params.Initialized = toPgBool(filter.Initialized())
	}

	if filter.Selector() != "" {
		keys, conditions := r.parseSelector(filter.Selector())
		params.SelectorKeys, err = json.Marshal(keys)
		if err != nil {
			return params, err
		}

		params.SelectorConditions, err = json.Marshal(conditions)
		if err != nil {
			return params, err
		}
	}

	if filter.LabelSelector() != nil {
		keys, conditions := r.parseLabelSelector(filter.LabelSelector())
		params.LabelKeys, err = json.Marshal(keys)
		if err != nil {
			return params, err
		}

		params.LabelConditions, err = json.Marshal(conditions)
		if err != nil {
			return params, err
		}
	}

	if filter.TagSelector() != "" {
		keys, conditions := r.parseTagSelector(filter.TagSelector())
		params.TagKeys, err = json.Marshal(keys)
		if err != nil {
			return params, err
		}

		params.TagConditions, err = json.Marshal(conditions)
		if err != nil {
			return params, err
		}
	}

	if filter.SkipDefined() {
		params.Fst = int32(filter.Skip())
	}

	return params, nil
}

type KeyCondition struct {
	Key      string `json:"key"`
	Operator string `json:"operator"` // "exists" or "not_exists"
}

type ValueCondition struct {
	Key    string   `json:"key"`
	Values []string `json:"values"` // Multiple values for the same key (OR logic within the same key)
}

// Parse selector into conditions
func (r *PostgresRepository) parseSelector(selector string) ([]KeyCondition, []ValueCondition) {
	keys := make([]KeyCondition, 0)
	conditions := make([]ValueCondition, 0)
	values := make(map[string][]string, 0)
	items := strings.Split(selector, ",")
	for _, item := range items {
		elements := strings.Split(item, "=")
		if len(elements) == 2 {
			values[utils.EscapeDots(elements[0])] = append(values[utils.EscapeDots(elements[0])], elements[1])
		} else if len(elements) == 1 {
			condType := "exists"
			keys = append(keys, KeyCondition{
				Operator: condType,
				Key:      utils.EscapeDots(elements[0]),
			})
		}
	}

	for key, value := range values {
		conditions = append(conditions, ValueCondition{
			Key:    key,
			Values: value,
		})
	}

	return keys, conditions
}

// Parse label selector into conditions
func (r *PostgresRepository) parseLabelSelector(labelSelector *testworkflow.LabelSelector) ([]KeyCondition, []ValueCondition) {
	keys := make([]KeyCondition, 0)
	conditions := make([]ValueCondition, 0)
	values := make(map[string][]string, 0)
	for _, label := range labelSelector.Or {
		if label.Value != nil {
			values[utils.EscapeDots(label.Key)] = append(values[utils.EscapeDots(label.Key)], *label.Value)
		} else if label.Exists != nil {
			// Label exists/not exists
			condType := "exists"
			if !*label.Exists {
				condType = "not_exists"
			}
			keys = append(keys, KeyCondition{
				Operator: condType,
				Key:      utils.EscapeDots(label.Key),
			})
		}
	}

	for key, value := range values {
		conditions = append(conditions, ValueCondition{
			Key:    key,
			Values: value,
		})
	}

	return keys, conditions
}

// Parse tag selector into conditions
func (r *PostgresRepository) parseTagSelector(tagSelector string) ([]KeyCondition, []ValueCondition) {
	keys := make([]KeyCondition, 0)
	conditions := make([]ValueCondition, 0)
	values := make(map[string][]string, 0)
	items := strings.Split(tagSelector, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		elements := strings.Split(item, "=")
		if len(elements) == 2 {
			values[utils.EscapeDots(elements[0])] = append(values[utils.EscapeDots(elements[0])], elements[1])
		} else if len(elements) == 1 {
			// Tag exists: tag
			condType := "exists"
			keys = append(keys, KeyCondition{
				Operator: condType,
				Key:      utils.EscapeDots(elements[0]),
			})
		}
	}

	for key, value := range values {
		conditions = append(conditions, ValueCondition{
			Key:    key,
			Values: value,
		})
	}

	return keys, conditions
}

func (r *PostgresRepository) buildTestWorkflowExecutionTotalParams(filter testworkflow.Filter) (sqlc.GetTestWorkflowExecutionsTotalsParams, error) {
	var err error
	params := sqlc.GetTestWorkflowExecutionsTotalsParams{}

	// Basic filters
	if filter.NameDefined() {
		params.WorkflowName = filter.Name()
	}

	if filter.NamesDefined() {
		params.WorkflowNames = filter.Names()
	}

	if filter.TextSearchDefined() {
		params.TextSearch = filter.TextSearch()
	}

	// Date filters
	if filter.StartDateDefined() {
		params.StartDate = toPgTimestamp(filter.StartDate())
	}

	if filter.EndDateDefined() {
		params.EndDate = toPgTimestamp(filter.EndDate())
	}

	if filter.LastNDaysDefined() {
		params.LastNDays = int32(filter.LastNDays())
	}

	// Status filters
	if filter.StatusesDefined() {
		statuses := filter.Statuses()
		pgStatuses := []string{}
		for _, status := range statuses {
			pgStatuses = append(pgStatuses, string(status))
		}
		params.Statuses = pgStatuses
	}

	// Runner filters
	if filter.RunnerIDDefined() {
		params.RunnerID = filter.RunnerID()
	}

	params.Assigned = pgtype.Bool{}
	if filter.AssignedDefined() {
		params.Assigned = toPgBool(filter.Assigned())
	}

	// Actor filters
	if filter.ActorNameDefined() {
		params.ActorName = filter.ActorName()
	}

	if filter.ActorTypeDefined() {
		params.ActorType = string(filter.ActorType())
	}

	// Group filter
	if filter.GroupIDDefined() {
		params.GroupID = filter.GroupID()
	}

	// Initialization filter
	params.Initialized = pgtype.Bool{}
	if filter.InitializedDefined() {
		params.Initialized = toPgBool(filter.Initialized())
	}

	if filter.Selector() != "" {
		keys, conditions := r.parseSelector(filter.Selector())
		params.SelectorKeys, err = json.Marshal(keys)
		if err != nil {
			return params, err
		}

		params.SelectorConditions, err = json.Marshal(conditions)
		if err != nil {
			return params, err
		}
	}

	if filter.LabelSelector() != nil {
		keys, conditions := r.parseLabelSelector(filter.LabelSelector())
		params.LabelKeys, err = json.Marshal(keys)
		if err != nil {
			return params, err
		}

		params.LabelConditions, err = json.Marshal(conditions)
		if err != nil {
			return params, err
		}
	}

	if filter.TagSelector() != "" {
		keys, conditions := r.parseTagSelector(filter.TagSelector())
		params.TagKeys, err = json.Marshal(keys)
		if err != nil {
			return params, err
		}

		params.TagConditions, err = json.Marshal(conditions)
		if err != nil {
			return params, err
		}
	}

	return params, nil
}

// Helper methods for building complex objects

func (r *PostgresRepository) parseExecutionJSONFields(execution *testkube.TestWorkflowExecution, runnerTarget, runnerOriginalTarget, tags, runningContext, configParams []byte) error {
	var err error
	if len(runnerTarget) > 0 {
		execution.RunnerTarget, err = fromJSONB[testkube.ExecutionTarget](runnerTarget)
		if err != nil {
			return err
		}
	}

	if len(runnerOriginalTarget) > 0 {
		execution.RunnerOriginalTarget, err = fromJSONB[testkube.ExecutionTarget](runnerOriginalTarget)
		if err != nil {
			return err
		}
	}

	if len(tags) > 0 {
		json.Unmarshal(tags, &execution.Tags)
	}

	if len(runningContext) > 0 {
		execution.RunningContext, err = fromJSONB[testkube.TestWorkflowRunningContext](runningContext)
		if err != nil {
			return err
		}
	}

	if len(configParams) > 0 {
		json.Unmarshal(configParams, &execution.ConfigParams)
	}

	return nil
}

func (r *PostgresRepository) buildResultFromRow(
	status, predictedStatus pgtype.Text,
	queuedAt, startedAt, finishedAt pgtype.Timestamptz,
	duration, totalDuration pgtype.Text,
	durationMs, pausedMs, totalDurationMs pgtype.Int4,
	pauses, initialization, steps []byte,
) (*testkube.TestWorkflowResult, error) {
	var err error
	result := &testkube.TestWorkflowResult{
		QueuedAt:        fromPgTimestamp(queuedAt),
		StartedAt:       fromPgTimestamp(startedAt),
		FinishedAt:      fromPgTimestamp(finishedAt),
		Duration:        fromPgText(duration),
		TotalDuration:   fromPgText(totalDuration),
		DurationMs:      fromPgInt4(durationMs),
		PausedMs:        fromPgInt4(pausedMs),
		TotalDurationMs: fromPgInt4(totalDurationMs),
	}

	if status.Valid {
		s := testkube.TestWorkflowStatus(status.String)
		result.Status = &s
	}

	if predictedStatus.Valid {
		ps := testkube.TestWorkflowStatus(predictedStatus.String)
		result.PredictedStatus = &ps
	}

	if len(pauses) > 0 {
		json.Unmarshal(pauses, &result.Pauses)
	}

	if len(initialization) > 0 {
		result.Initialization, err = fromJSONB[testkube.TestWorkflowStepResult](initialization)
		if err != nil {
			return nil, err
		}
	}

	if len(steps) > 0 {
		json.Unmarshal(steps, &result.Steps)
	}

	return result, nil
}

func (r *PostgresRepository) buildWorkflowFromRow(
	name, namespace, description pgtype.Text,
	labels, annotations []byte,
	created, updated pgtype.Timestamptz,
	spec []byte,
	readOnly pgtype.Bool,
	status []byte,
) (*testkube.TestWorkflow, error) {
	var err error
	workflow := &testkube.TestWorkflow{
		Name:        fromPgText(name),
		Namespace:   fromPgText(namespace),
		Description: fromPgText(description),
		Created:     fromPgTimestamp(created),
		Updated:     fromPgTimestamp(updated),
		ReadOnly:    fromPgBool(readOnly),
	}

	if len(labels) > 0 {
		json.Unmarshal(labels, &workflow.Labels)
	}

	if len(annotations) > 0 {
		json.Unmarshal(annotations, &workflow.Annotations)
	}

	if len(spec) > 0 {
		workflow.Spec, err = fromJSONB[testkube.TestWorkflowSpec](spec)
		if err != nil {
			return nil, err
		}
	}

	if len(status) > 0 {
		workflow.Status, err = fromJSONB[testkube.TestWorkflowStatusSummary](status)
		if err != nil {
			return nil, err
		}
	}

	return workflow, nil
}

func (r *PostgresRepository) updateMainExecution(ctx context.Context, qtx sqlc.TestWorkflowExecutionQueriesInterface, execution *testkube.TestWorkflowExecution) error {
	runnerTarget, err := toJSONB(execution.RunnerTarget)
	if err != nil {
		return err
	}

	runnerOriginalTarget, err := toJSONB(execution.RunnerOriginalTarget)
	if err != nil {
		return err
	}

	tags, err := toJSONB(execution.Tags)
	if err != nil {
		return err
	}

	runningContext, err := toJSONB(execution.RunningContext)
	if err != nil {
		return err
	}

	configParams, err := toJSONB(execution.ConfigParams)
	if err != nil {
		return err
	}

	// Placeholder - you would call the generated method here:
	return qtx.UpdateTestWorkflowExecution(ctx, sqlc.UpdateTestWorkflowExecutionParams{
		GroupID:                   toPgText(execution.GroupId),
		RunnerID:                  toPgText(execution.RunnerId),
		RunnerTarget:              runnerTarget,
		RunnerOriginalTarget:      runnerOriginalTarget,
		Name:                      execution.Name,
		Namespace:                 toPgText(execution.Namespace),
		Number:                    toPgInt4(execution.Number),
		ScheduledAt:               toPgTimestamp(execution.ScheduledAt),
		AssignedAt:                toPgTimestamp(execution.AssignedAt),
		StatusAt:                  toPgTimestamp(execution.StatusAt),
		TestWorkflowExecutionName: toPgText(execution.TestWorkflowExecutionName),
		DisableWebhooks:           toPgBool(execution.DisableWebhooks),
		Tags:                      tags,
		RunningContext:            runningContext,
		ConfigParams:              configParams,
		ID:                        execution.Id,
	})
}

// populateConfigParams - same as in mongo repository
func populateConfigParams(resolvedWorkflow *testkube.TestWorkflow, configParams map[string]testkube.TestWorkflowExecutionConfigValue) map[string]testkube.TestWorkflowExecutionConfigValue {
	if configParams == nil {
		configParams = make(map[string]testkube.TestWorkflowExecutionConfigValue)
	}

	for k, v := range resolvedWorkflow.Spec.Config {
		if v.Sensitive {
			configParams[k] = testkube.TestWorkflowExecutionConfigValue{
				Sensitive:         true,
				EmptyValue:        true,
				EmptyDefaultValue: true,
			}
			continue
		}

		if _, ok := configParams[k]; !ok {
			configParams[k] = testkube.TestWorkflowExecutionConfigValue{
				EmptyValue: true,
			}
		}

		data := configParams[k]
		if len(data.Value) > configParamSizeLimit {
			data.Value = data.Value[:configParamSizeLimit]
			data.Truncated = true
		}

		if v.Default_ != nil {
			data.DefaultValue = v.Default_.Value
		} else {
			data.EmptyDefaultValue = true
		}

		configParams[k] = data
	}

	return configParams
}
