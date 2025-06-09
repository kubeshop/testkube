package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
)

// MockQueriesInterface implementation
type MockQueriesInterface struct {
	mock.Mock
}

func (m *MockQueriesInterface) FindLeaseById(ctx context.Context, leaseID string) (sqlc.Lease, error) {
	args := m.Called(ctx, leaseID)
	return args.Get(0).(sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) InsertLease(ctx context.Context, arg sqlc.InsertLeaseParams) (sqlc.Lease, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) UpdateLease(ctx context.Context, arg sqlc.UpdateLeaseParams) (sqlc.Lease, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) UpsertLease(ctx context.Context, arg sqlc.UpsertLeaseParams) (sqlc.Lease, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) GetLeaseByClusterId(ctx context.Context, clusterID string) (sqlc.Lease, error) {
	args := m.Called(ctx, clusterID)
	return args.Get(0).(sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) GetLeasesByIdentifier(ctx context.Context, identifier string) ([]sqlc.Lease, error) {
	args := m.Called(ctx, identifier)
	return args.Get(0).([]sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) GetExpiredLeases(ctx context.Context, expirationTime pgtype.Timestamptz) ([]sqlc.Lease, error) {
	args := m.Called(ctx, expirationTime)
	return args.Get(0).([]sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) DeleteLease(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueriesInterface) DeleteLeasesByClusterId(ctx context.Context, clusterID string) error {
	args := m.Called(ctx, clusterID)
	return args.Error(0)
}

func (m *MockQueriesInterface) DeleteExpiredLeases(ctx context.Context, expirationTime pgtype.Timestamptz) error {
	args := m.Called(ctx, expirationTime)
	return args.Error(0)
}

func (m *MockQueriesInterface) GetAllLeases(ctx context.Context) ([]sqlc.Lease, error) {
	args := m.Called(ctx)
	return args.Get(0).([]sqlc.Lease), args.Error(1)
}

func (m *MockQueriesInterface) CountLeasesByClusterId(ctx context.Context, clusterID string) (int64, error) {
	args := m.Called(ctx, clusterID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueriesInterface) GetLatestLeaseByIdentifierAndCluster(ctx context.Context, arg sqlc.GetLatestLeaseByIdentifierAndClusterParams) (sqlc.Lease, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.Lease), args.Error(1)
}

func createPostgresLeaseBackend(queries sqlc.LeaseBackendQueriesInterface, db sqlc.DatabaseInterface) *PostgresLeaseBackend {
	return &PostgresLeaseBackend{
		db:      db,
		queries: queries,
	}
}

func TestPostgresLeaseBackend_TryAcquire(t *testing.T) {
	t.Run("AcquireNewLease", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()
		id := "test-identifier"
		clusterID := "test-cluster"
		leaseID := "lease-test-cluster"

		// Mock lease not found, then successful insert
		mockQueries.On("FindLeaseById", ctx, leaseID).Return(sqlc.Lease{}, pgx.ErrNoRows)

		expectedLease := sqlc.Lease{
			ID:         leaseID,
			Identifier: id,
			ClusterID:  clusterID,
			AcquiredAt: toPgTimestamp(time.Now()),
			RenewedAt:  toPgTimestamp(time.Now()),
		}
		mockQueries.On("InsertLease", ctx, mock.AnythingOfType("sqlc.InsertLeaseParams")).Return(expectedLease, nil)

		// Act
		acquired, err := backend.TryAcquire(ctx, id, clusterID)

		// Assert
		assert.NoError(t, err)
		assert.True(t, acquired)
		mockQueries.AssertExpectations(t)
	})

	t.Run("AcquireExistingValidLease", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()
		id := "test-identifier"
		clusterID := "test-cluster"
		leaseID := "lease-test-cluster"

		// Mock existing valid lease
		existingLease := sqlc.Lease{
			ID:         leaseID,
			Identifier: id,
			ClusterID:  clusterID,
			AcquiredAt: toPgTimestamp(time.Now().Add(-time.Minute)),
			RenewedAt:  toPgTimestamp(time.Now()), // Recent renewal
		}
		mockQueries.On("FindLeaseById", ctx, leaseID).Return(existingLease, nil)

		// Act
		acquired, err := backend.TryAcquire(ctx, id, clusterID)

		// Assert
		assert.NoError(t, err)
		assert.True(t, acquired)
		mockQueries.AssertExpectations(t)
	})

	t.Run("AcquireExpiredLease", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()
		id := "test-identifier"
		clusterID := "test-cluster"
		leaseID := "lease-test-cluster"

		// Mock expired lease
		expiredLease := sqlc.Lease{
			ID:         leaseID,
			Identifier: "other-identifier",
			ClusterID:  clusterID,
			AcquiredAt: toPgTimestamp(time.Now().Add(-time.Hour)),
			RenewedAt:  toPgTimestamp(time.Now().Add(-leasebackend.DefaultMaxLeaseDuration).Add(-time.Minute)), // Expired
		}
		mockQueries.On("FindLeaseById", ctx, leaseID).Return(expiredLease, nil)

		// Mock successful update
		updatedLease := sqlc.Lease{
			ID:         leaseID,
			Identifier: id,
			ClusterID:  clusterID,
			AcquiredAt: toPgTimestamp(time.Now()),
			RenewedAt:  toPgTimestamp(time.Now()),
		}
		mockQueries.On("UpdateLease", ctx, mock.AnythingOfType("sqlc.UpdateLeaseParams")).Return(updatedLease, nil)

		// Act
		acquired, err := backend.TryAcquire(ctx, id, clusterID)

		// Assert
		assert.NoError(t, err)
		assert.True(t, acquired)
		mockQueries.AssertExpectations(t)
	})

	t.Run("FailToAcquireActiveLease", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()
		id := "test-identifier"
		clusterID := "test-cluster"
		leaseID := "lease-test-cluster"

		// Mock active lease held by another identifier
		activeLease := sqlc.Lease{
			ID:         leaseID,
			Identifier: "other-identifier",
			ClusterID:  clusterID,
			AcquiredAt: toPgTimestamp(time.Now().Add(-time.Minute)),
			RenewedAt:  toPgTimestamp(time.Now()), // Recent renewal
		}
		mockQueries.On("FindLeaseById", ctx, leaseID).Return(activeLease, nil)

		// Act
		acquired, err := backend.TryAcquire(ctx, id, clusterID)

		// Assert
		assert.NoError(t, err)
		assert.False(t, acquired)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresLeaseBackend_GetExpiredLeases(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()

		expiredLeases := []sqlc.Lease{
			{
				ID:         "lease-cluster1",
				Identifier: "id1",
				ClusterID:  "cluster1",
				AcquiredAt: toPgTimestamp(time.Now().Add(-time.Hour)),
				RenewedAt:  toPgTimestamp(time.Now().Add(-leasebackend.DefaultMaxLeaseDuration).Add(-time.Minute)),
			},
			{
				ID:         "lease-cluster2",
				Identifier: "id2",
				ClusterID:  "cluster2",
				AcquiredAt: toPgTimestamp(time.Now().Add(-time.Hour)),
				RenewedAt:  toPgTimestamp(time.Now().Add(-leasebackend.DefaultMaxLeaseDuration).Add(-time.Minute)),
			},
		}

		mockQueries.On("GetExpiredLeases", ctx, mock.AnythingOfType("pgtype.Timestamptz")).Return(expiredLeases, nil)

		// Act
		result, err := backend.GetExpiredLeases(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "id1", result[0].Identifier)
		assert.Equal(t, "id2", result[1].Identifier)
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresLeaseBackend_CleanupExpiredLeases(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()

		mockQueries.On("DeleteExpiredLeases", ctx, mock.AnythingOfType("pgtype.Timestamptz")).Return(nil)

		// Act
		err := backend.CleanupExpiredLeases(ctx)

		// Assert
		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()

		mockQueries.On("DeleteExpiredLeases", ctx, mock.AnythingOfType("pgtype.Timestamptz")).Return(errors.New("database error"))

		// Act
		err := backend.CleanupExpiredLeases(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error cleaning up expired leases")
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresLeaseBackend_GetLeaseByCluster(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()
		clusterID := "test-cluster"

		expectedLease := sqlc.Lease{
			ID:         "lease-test-cluster",
			Identifier: "test-id",
			ClusterID:  clusterID,
			AcquiredAt: toPgTimestamp(time.Now()),
			RenewedAt:  toPgTimestamp(time.Now()),
		}

		mockQueries.On("GetLeaseByClusterId", ctx, clusterID).Return(expectedLease, nil)

		// Act
		result, err := backend.GetLeaseByCluster(ctx, clusterID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-id", result.Identifier)
		assert.Equal(t, clusterID, result.ClusterID)
		mockQueries.AssertExpectations(t)
	})

	t.Run("NotFound", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		backend := createPostgresLeaseBackend(mockQueries, nil)

		ctx := context.Background()
		clusterID := "nonexistent-cluster"

		mockQueries.On("GetLeaseByClusterId", ctx, clusterID).Return(sqlc.Lease{}, pgx.ErrNoRows)

		// Act
		result, err := backend.GetLeaseByCluster(ctx, clusterID)

		// Assert
		assert.NoError(t, err)
		assert.Nil(t, result)
		mockQueries.AssertExpectations(t)
	})
}

func TestLeaseStatus(t *testing.T) {
	tests := []struct {
		name              string
		lease             *sqlc.Lease
		id                string
		clusterID         string
		expectedAcquired  bool
		expectedRenewable bool
	}{
		{
			name:              "Nil lease",
			lease:             nil,
			id:                "test",
			clusterID:         "cluster",
			expectedAcquired:  false,
			expectedRenewable: false,
		},
		{
			name: "Expired lease",
			lease: &sqlc.Lease{
				Identifier: "other",
				ClusterID:  "cluster",
				RenewedAt:  toPgTimestamp(time.Now().Add(-leasebackend.DefaultMaxLeaseDuration).Add(-time.Minute)),
			},
			id:                "test",
			clusterID:         "cluster",
			expectedAcquired:  false,
			expectedRenewable: true,
		},
		{
			name: "My active lease",
			lease: &sqlc.Lease{
				Identifier: "test",
				ClusterID:  "cluster",
				RenewedAt:  toPgTimestamp(time.Now()),
			},
			id:                "test",
			clusterID:         "cluster",
			expectedAcquired:  true,
			expectedRenewable: false,
		},
		{
			name: "Other's active lease",
			lease: &sqlc.Lease{
				Identifier: "other",
				ClusterID:  "cluster",
				RenewedAt:  toPgTimestamp(time.Now()),
			},
			id:                "test",
			clusterID:         "cluster",
			expectedAcquired:  false,
			expectedRenewable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acquired, renewable := leaseStatus(tt.lease, tt.id, tt.clusterID)
			assert.Equal(t, tt.expectedAcquired, acquired)
			assert.Equal(t, tt.expectedRenewable, renewable)
		})
	}
}

func TestNewLeaseID(t *testing.T) {
	tests := []struct {
		clusterID string
		expected  string
	}{
		{"cluster1", "lease-cluster1"},
		{"test-cluster", "lease-test-cluster"},
		{"", "lease-"},
	}

	for _, tt := range tests {
		t.Run(tt.clusterID, func(t *testing.T) {
			result := newLeaseID(tt.clusterID)
			assert.Equal(t, tt.expected, result)
		})
	}
}
