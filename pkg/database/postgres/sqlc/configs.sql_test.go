// config_queries_test.go
package sqlc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLCConfigQueries_GetConfig tests the GetConfig query syntax
func TestSQLCConfigQueries_GetConfig(t *testing.T) {
	// Create mock database connection
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Define expected query pattern
	expectedQuery := `SELECT id, cluster_id, enable_telemetry, created_at, updated_at 
FROM configs 
WHERE id = \$1`

	// Mock expected result
	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	}).AddRow(
		"api", "cluster123", true, time.Now(), time.Now(),
	)

	mock.ExpectQuery(expectedQuery).WithArgs("api").WillReturnRows(rows)

	// Execute query
	result, err := queries.GetConfig(ctx, "api")

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "api", result.ID)
	assert.Equal(t, "cluster123", result.ClusterID)
	assert.True(t, result.EnableTelemetry.Bool)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_GetConfigByFixedId(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `SELECT id, cluster_id, enable_telemetry, created_at, updated_at 
FROM configs 
WHERE id = 'api'`

	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	}).AddRow(
		"api", "cluster456", false, time.Now(), time.Now(),
	)

	mock.ExpectQuery(expectedQuery).WillReturnRows(rows)

	result, err := queries.GetConfigByFixedId(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "api", result.ID)
	assert.Equal(t, "cluster456", result.ClusterID)
	assert.False(t, result.EnableTelemetry.Bool)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_UpsertConfig(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `INSERT INTO configs \(id, cluster_id, enable_telemetry\)
VALUES \(\$1, \$2, \$3\)
ON CONFLICT \(id\) DO UPDATE SET
    cluster_id = EXCLUDED\.cluster_id,
    enable_telemetry = EXCLUDED\.enable_telemetry,
    updated_at = NOW\(\)
RETURNING id, cluster_id, enable_telemetry, created_at, updated_at`

	params := UpsertConfigParams{
		ID:              "api",
		ClusterID:       "cluster789",
		EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
	}

	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	}).AddRow(
		"api", "cluster789", true, time.Now(), time.Now(),
	)

	mock.ExpectQuery(expectedQuery).WithArgs(
		params.ID, params.ClusterID, params.EnableTelemetry,
	).WillReturnRows(rows)

	result, err := queries.UpsertConfig(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, "api", result.ID)
	assert.Equal(t, "cluster789", result.ClusterID)
	assert.True(t, result.EnableTelemetry.Bool)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_UpdateClusterId(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `UPDATE configs 
SET cluster_id = \$1, updated_at = NOW\(\)
WHERE id = \$2`

	params := UpdateClusterIdParams{
		ClusterID: "new-cluster-id",
		ID:        "api",
	}

	mock.ExpectExec(expectedQuery).WithArgs(
		params.ClusterID, params.ID,
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = queries.UpdateClusterId(ctx, params)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_UpdateTelemetryEnabled(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `UPDATE configs 
SET enable_telemetry = \$1, updated_at = NOW\(\)
WHERE id = \$2`

	params := UpdateTelemetryEnabledParams{
		EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
		ID:              "api",
	}

	mock.ExpectExec(expectedQuery).WithArgs(
		params.EnableTelemetry, params.ID,
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = queries.UpdateTelemetryEnabled(ctx, params)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_GetClusterId(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `SELECT cluster_id FROM configs WHERE id = \$1`

	rows := mock.NewRows([]string{"cluster_id"}).AddRow("cluster999")

	mock.ExpectQuery(expectedQuery).WithArgs("api").WillReturnRows(rows)

	result, err := queries.GetClusterId(ctx, "api")

	assert.NoError(t, err)
	assert.Equal(t, "cluster999", result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_GetTelemetryEnabled(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `SELECT enable_telemetry FROM configs WHERE id = \$1`

	rows := mock.NewRows([]string{"enable_telemetry"}).AddRow(true)

	mock.ExpectQuery(expectedQuery).WithArgs("api").WillReturnRows(rows)

	result, err := queries.GetTelemetryEnabled(ctx, "api")

	assert.NoError(t, err)
	assert.True(t, result.Bool)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCConfigQueries_CreateConfigIfNotExists(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `INSERT INTO configs \(id, cluster_id, enable_telemetry\)
VALUES \(\$1, \$2, \$3\)
ON CONFLICT \(id\) DO NOTHING`

	params := CreateConfigIfNotExistsParams{
		ID:              "api",
		ClusterID:       "new-cluster",
		EnableTelemetry: pgtype.Bool{Bool: false, Valid: true},
	}

	mock.ExpectExec(expectedQuery).WithArgs(
		params.ID, params.ClusterID, params.EnableTelemetry,
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = queries.CreateConfigIfNotExists(ctx, params)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test with empty cluster ID
func TestSQLCConfigQueries_UpsertConfig_EmptyClusterId(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `INSERT INTO configs \(id, cluster_id, enable_telemetry\)
VALUES \(\$1, \$2, \$3\)
ON CONFLICT \(id\) DO UPDATE SET
    cluster_id = EXCLUDED\.cluster_id,
    enable_telemetry = EXCLUDED\.enable_telemetry,
    updated_at = NOW\(\)
RETURNING id, cluster_id, enable_telemetry, created_at, updated_at`

	params := UpsertConfigParams{
		ID:              "api",
		ClusterID:       "",
		EnableTelemetry: pgtype.Bool{Bool: false, Valid: true},
	}

	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	}).AddRow(
		"api", "", false, time.Now(), time.Now(),
	)

	mock.ExpectQuery(expectedQuery).WithArgs(
		params.ID, params.ClusterID, params.EnableTelemetry,
	).WillReturnRows(rows)

	result, err := queries.UpsertConfig(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, "api", result.ID)
	assert.Equal(t, "", result.ClusterID)
	assert.False(t, result.EnableTelemetry.Bool)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test query with different ID
func TestSQLCConfigQueries_GetConfig_DifferentId(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `SELECT id, cluster_id, enable_telemetry, created_at, updated_at 
FROM configs 
WHERE id = \$1`

	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	}).AddRow(
		"test-config", "cluster-test", false, time.Now(), time.Now(),
	)

	mock.ExpectQuery(expectedQuery).WithArgs("test-config").WillReturnRows(rows)

	result, err := queries.GetConfig(ctx, "test-config")

	assert.NoError(t, err)
	assert.Equal(t, "test-config", result.ID)
	assert.Equal(t, "cluster-test", result.ClusterID)
	assert.False(t, result.EnableTelemetry.Bool)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test multiple configurations scenario
func TestSQLCConfigQueries_UpsertConfig_MultipleConfigs(t *testing.T) {
	testCases := []struct {
		name           string
		params         UpsertConfigParams
		expectedResult Config
	}{
		{
			name: "Config with telemetry enabled",
			params: UpsertConfigParams{
				ID:              "api",
				ClusterID:       "cluster-enabled",
				EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
			},
			expectedResult: Config{
				ID:              "api",
				ClusterID:       "cluster-enabled",
				EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
			},
		},
		{
			name: "Config with telemetry disabled",
			params: UpsertConfigParams{
				ID:              "api",
				ClusterID:       "cluster-disabled",
				EnableTelemetry: pgtype.Bool{Bool: false, Valid: true},
			},
			expectedResult: Config{
				ID:              "api",
				ClusterID:       "cluster-disabled",
				EnableTelemetry: pgtype.Bool{Bool: false, Valid: true},
			},
		},
		{
			name: "Config with long cluster ID",
			params: UpsertConfigParams{
				ID:              "api",
				ClusterID:       "cluster-very-long-identifier-with-many-characters-12345",
				EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
			},
			expectedResult: Config{
				ID:              "api",
				ClusterID:       "cluster-very-long-identifier-with-many-characters-12345",
				EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			queries := New(mock)
			ctx := context.Background()

			expectedQuery := `INSERT INTO configs \(id, cluster_id, enable_telemetry\)
VALUES \(\$1, \$2, \$3\)
ON CONFLICT \(id\) DO UPDATE SET
    cluster_id = EXCLUDED\.cluster_id,
    enable_telemetry = EXCLUDED\.enable_telemetry,
    updated_at = NOW\(\)
RETURNING id, cluster_id, enable_telemetry, created_at, updated_at`

			rows := mock.NewRows([]string{
				"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
			}).AddRow(
				tc.expectedResult.ID,
				tc.expectedResult.ClusterID,
				tc.expectedResult.EnableTelemetry,
				time.Now(),
				time.Now(),
			)

			mock.ExpectQuery(expectedQuery).WithArgs(
				tc.params.ID, tc.params.ClusterID, tc.params.EnableTelemetry,
			).WillReturnRows(rows)

			result, err := queries.UpsertConfig(ctx, tc.params)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResult.ID, result.ID)
			assert.Equal(t, tc.expectedResult.ClusterID, result.ClusterID)
			assert.Equal(t, tc.expectedResult.EnableTelemetry, result.EnableTelemetry)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Test error scenarios
func TestSQLCConfigQueries_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(pgxmock.PgxPoolIface)
		executeQuery  func(*Queries, context.Context) error
		expectedError bool
	}{
		{
			name: "GetConfig - Record not found",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, cluster_id, enable_telemetry, created_at, updated_at FROM configs WHERE id = \$1`).
					WithArgs("nonexistent").
					WillReturnError(errors.New(""))
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.GetConfig(ctx, "nonexistent")
				return err
			},
			expectedError: true,
		},
		{
			name: "UpdateClusterId - No rows affected",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`UPDATE configs SET cluster_id = \$1, updated_at = NOW\(\) WHERE id = \$2`).
					WithArgs("new-cluster", "nonexistent").
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				return q.UpdateClusterId(ctx, UpdateClusterIdParams{
					ClusterID: "new-cluster",
					ID:        "nonexistent",
				})
			},
			expectedError: false, // UPDATE with 0 rows is not an error in PostgreSQL
		},
		{
			name: "UpsertConfig - Database constraint violation",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`INSERT INTO configs`).
					WithArgs("api", "cluster", pgtype.Bool{Bool: true, Valid: true}).
					WillReturnError(errors.New(""))
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.UpsertConfig(ctx, UpsertConfigParams{
					ID:              "api",
					ClusterID:       "cluster",
					EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
				})
				return err
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			queries := New(mock)
			ctx := context.Background()

			tt.setupMock(mock)

			err = tt.executeQuery(queries, ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Test query parameter validation
func TestSQLCQueries_ParameterValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(pgxmock.PgxPoolIface)
		executeQuery  func(*Queries, context.Context) error
		expectedError bool
	}{
		{
			name: "UpsertConfig with empty ID should work",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := mock.NewRows([]string{
					"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
				}).AddRow("", "cluster", false, time.Now(), time.Now())

				mock.ExpectQuery(`INSERT INTO configs`).
					WithArgs("", "cluster", pgtype.Bool{Bool: false, Valid: true}).
					WillReturnRows(rows)
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.UpsertConfig(ctx, UpsertConfigParams{
					ID:              "",
					ClusterID:       "cluster",
					EnableTelemetry: pgtype.Bool{Bool: false, Valid: true},
				})
				return err
			},
			expectedError: false,
		},
		{
			name: "UpdateClusterId with empty cluster ID should work",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`UPDATE configs SET cluster_id = \$1, updated_at = NOW\(\) WHERE id = \$2`).
					WithArgs("", "api").
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				return q.UpdateClusterId(ctx, UpdateClusterIdParams{
					ClusterID: "",
					ID:        "api",
				})
			},
			expectedError: false,
		},
		{
			name: "GetTelemetryEnabled with special characters in ID",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := mock.NewRows([]string{"enable_telemetry"}).AddRow(true)
				mock.ExpectQuery(`SELECT enable_telemetry FROM configs WHERE id = \$1`).
					WithArgs("api-test_config.v1").
					WillReturnRows(rows)
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.GetTelemetryEnabled(ctx, "api-test_config.v1")
				return err
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			queries := New(mock)
			ctx := context.Background()

			tt.setupMock(mock)

			err = tt.executeQuery(queries, ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Benchmark tests to validate query performance characteristics
func BenchmarkSQLCQueries_GetConfig(b *testing.B) {
	mock, err := pgxmock.NewPool()
	require.NoError(b, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Setup mock expectations
	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	})

	for i := 0; i < b.N; i++ {
		rows.AddRow("api", "cluster123", true, time.Now(), time.Now())
		mock.ExpectQuery(`SELECT`).WithArgs("api").WillReturnRows(rows)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queries.GetConfig(ctx, "api")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLCQueries_UpsertConfig(b *testing.B) {
	mock, err := pgxmock.NewPool()
	require.NoError(b, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Setup mock expectations
	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	})

	for i := 0; i < b.N; i++ {
		rows.AddRow("api", "cluster123", true, time.Now(), time.Now())
		mock.ExpectQuery(`INSERT INTO configs`).
			WithArgs("api", "cluster123", pgtype.Bool{Bool: true, Valid: true}).
			WillReturnRows(rows)
	}

	params := UpsertConfigParams{
		ID:              "api",
		ClusterID:       "cluster123",
		EnableTelemetry: pgtype.Bool{Bool: true, Valid: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queries.UpsertConfig(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test query result mapping
func TestSQLCQueries_ResultMapping(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Test with various data types and edge cases
	testTime := time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC)

	expectedQuery := `SELECT id, cluster_id, enable_telemetry, created_at, updated_at 
FROM configs 
WHERE id = \$1`

	rows := mock.NewRows([]string{
		"id", "cluster_id", "enable_telemetry", "created_at", "updated_at",
	}).AddRow(
		"api", "cluster-with-special-chars_123", true, testTime, testTime,
	)

	mock.ExpectQuery(expectedQuery).WithArgs("api").WillReturnRows(rows)

	result, err := queries.GetConfig(ctx, "api")

	assert.NoError(t, err)
	assert.Equal(t, "api", result.ID)
	assert.Equal(t, "cluster-with-special-chars_123", result.ClusterID)
	assert.True(t, result.EnableTelemetry.Bool)
	assert.Equal(t, testTime, result.CreatedAt.Time)
	assert.Equal(t, testTime, result.UpdatedAt.Time)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test concurrent query execution simulation
func TestSQLCQueries_ConcurrentExecution(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Simulate multiple concurrent reads
	expectedQuery := `SELECT enable_telemetry FROM configs WHERE id = \$1`

	for i := 0; i < 5; i++ {
		rows := mock.NewRows([]string{"enable_telemetry"}).AddRow(true)
		mock.ExpectQuery(expectedQuery).WithArgs("api").WillReturnRows(rows)
	}

	// Execute multiple queries
	for i := 0; i < 5; i++ {
		result, err := queries.GetTelemetryEnabled(ctx, "api")
		assert.NoError(t, err)
		assert.True(t, result.Bool)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}
