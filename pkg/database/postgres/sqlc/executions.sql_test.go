package sqlc

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLCTestWorkflowExecutionQueries_GetTestWorkflowExecutionsTotals(t *testing.T) {
	// Create mock database connection
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Mock expected result
	rows := mock.NewRows([]string{"status", "count"}).AddRow("passed", 10)

	// Expect any query - we just want to verify the function can be called
	mock.ExpectQuery("SELECT").WithArgs(
		"org-id",
		"env-id",
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
		[][]string{},         // tag_conditions
		[]string{},           // label_keys
		[][]string{},         // label_conditions
		[]string{},           // selector_keys
		[][]string{},         // selector_conditions
	).WillReturnRows(rows)

	// Execute query
	result, err := queries.GetTestWorkflowExecutionsTotals(ctx, GetTestWorkflowExecutionsTotalsParams{
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
		TagConditions:      [][]string{},
		LabelKeys:          []string{},
		LabelConditions:    [][]string{},
		SelectorKeys:       []string{},
		SelectorConditions: [][]string{},
	})

	// Assertions
	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "passed", result[0].Status.String)
	assert.Equal(t, int64(10), result[0].Count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
