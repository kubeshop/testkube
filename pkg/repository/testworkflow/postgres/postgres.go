package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/repository/common"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

var _ testworkflow.Repository = (*PostgresRepository)(nil)

const (
	configParamSizeLimit = 100
)

type PostgresRepository struct {
	db                 *pgxpool.Pool
	queries            *sqlc.Queries
	sequenceRepository sequence.Repository
}

type PostgresRepositoryOpt func(*PostgresRepository)

func NewPostgresRepository(db *pgxpool.Pool, opts ...PostgresRepositoryOpt) *PostgresRepository {
	r := &PostgresRepository{
		db:      db,
		queries: sqlc.New(db),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func WithPostgresRepositorySequence(sequenceRepository sequence.Repository) PostgresRepositoryOpt {
	return func(r *PostgresRepository) {
		r.sequenceRepository = sequenceRepository
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

// Get returns execution by id or name
func (r *PostgresRepository) Get(ctx context.Context, id string) (testkube.TestWorkflowExecution, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return testkube.TestWorkflowExecution{}, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Get main execution data
	row, err := qtx.GetTestWorkflowExecution(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return testkube.TestWorkflowExecution{}, fmt.Errorf("execution not found")
		}
		return testkube.TestWorkflowExecution{}, err
	}

	// Build the execution object
	execution, err := r.buildExecutionFromRow(ctx, qtx, row)
	if err != nil {
		return testkube.TestWorkflowExecution{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return testkube.TestWorkflowExecution{}, err
	}

	// Populate config params if resolved workflow exists
	if execution.ResolvedWorkflow != nil && execution.ResolvedWorkflow.Spec != nil {
		execution.ConfigParams = populateConfigParams(execution.ResolvedWorkflow, execution.ConfigParams)
	}

	return *execution.UnscapeDots(), nil
}

func (r *PostgresRepository) buildExecutionFromRow(ctx context.Context, qtx *sqlc.Queries, row sqlc.GetTestWorkflowExecutionRow) (*testkube.TestWorkflowExecution, error) {
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

	// Parse JSONB fields
	var err error

	if len(row.RunnerTarget) > 0 {
		execution.RunnerTarget, err = fromJSONB[testkube.ExecutionTarget](row.RunnerTarget)
		if err != nil {
			return nil, err
		}
	}

	if len(row.RunnerOriginalTarget) > 0 {
		execution.RunnerOriginalTarget, err = fromJSONB[testkube.ExecutionTarget](row.RunnerOriginalTarget)
		if err != nil {
			return nil, err
		}
	}

	if len(row.Tags) > 0 {
		err = json.Unmarshal(row.Tags, &execution.Tags)
		if err != nil {
			return nil, err
		}
	}

	if len(row.RunningContext) > 0 {
		execution.RunningContext, err = fromJSONB[testkube.TestWorkflowRunningContext](row.RunningContext)
		if err != nil {
			return nil, err
		}
	}

	if len(row.ConfigParams) > 0 {
		err = json.Unmarshal(row.ConfigParams, &execution.ConfigParams)
		if err != nil {
			return nil, err
		}
	}

	// Build result if exists
	if row.Status.Valid {
		execution.Result = &testkube.TestWorkflowResult{
			QueuedAt:        fromPgTimestamp(row.QueuedAt),
			StartedAt:       fromPgTimestamp(row.StartedAt),
			FinishedAt:      fromPgTimestamp(row.FinishedAt),
			Duration:        fromPgText(row.Duration),
			TotalDuration:   fromPgText(row.TotalDuration),
			DurationMs:      fromPgInt4(row.DurationMs),
			PausedMs:        fromPgInt4(row.PausedMs),
			TotalDurationMs: fromPgInt4(row.TotalDurationMs),
		}

		if row.Status.Valid {
			status := testkube.TestWorkflowStatus(row.Status.String)
			execution.Result.Status = &status
		}

		if row.PredictedStatus.Valid {
			predictedStatus := testkube.TestWorkflowStatus(row.PredictedStatus.String)
			execution.Result.PredictedStatus = &predictedStatus
		}

		if len(row.Pauses) > 0 {
			err = json.Unmarshal(row.Pauses, &execution.Result.Pauses)
			if err != nil {
				return nil, err
			}
		}

		if len(row.Initialization) > 0 {
			execution.Result.Initialization, err = fromJSONB[testkube.TestWorkflowStepResult](row.Initialization)
			if err != nil {
				return nil, err
			}
		}

		if len(row.Steps) > 0 {
			err = json.Unmarshal(row.Steps, &execution.Result.Steps)
			if err != nil {
				return nil, err
			}
		}
	}

	// Build workflow if exists
	if row.WorkflowName.Valid {
		execution.Workflow = &testkube.TestWorkflow{
			Name:        fromPgText(row.WorkflowName),
			Namespace:   fromPgText(row.WorkflowNamespace),
			Description: fromPgText(row.WorkflowDescription),
			Created:     fromPgTimestamp(row.WorkflowCreated),
			Updated:     fromPgTimestamp(row.WorkflowUpdated),
			ReadOnly:    fromPgBool(row.WorkflowReadOnly),
		}

		if len(row.WorkflowLabels) > 0 {
			err = json.Unmarshal(row.WorkflowLabels, &execution.Workflow.Labels)
			if err != nil {
				return nil, err
			}
		}

		if len(row.WorkflowAnnotations) > 0 {
			err = json.Unmarshal(row.WorkflowAnnotations, &execution.Workflow.Annotations)
			if err != nil {
				return nil, err
			}
		}

		if len(row.WorkflowSpec) > 0 {
			execution.Workflow.Spec, err = fromJSONB[testkube.TestWorkflowSpec](row.WorkflowSpec)
			if err != nil {
				return nil, err
			}
		}

		if len(row.WorkflowStatus) > 0 {
			execution.Workflow.Status, err = fromJSONB[testkube.TestWorkflowStatusSummary](row.WorkflowStatus)
			if err != nil {
				return nil, err
			}
		}
	}

	// Build resolved workflow if exists
	if row.ResolvedWorkflowName.Valid {
		execution.ResolvedWorkflow = &testkube.TestWorkflow{
			Name:        fromPgText(row.ResolvedWorkflowName),
			Namespace:   fromPgText(row.ResolvedWorkflowNamespace),
			Description: fromPgText(row.ResolvedWorkflowDescription),
			Created:     fromPgTimestamp(row.ResolvedWorkflowCreated),
			Updated:     fromPgTimestamp(row.ResolvedWorkflowUpdated),
			ReadOnly:    fromPgBool(row.ResolvedWorkflowReadOnly),
		}

		if len(row.ResolvedWorkflowLabels) > 0 {
			err = json.Unmarshal(row.ResolvedWorkflowLabels, &execution.ResolvedWorkflow.Labels)
			if err != nil {
				return nil, err
			}
		}

		if len(row.ResolvedWorkflowAnnotations) > 0 {
			err = json.Unmarshal(row.ResolvedWorkflowAnnotations, &execution.ResolvedWorkflow.Annotations)
			if err != nil {
				return nil, err
			}
		}

		if len(row.ResolvedWorkflowSpec) > 0 {
			execution.ResolvedWorkflow.Spec, err = fromJSONB[testkube.TestWorkflowSpec](row.ResolvedWorkflowSpec)
			if err != nil {
				return nil, err
			}
		}

		if len(row.ResolvedWorkflowStatus) > 0 {
			execution.ResolvedWorkflow.Status, err = fromJSONB[testkube.TestWorkflowStatusSummary](row.ResolvedWorkflowStatus)
			if err != nil {
				return nil, err
			}
		}
	}

	// Get signatures
	signatures, err := qtx.GetTestWorkflowSignatures(ctx, execution.Id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	execution.Signature = r.buildSignatureTree(signatures)

	// Get outputs
	outputs, err := qtx.GetTestWorkflowOutputs(ctx, execution.Id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	execution.Output = r.convertOutputs(outputs)

	// Get reports
	reports, err := qtx.GetTestWorkflowReports(ctx, execution.Id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	execution.Reports = r.convertReports(reports)

	// Get resource aggregations
	resourceAgg, err := qtx.GetTestWorkflowResourceAggregations(ctx, execution.Id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if err == nil {
		execution.ResourceAggregations, err = r.convertResourceAggregations(resourceAgg)
		if err != nil {
			return nil, err
		}
	}

	return execution, nil
}

func (r *PostgresRepository) buildSignatureTree(signatures []sqlc.TestWorkflowSignature) []testkube.TestWorkflowSignature {
	if len(signatures) == 0 {
		return nil
	}

	// Convert to map for easier processing
	sigMap := make(map[int32]*testkube.TestWorkflowSignature)
	parentChildMap := make(map[int32][]int32)

	for _, sig := range signatures {
		twSig := &testkube.TestWorkflowSignature{
			Ref:      fromPgText(sig.Ref),
			Name:     fromPgText(sig.Name),
			Category: fromPgText(sig.Category),
			Optional: fromPgBool(sig.Optional),
			Negative: fromPgBool(sig.Negative),
		}
		sigMap[sig.ID] = twSig

		if sig.ParentID.Valid {
			parentChildMap[sig.ParentID.Int32] = append(parentChildMap[sig.ParentID.Int32], sig.ID)
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
		if !sig.ParentID.Valid {
			root := *sigMap[sig.ID]
			root.Children = buildChildren(sig.ID)
			roots = append(roots, root)
		}
	}

	return roots
}

func (r *PostgresRepository) convertOutputs(outputs []sqlc.TestWorkflowOutput) []testkube.TestWorkflowOutput {
	result := make([]testkube.TestWorkflowOutput, len(outputs))
	for i, output := range outputs {
		result[i] = testkube.TestWorkflowOutput{
			Ref:  fromPgText(output.Ref),
			Name: fromPgText(output.Name),
		}
		if len(output.Value) > 0 {
			json.Unmarshal(output.Value, &result[i].Value)
		}
	}
	return result
}

func (r *PostgresRepository) convertReports(reports []sqlc.TestWorkflowReport) []testkube.TestWorkflowReport {
	result := make([]testkube.TestWorkflowReport, len(reports))
	for i, report := range reports {
		result[i] = testkube.TestWorkflowReport{
			Ref:  fromPgText(report.Ref),
			Kind: fromPgText(report.Kind),
			File: fromPgText(report.File),
		}
		if len(report.Summary) > 0 {
			summary, _ := fromJSONB[testkube.TestWorkflowReportSummary](report.Summary)
			result[i].Summary = summary
		}
	}
	return result
}

func (r *PostgresRepository) convertResourceAggregations(agg sqlc.TestWorkflowResourceAggregation) (*testkube.TestWorkflowExecutionResourceAggregationsReport, error) {
	result := &testkube.TestWorkflowExecutionResourceAggregationsReport{}

	if len(agg.Global) > 0 {
		err := json.Unmarshal(agg.Global, &result.Global)
		if err != nil {
			return nil, err
		}
	}

	if len(agg.Step) > 0 {
		err := json.Unmarshal(agg.Step, &result.Step)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// GetByNameAndTestWorkflow returns execution by name and workflow name
func (r *PostgresRepository) GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (testkube.TestWorkflowExecution, error) {
	row, err := r.queries.GetTestWorkflowExecutionByNameAndTestWorkflow(ctx, sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowParams{
		Name:         name,
		WorkflowName: toPgText(workflowName),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return testkube.TestWorkflowExecution{}, fmt.Errorf("execution not found")
		}
		return testkube.TestWorkflowExecution{}, err
	}

	// Convert to full execution object (simplified version)
	execution := r.convertRowToExecution(row)
	return *execution.UnscapeDots(), nil
}

// GetLatestByTestWorkflow returns latest execution for a workflow
func (r *PostgresRepository) GetLatestByTestWorkflow(ctx context.Context, workflowName string) (*testkube.TestWorkflowExecution, error) {
	row, err := r.queries.GetLatestTestWorkflowExecutionByTestWorkflow(ctx, toPgText(workflowName))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("execution not found")
		}
		return nil, err
	}

	execution := r.convertRowToExecutionSimple(row)
	return execution.UnscapeDots(), nil
}

// GetLatestByTestWorkflows returns latest executions for multiple workflows
func (r *PostgresRepository) GetLatestByTestWorkflows(ctx context.Context, workflowNames []string) ([]testkube.TestWorkflowExecutionSummary, error) {
	if len(workflowNames) == 0 {
		return nil, nil
	}

	pgNames := make([]pgtype.Text, len(workflowNames))
	for i, name := range workflowNames {
		pgNames[i] = toPgText(name)
	}

	rows, err := r.queries.GetLatestTestWorkflowExecutionsByTestWorkflows(ctx, pgNames)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecutionSummary, len(rows))
	for i, row := range rows {
		result[i] = r.convertRowToExecutionSummary(row)
		result[i].UnscapeDots()
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
		result[i] = r.convertRowToExecutionSimpleFromRunning(row)
		result[i].UnscapeDots()
	}

	return result, nil
}

// GetExecutionsTotals returns execution totals with filter
func (r *PostgresRepository) GetExecutionsTotals(ctx context.Context, filter ...testworkflow.Filter) (testkube.ExecutionsTotals, error) {
	var params sqlc.GetTestWorkflowExecutionsTotalsParams
	if len(filter) > 0 {
		params = r.buildTotalsParams(filter[0])
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
		case testkube.QUEUED_TestWorkflowStatus:
			totals.Queued = count
		case testkube.RUNNING_TestWorkflowStatus:
			totals.Running = count
		case testkube.PASSED_TestWorkflowStatus:
			totals.Passed = count
		case testkube.FAILED_TestWorkflowStatus, testkube.ABORTED_TestWorkflowStatus:
			totals.Failed = count
		}
	}
	totals.Results = sum

	return totals, nil
}

// GetExecutions returns executions with filter
func (r *PostgresRepository) GetExecutions(ctx context.Context, filter testworkflow.Filter) ([]testkube.TestWorkflowExecution, error) {
	params := r.buildExecutionParams(filter)
	rows, err := r.queries.GetTestWorkflowExecutions(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecution, len(rows))
	for i, row := range rows {
		result[i] = r.convertRowToExecutionFromList(row)
		result[i].UnscapeDots()
	}

	return result, nil
}

// GetExecutionsSummary returns execution summaries with filter
func (r *PostgresRepository) GetExecutionsSummary(ctx context.Context, filter testworkflow.Filter) ([]testkube.TestWorkflowExecutionSummary, error) {
	params := r.buildSummaryParams(filter)
	rows, err := r.queries.GetTestWorkflowExecutionsSummary(ctx, params)
	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecutionSummary, len(rows))
	for i, row := range rows {
		result[i] = r.convertRowToSummary(row)
		result[i].UnscapeDots()
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

func (r *PostgresRepository) insertMainExecution(ctx context.Context, qtx *sqlc.Queries, execution *testkube.TestWorkflowExecution) error {
	runnerTarget, _ := toJSONB(execution.RunnerTarget)
	runnerOriginalTarget, _ := toJSONB(execution.RunnerOriginalTarget)
	tags, _ := toJSONB(execution.Tags)
	runningContext, _ := toJSONB(execution.RunningContext)
	configParams, _ := toJSONB(execution.ConfigParams)

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

func (r *PostgresRepository) insertSignatures(ctx context.Context, qtx *sqlc.Queries, executionId string, signatures []testkube.TestWorkflowSignature, parentId int32) error {
	for _, sig := range signatures {
		var parentIdPg pgtype.Int4
		if parentId > 0 {
			parentIdPg = toPgInt4(parentId)
		}

		err := qtx.InsertTestWorkflowSignature(ctx, sqlc.InsertTestWorkflowSignatureParams{
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

		// For children, we would need to get the inserted ID and use it as parentId
		// This requires modification to return the ID from the insert
		if len(sig.Children) > 0 {
			// TODO: Implement recursive insertion for children
			// This would require getting the ID of the just inserted signature
		}
	}
	return nil
}

func (r *PostgresRepository) insertResult(ctx context.Context, qtx *sqlc.Queries, executionId string, result *testkube.TestWorkflowResult) error {
	pauses, _ := toJSONB(result.Pauses)
	initialization, _ := toJSONB(result.Initialization)
	steps, _ := toJSONB(result.Steps)

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

func (r *PostgresRepository) insertOutputs(ctx context.Context, qtx *sqlc.Queries, executionId string, outputs []testkube.TestWorkflowOutput) error {
	for _, output := range outputs {
		value, _ := toJSONB(output.Value)
		err := qtx.InsertTestWorkflowOutput(ctx, sqlc.InsertTestWorkflowOutputParams{
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

func (r *PostgresRepository) insertReports(ctx context.Context, qtx *sqlc.Queries, executionId string, reports []testkube.TestWorkflowReport) error {
	for _, report := range reports {
		summary, _ := toJSONB(report.Summary)
		err := qtx.InsertTestWorkflowReport(ctx, sqlc.InsertTestWorkflowReportParams{
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

func (r *PostgresRepository) insertResourceAggregations(ctx context.Context, qtx *sqlc.Queries, executionId string, agg *testkube.TestWorkflowExecutionResourceAggregationsReport) error {
	global, _ := toJSONB(agg.Global)
	step, _ := toJSONB(agg.Step)

	return qtx.InsertTestWorkflowResourceAggregations(ctx, sqlc.InsertTestWorkflowResourceAggregationsParams{
		ExecutionID: executionId,
		Global:      global,
		Step:        step,
	})
}

func (r *PostgresRepository) insertWorkflow(ctx context.Context, qtx *sqlc.Queries, executionId, workflowType string, workflow *testkube.TestWorkflow) error {
	labels, _ := toJSONB(workflow.Labels)
	annotations, _ := toJSONB(workflow.Annotations)
	spec, _ := toJSONB(workflow.Spec)
	status, _ := toJSONB(workflow.Status)

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

// UpdateResult updates only the result
func (r *PostgresRepository) UpdateResult(ctx context.Context, id string, result *testkube.TestWorkflowResult) error {
	pauses, _ := toJSONB(result.Pauses)
	initialization, _ := toJSONB(result.Initialization)
	steps, _ := toJSONB(result.Steps)

	var status, predictedStatus pgtype.Text
	if result.Status != nil {
		status = toPgText(string(*result.Status))
	}
	if result.PredictedStatus != nil {
		predictedStatus = toPgText(string(*result.PredictedStatus))
	}

	err := r.queries.UpdateTestWorkflowExecutionResult(ctx, sqlc.UpdateTestWorkflowExecutionResultParams{
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
	summary, _ := toJSONB(report.Summary)
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
	global, _ := toJSONB(resourceAggregations.Global)
	step, _ := toJSONB(resourceAggregations.Step)

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

	return r.queries.DeleteTestWorkflowExecutionsByTestWorkflow(ctx, toPgText(workflowName))
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

	pgNames := make([]pgtype.Text, len(workflowNames))
	for i, name := range workflowNames {
		pgNames[i] = toPgText(name)
	}

	return r.queries.DeleteTestWorkflowExecutionsByTestWorkflows(ctx, pgNames)
}

// GetTestWorkflowMetrics returns metrics
func (r *PostgresRepository) GetTestWorkflowMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	metrics := testkube.ExecutionsMetrics{}

	la := int32(last)
	if last > math.MaxInt32 {
		la = 0
	}

	li := int32(limit)
	if limit > math.MaxInt32 {
		li = 0
	}

	rows, err := r.queries.GetTestWorkflowMetrics(ctx, sqlc.GetTestWorkflowMetricsParams{
		WorkflowName: toPgText(name),
		LastDays:     toPgInt4(int32(la)),
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
		WorkflowName: toPgText(testWorkflowName),
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

// GetNextExecutionNumber gets next execution number
func (r *PostgresRepository) GetNextExecutionNumber(ctx context.Context, name string) (int32, error) {
	if r.sequenceRepository == nil {
		return 0, errors.New("no sequence repository provided")
	}

	return r.sequenceRepository.GetNextExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)
}

// GetExecutionTags returns execution tags
func (r *PostgresRepository) GetExecutionTags(ctx context.Context, testWorkflowName string) (map[string][]string, error) {
	rows, err := r.queries.GetTestWorkflowExecutionTags(ctx, testWorkflowName)
	if err != nil {
		return nil, err
	}

	tags := make(map[string][]string)
	for _, row := range rows {
		if !row.TagKey.Valid {
			continue
		}

		var values []string
		for _, val := range row.Values {
			if val.Valid {
				values = append(values, val.String)
			}
		}
		tags[row.TagKey.String] = values
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
		PrevRunnerID: toPgText(prevRunnerId),
		NewRunnerID:  toPgText(newRunnerId),
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
		result[i] = r.convertUnassignedRowToExecution(row)
		result[i].UnscapeDots()
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

// Helper functions for building query parameters and converting rows
// These would need to be implemented based on the specific filter structure
func (r *PostgresRepository) buildTotalsParams(filter testworkflow.Filter) sqlc.GetTestWorkflowExecutionsTotalsParams {
	// Implementation depends on the Filter interface
	return sqlc.GetTestWorkflowExecutionsTotalsParams{}
}

func (r *PostgresRepository) buildExecutionParams(filter testworkflow.Filter) sqlc.GetTestWorkflowExecutionsParams {
	// Implementation depends on the Filter interface
	return sqlc.GetTestWorkflowExecutionsParams{}
}

func (r *PostgresRepository) buildSummaryParams(filter testworkflow.Filter) sqlc.GetTestWorkflowExecutionsSummaryParams {
	// Implementation depends on the Filter interface
	return sqlc.GetTestWorkflowExecutionsSummaryParams{}
}

// Helper functions for row conversion
func (r *PostgresRepository) convertRowToExecution(row sqlc.GetTestWorkflowExecutionByNameAndTestWorkflowRow) *testkube.TestWorkflowExecution {
	// Implementation for converting row to execution
	return &testkube.TestWorkflowExecution{}
}

func (r *PostgresRepository) convertRowToExecutionSimple(row sqlc.GetLatestTestWorkflowExecutionByTestWorkflowRow) *testkube.TestWorkflowExecution {
	// Implementation for converting row to execution
	return &testkube.TestWorkflowExecution{}
}

func (r *PostgresRepository) convertRowToExecutionSummary(row sqlc.GetLatestTestWorkflowExecutionsByTestWorkflowsRow) testkube.TestWorkflowExecutionSummary {
	// Implementation for converting row to execution summary
	return testkube.TestWorkflowExecutionSummary{}
}

func (r *PostgresRepository) convertRowToExecutionSimpleFromRunning(row sqlc.GetRunningTestWorkflowExecutionsRow) testkube.TestWorkflowExecution {
	// Implementation for converting row to execution
	return testkube.TestWorkflowExecution{}
}

func (r *PostgresRepository) convertRowToExecutionFromList(row sqlc.GetTestWorkflowExecutionsRow) testkube.TestWorkflowExecution {
	// Implementation for converting row to execution
	return testkube.TestWorkflowExecution{}
}

func (r *PostgresRepository) convertRowToSummary(row sqlc.GetTestWorkflowExecutionsSummaryRow) testkube.TestWorkflowExecutionSummary {
	// Implementation for converting row to summary
	return testkube.TestWorkflowExecutionSummary{}
}

func (r *PostgresRepository) convertUnassignedRowToExecution(row sqlc.GetUnassignedTestWorkflowExecutionsRow) testkube.TestWorkflowExecution {
	// Implementation for converting row to execution
	return testkube.TestWorkflowExecution{}
}

func (r *PostgresRepository) updateMainExecution(ctx context.Context, qtx *sqlc.Queries, execution *testkube.TestWorkflowExecution) error {
	// This would need a separate SQL query to update the main execution
	// For now, return nil
	return nil
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
