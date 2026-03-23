package sqlc

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commonExecutionColumns are the column names returned by the main execution listing queries.
var commonExecutionColumns = []string{
	"id", "group_id", "runner_id", "runner_target", "runner_original_target", "name", "namespace", "number",
	"scheduled_at", "assigned_at", "status_at", "test_workflow_execution_name", "disable_webhooks",
	"tags", "running_context", "config_params", "runtime", "created_at", "updated_at",
	"status", "predicted_status", "queued_at", "started_at", "finished_at",
	"duration", "total_duration", "duration_ms", "paused_ms", "total_duration_ms",
	"pauses", "initialization", "steps",
	"workflow_name", "workflow_namespace", "workflow_description", "workflow_labels", "workflow_annotations",
	"workflow_created", "workflow_updated", "workflow_spec", "workflow_read_only", "workflow_status",
	"resolved_workflow_name", "resolved_workflow_namespace", "resolved_workflow_description",
	"resolved_workflow_labels", "resolved_workflow_annotations", "resolved_workflow_created",
	"resolved_workflow_updated", "resolved_workflow_spec", "resolved_workflow_read_only", "resolved_workflow_status",
	"signatures_json", "outputs_json", "reports_json", "resource_aggregations_global", "resource_aggregations_step",
}

// sampleExecutionRow returns a minimal set of values for commonExecutionColumns.
func sampleExecutionRow() []interface{} {
	now := time.Now()
	return []interface{}{
		"test-id", "group-1", "runner-1", []byte(`{}`), []byte(`{}`),
		"test-execution", "default", int64(1),
		now, now, now, "test-execution-name", false,
		[]byte(`{"env":"test"}`), []byte(`{}`), []byte(`{}`), []byte(`{}`), now, now,
		"passed", "passed", now, now, now,
		"5m", "5m", int64(300000), int64(0), int64(300000),
		[]byte(`[]`), []byte(`{}`), []byte(`{}`),
		"test-workflow", "default", "Test workflow", []byte(`{}`), []byte(`{}`),
		now, now, []byte(`{}`), false, []byte(`{}`),
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		[]byte(`[]`), []byte(`[]`), []byte(`[]`), []byte(`{}`), []byte(`{}`),
	}
}

// defaultListingParams returns a GetTestWorkflowExecutionsTotalsParams with all-empty/zero values
// and the given org/env IDs.
func defaultTotalsParams(orgID, envID string) GetTestWorkflowExecutionsTotalsParams {
	return GetTestWorkflowExecutionsTotalsParams{
		OrganizationID:     orgID,
		EnvironmentID:      envID,
		WorkflowName:       "",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{},
		TagConditions:      []string{},
		LabelKeys:          []string{},
		LabelConditions:    []string{},
		SelectorKeys:       []string{},
		SelectorConditions: []string{},
	}
}

func defaultListingArgs(orgID, envID string) []interface{} {
	return []interface{}{
		orgID,
		envID,
		"",                   // workflow_name
		[]string{},           // workflow_names
		"",                   // text_search
		pgtype.Timestamptz{}, // start_date
		pgtype.Timestamptz{}, // end_date
		int32(0),             // last_n_days
		[]string{},           // statuses
		"",                   // runner_id
		pgtype.Bool{},        // assigned
		"",                   // actor_name
		"",                   // actor_type
		"",                   // group_id
		pgtype.Bool{},        // initialized
		[]byte("[]"),         // health_ranges
		[]string{},           // tag_keys
		[]string{},           // tag_conditions
		[]string{},           // label_keys
		[]string{},           // label_conditions
		[]string{},           // selector_keys
		[]string{},           // selector_conditions
	}
}

func TestSQLCTestWorkflowExecutionQueries_GetTestWorkflowExecutionsTotals(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows([]string{"status", "count"}).
		AddRow("passed", int64(5)).
		AddRow("failed", int64(2))

	params := defaultTotalsParams("org-id", "env-id")
	mock.ExpectQuery(regexp.QuoteMeta(getTestWorkflowExecutionsTotals)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
	).WillReturnRows(rows)

	result, err := queries.GetTestWorkflowExecutionsTotals(ctx, params)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "passed", result[0].Status.String)
	assert.Equal(t, int64(5), result[0].Count)
	assert.Equal(t, "failed", result[1].Status.String)
	assert.Equal(t, int64(2), result[1].Count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCTestWorkflowExecutionQueries_GetTestWorkflowExecutionsTotals_WithFilters(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows([]string{"status", "count"}).AddRow("passed", int64(3))

	params := GetTestWorkflowExecutionsTotalsParams{
		OrganizationID:     "org-id",
		EnvironmentID:      "env-id",
		WorkflowName:       "my-workflow",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{"passed"},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{"environment"},
		TagConditions:      []string{"team=backend"},
		LabelKeys:          []string{"app:not_exists"},
		LabelConditions:    []string{"version=v1"},
		SelectorKeys:       []string{"region"},
		SelectorConditions: []string{"env=prod"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(getTestWorkflowExecutionsTotals)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
	).WillReturnRows(rows)

	result, err := queries.GetTestWorkflowExecutionsTotals(ctx, params)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "passed", result[0].Status.String)
	assert.Equal(t, int64(3), result[0].Count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCTestWorkflowExecutionQueries_CountTestWorkflowExecutions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows([]string{"count"}).AddRow(int64(42))

	params := CountTestWorkflowExecutionsParams{
		OrganizationID:     "org-id",
		EnvironmentID:      "env-id",
		WorkflowName:       "",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{},
		TagConditions:      []string{},
		LabelKeys:          []string{},
		LabelConditions:    []string{},
		SelectorKeys:       []string{},
		SelectorConditions: []string{},
	}

	mock.ExpectQuery(regexp.QuoteMeta(countTestWorkflowExecutions)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
	).WillReturnRows(rows)

	count, err := queries.CountTestWorkflowExecutions(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, int64(42), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCTestWorkflowExecutionQueries_CountTestWorkflowExecutions_WithTextArrayFilters(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows([]string{"count"}).AddRow(int64(7))

	params := CountTestWorkflowExecutionsParams{
		OrganizationID:     "org-id",
		EnvironmentID:      "env-id",
		WorkflowName:       "",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{"passed", "failed"},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{"environment", "app:not_exists"},
		TagConditions:      []string{"team=backend", "team=frontend"},
		LabelKeys:          []string{"region"},
		LabelConditions:    []string{"version=v1"},
		SelectorKeys:       []string{"tier"},
		SelectorConditions: []string{"env=staging"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(countTestWorkflowExecutions)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
	).WillReturnRows(rows)

	count, err := queries.CountTestWorkflowExecutions(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, int64(7), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCTestWorkflowExecutionQueries_GetFinishedTestWorkflowExecutions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows(commonExecutionColumns).AddRow(sampleExecutionRow()...)

	params := GetFinishedTestWorkflowExecutionsParams{
		OrganizationID:     "org-id",
		EnvironmentID:      "env-id",
		WorkflowName:       "",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{},
		TagConditions:      []string{},
		LabelKeys:          []string{},
		LabelConditions:    []string{},
		SelectorKeys:       []string{},
		SelectorConditions: []string{},
		Fst:                0,
		Lmt:                int32(100),
	}

	mock.ExpectQuery(regexp.QuoteMeta(getFinishedTestWorkflowExecutions)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
		params.Fst, params.Lmt,
	).WillReturnRows(rows)

	result, err := queries.GetFinishedTestWorkflowExecutions(ctx, params)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "test-id", result[0].ID)
	assert.Equal(t, "test-execution", result[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCTestWorkflowExecutionQueries_GetTestWorkflowExecutions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows(commonExecutionColumns).AddRow(sampleExecutionRow()...)

	params := GetTestWorkflowExecutionsParams{
		OrganizationID:     "org-id",
		EnvironmentID:      "env-id",
		WorkflowName:       "",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{},
		TagConditions:      []string{},
		LabelKeys:          []string{},
		LabelConditions:    []string{},
		SelectorKeys:       []string{},
		SelectorConditions: []string{},
		Fst:                0,
		Lmt:                int32(100),
	}

	mock.ExpectQuery(regexp.QuoteMeta(getTestWorkflowExecutions)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
		params.Fst, params.Lmt,
	).WillReturnRows(rows)

	result, err := queries.GetTestWorkflowExecutions(ctx, params)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "test-id", result[0].ID)
	assert.Equal(t, "test-execution", result[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCTestWorkflowExecutionQueries_GetTestWorkflowExecutionsSummary(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	rows := mock.NewRows(commonExecutionColumns).AddRow(sampleExecutionRow()...)

	params := GetTestWorkflowExecutionsSummaryParams{
		OrganizationID:     "org-id",
		EnvironmentID:      "env-id",
		WorkflowName:       "",
		WorkflowNames:      []string{},
		TextSearch:         "",
		StartDate:          pgtype.Timestamptz{},
		EndDate:            pgtype.Timestamptz{},
		LastNDays:          0,
		Statuses:           []string{},
		RunnerID:           "",
		Assigned:           pgtype.Bool{},
		ActorName:          "",
		ActorType:          "",
		GroupID:            "",
		Initialized:        pgtype.Bool{},
		HealthRanges:       []byte("[]"),
		TagKeys:            []string{},
		TagConditions:      []string{},
		LabelKeys:          []string{},
		LabelConditions:    []string{},
		SelectorKeys:       []string{},
		SelectorConditions: []string{},
		Fst:                0,
		Lmt:                int32(100),
	}

	mock.ExpectQuery(regexp.QuoteMeta(getTestWorkflowExecutionsSummary)).WithArgs(
		params.OrganizationID, params.EnvironmentID,
		params.WorkflowName, params.WorkflowNames, params.TextSearch,
		params.StartDate, params.EndDate, params.LastNDays,
		params.Statuses, params.RunnerID, params.Assigned,
		params.ActorName, params.ActorType, params.GroupID,
		params.Initialized, params.HealthRanges,
		params.TagKeys, params.TagConditions,
		params.LabelKeys, params.LabelConditions,
		params.SelectorKeys, params.SelectorConditions,
		params.Fst, params.Lmt,
	).WillReturnRows(rows)

	result, err := queries.GetTestWorkflowExecutionsSummary(ctx, params)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "test-id", result[0].ID)
	assert.Equal(t, "test-execution", result[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}
