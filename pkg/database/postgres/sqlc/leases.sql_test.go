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

// TestSQLCLeaseQueries_FindLeaseById tests the FindLeaseById query syntax
func TestSQLCLeaseQueries_FindLeaseById(t *testing.T) {
	// Create mock database connection
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Define expected query pattern
	expectedQuery := `SELECT id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at
FROM leases 
WHERE id = \$1`

	// Mock expected result
	testTime := time.Now()
	rows := mock.NewRows([]string{
		"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
	}).AddRow(
		"lease-cluster123", "test-identifier", "cluster123", testTime, testTime, testTime, testTime,
	)

	mock.ExpectQuery(expectedQuery).WithArgs("lease-cluster123").WillReturnRows(rows)

	// Execute query
	result, err := queries.FindLeaseById(ctx, "lease-cluster123")

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "lease-cluster123", result.ID)
	assert.Equal(t, "test-identifier", result.Identifier)
	assert.Equal(t, "cluster123", result.ClusterID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCLeaseQueries_InsertLease(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `INSERT INTO leases \(id, identifier, cluster_id, acquired_at, renewed_at\)
VALUES \(\$1, \$2, \$3, \$4, \$5\)
RETURNING id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at`

	testTime := time.Now()
	params := InsertLeaseParams{
		ID:         "lease-test-cluster",
		Identifier: "test-identifier",
		ClusterID:  "test-cluster",
		AcquiredAt: pgtype.Timestamptz{Time: testTime, Valid: true},
		RenewedAt:  pgtype.Timestamptz{Time: testTime, Valid: true},
	}

	rows := mock.NewRows([]string{
		"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
	}).AddRow(
		"lease-test-cluster", "test-identifier", "test-cluster", testTime, testTime, testTime, testTime,
	)

	mock.ExpectQuery(expectedQuery).WithArgs(
		params.ID, params.Identifier, params.ClusterID, params.AcquiredAt, params.RenewedAt,
	).WillReturnRows(rows)

	result, err := queries.InsertLease(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, "lease-test-cluster", result.ID)
	assert.Equal(t, "test-identifier", result.Identifier)
	assert.Equal(t, "test-cluster", result.ClusterID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLCLeaseQueries_UpdateLease(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	expectedQuery := `UPDATE leases 
SET 
    identifier = \$1,
    cluster_id = \$2,
    acquired_at = \$3,
    renewed_at = \$4,
    updated_at = NOW\(\)
WHERE id = \$5
RETURNING id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at`

	testTime := time.Now()
	params := UpdateLeaseParams{
		Identifier: "updated-identifier",
		ClusterID:  "updated-cluster",
		AcquiredAt: pgtype.Timestamptz{Time: testTime, Valid: true},
		RenewedAt:  pgtype.Timestamptz{Time: testTime, Valid: true},
		ID:         "lease-test-cluster",
	}

	rows := mock.NewRows([]string{
		"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
	}).AddRow(
		"lease-test-cluster", "updated-identifier", "updated-cluster", testTime, testTime, testTime, testTime,
	)

	mock.ExpectQuery(expectedQuery).WithArgs(
		params.Identifier, params.ClusterID, params.AcquiredAt, params.RenewedAt, params.ID,
	).WillReturnRows(rows)

	result, err := queries.UpdateLease(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, "lease-test-cluster", result.ID)
	assert.Equal(t, "updated-identifier", result.Identifier)
	assert.Equal(t, "updated-cluster", result.ClusterID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test with various edge cases and data types
func TestSQLCLeaseQueries_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(pgxmock.PgxPoolIface)
		executeQuery  func(*Queries, context.Context) error
		expectedError bool
	}{
		{
			name: "FindLeaseById - Record not found",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at FROM leases WHERE id = \$1`).
					WithArgs("nonexistent").
					WillReturnError(errors.New(""))
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.FindLeaseById(ctx, "nonexistent")
				return err
			},
			expectedError: true,
		},
		{
			name: "InsertLease - Constraint violation",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`INSERT INTO leases`).
					WithArgs("duplicate-id", "identifier", "cluster", pgtype.Timestamptz{}, pgtype.Timestamptz{}).
					WillReturnError(errors.New(""))
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.InsertLease(ctx, InsertLeaseParams{
					ID:         "duplicate-id",
					Identifier: "identifier",
					ClusterID:  "cluster",
					AcquiredAt: pgtype.Timestamptz{},
					RenewedAt:  pgtype.Timestamptz{},
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

// Test query parameter validation with various data types
func TestSQLCLeaseQueries_ParameterValidation(t *testing.T) {
	testTime := time.Now()
	tests := []struct {
		name          string
		setupMock     func(pgxmock.PgxPoolIface)
		executeQuery  func(*Queries, context.Context) error
		expectedError bool
	}{
		{
			name: "InsertLease with empty strings should work",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := mock.NewRows([]string{
					"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
				}).AddRow("", "", "", testTime, testTime, testTime, testTime)

				mock.ExpectQuery(`INSERT INTO leases`).
					WithArgs("", "", "", pgtype.Timestamptz{Time: testTime, Valid: true}, pgtype.Timestamptz{Time: testTime, Valid: true}).
					WillReturnRows(rows)
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				_, err := q.InsertLease(ctx, InsertLeaseParams{
					ID:         "",
					Identifier: "",
					ClusterID:  "",
					AcquiredAt: pgtype.Timestamptz{Time: testTime, Valid: true},
					RenewedAt:  pgtype.Timestamptz{Time: testTime, Valid: true},
				})
				return err
			},
			expectedError: false,
		},
		{
			name: "UpdateLease with very long strings",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				longString := string(make([]byte, 1000)) // Very long string
				rows := mock.NewRows([]string{
					"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
				}).AddRow("lease-id", longString, longString, testTime, testTime, testTime, testTime)

				mock.ExpectQuery(`UPDATE leases SET`).
					WithArgs(longString, longString, pgtype.Timestamptz{Time: testTime, Valid: true}, pgtype.Timestamptz{Time: testTime, Valid: true}, "lease-id").
					WillReturnRows(rows)
			},
			executeQuery: func(q *Queries, ctx context.Context) error {
				longString := string(make([]byte, 1000))
				_, err := q.UpdateLease(ctx, UpdateLeaseParams{
					Identifier: longString,
					ClusterID:  longString,
					AcquiredAt: pgtype.Timestamptz{Time: testTime, Valid: true},
					RenewedAt:  pgtype.Timestamptz{Time: testTime, Valid: true},
					ID:         "lease-id",
				})
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
func BenchmarkSQLCLeaseQueries_FindLeaseById(b *testing.B) {
	mock, err := pgxmock.NewPool()
	require.NoError(b, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Setup mock expectations
	testTime := time.Now()
	rows := mock.NewRows([]string{
		"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
	})

	for i := 0; i < b.N; i++ {
		rows.AddRow("lease-test", "identifier", "cluster", testTime, testTime, testTime, testTime)
		mock.ExpectQuery(`SELECT`).WithArgs("lease-test").WillReturnRows(rows)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queries.FindLeaseById(ctx, "lease-test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test query result mapping with various timestamp scenarios
func TestSQLCLeaseQueries_TimestampHandling(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	queries := New(mock)
	ctx := context.Background()

	// Test with various timestamp scenarios
	testCases := []struct {
		name       string
		acquiredAt time.Time
		renewedAt  time.Time
		createdAt  time.Time
		updatedAt  time.Time
	}{
		{
			name:       "Current timestamps",
			acquiredAt: time.Now(),
			renewedAt:  time.Now(),
			createdAt:  time.Now(),
			updatedAt:  time.Now(),
		},
		{
			name:       "Past timestamps",
			acquiredAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			renewedAt:  time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
			createdAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			updatedAt:  time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "Future timestamps",
			acquiredAt: time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC),
			renewedAt:  time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC),
			createdAt:  time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC),
			updatedAt:  time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedQuery := `SELECT id, identifier, cluster_id, acquired_at, renewed_at, created_at, updated_at FROM leases WHERE id = \$1`

			rows := mock.NewRows([]string{
				"id", "identifier", "cluster_id", "acquired_at", "renewed_at", "created_at", "updated_at",
			}).AddRow(
				"lease-timestamp-test", "test-identifier", "test-cluster",
				tc.acquiredAt, tc.renewedAt, tc.createdAt, tc.updatedAt,
			)

			mock.ExpectQuery(expectedQuery).WithArgs("lease-timestamp-test").WillReturnRows(rows)

			result, err := queries.FindLeaseById(ctx, "lease-timestamp-test")

			assert.NoError(t, err)
			assert.Equal(t, "lease-timestamp-test", result.ID)
			assert.Equal(t, "test-identifier", result.Identifier)
			assert.Equal(t, "test-cluster", result.ClusterID)
			assert.True(t, tc.acquiredAt.Equal(result.AcquiredAt.Time))
			assert.True(t, tc.renewedAt.Equal(result.RenewedAt.Time))
			assert.True(t, tc.createdAt.Equal(result.CreatedAt.Time))
			assert.True(t, tc.updatedAt.Equal(result.UpdatedAt.Time))
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}
