package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
)

// MockQueriesInterface implementation
type MockQueriesInterface struct {
	mock.Mock
}

func (m *MockQueriesInterface) UpsertAndIncrementExecutionSequence(ctx context.Context, arg sqlc.UpsertAndIncrementExecutionSequenceParams) (sqlc.ExecutionSequence, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sqlc.ExecutionSequence), args.Error(1)
}

func (m *MockQueriesInterface) DeleteExecutionSequence(ctx context.Context, arg sqlc.DeleteExecutionSequenceParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQueriesInterface) DeleteExecutionSequences(ctx context.Context, arg sqlc.DeleteExecutionSequencesParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQueriesInterface) DeleteAllExecutionSequences(ctx context.Context, arg sqlc.DeleteAllExecutionSequencesParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func TestPostgresRepository_GetNextExecutionNumber(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		name := "test-execution"

		expectedResult := sqlc.ExecutionSequence{
			Name:   name,
			Number: 5,
		}
		mockQueries.On("UpsertAndIncrementExecutionSequence", ctx, sqlc.UpsertAndIncrementExecutionSequenceParams{Name: name, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(expectedResult, nil)

		// Act
		result, err := repo.GetNextExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, int32(5), result)
		mockQueries.AssertExpectations(t)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("UpsertAndIncrementExecutionSequence", ctx, sqlc.UpsertAndIncrementExecutionSequenceParams{Name: name, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(sqlc.ExecutionSequence{}, errors.New("database error"))

		// Act
		result, err := repo.GetNextExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, int32(0), result)
		assert.Contains(t, err.Error(), "failed to get next execution number")
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_DeleteExecutionNumber(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("DeleteExecutionSequence", ctx, sqlc.DeleteExecutionSequenceParams{Name: name, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(nil)

		// Act
		err := repo.DeleteExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
	})

	t.Run("NotFoundError", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("DeleteExecutionSequence", ctx, sqlc.DeleteExecutionSequenceParams{Name: name, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(pgx.ErrNoRows)

		// Act
		err := repo.DeleteExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.NoError(t, err) // Should not return error for not found
		mockQueries.AssertExpectations(t)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("DeleteExecutionSequence", ctx, sqlc.DeleteExecutionSequenceParams{Name: name, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(errors.New("database error"))

		// Act
		err := repo.DeleteExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete execution sequence")
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_DeleteExecutionNumbers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		names := []string{"exec1", "exec2"}

		mockQueries.On("DeleteExecutionSequences", ctx, sqlc.DeleteExecutionSequencesParams{Names: names, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(nil)

		// Act
		err := repo.DeleteExecutionNumbers(ctx, names, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
	})

	t.Run("EmptyNames", func(t *testing.T) {
		// Arrange
		repo := &PostgresRepository{}

		ctx := context.Background()
		names := []string{}

		// Act
		err := repo.DeleteExecutionNumbers(ctx, names, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()
		names := []string{"exec1", "exec2"}

		mockQueries.On("DeleteExecutionSequences", ctx, sqlc.DeleteExecutionSequencesParams{Names: names, OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(errors.New("database error"))

		// Act
		err := repo.DeleteExecutionNumbers(ctx, names, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete execution sequences")
		mockQueries.AssertExpectations(t)
	})
}

func TestPostgresRepository_DeleteAllExecutionNumbers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()

		mockQueries.On("DeleteAllExecutionSequences", ctx, sqlc.DeleteAllExecutionSequencesParams{OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(nil)

		// Act
		err := repo.DeleteAllExecutionNumbers(ctx, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.NoError(t, err)
		mockQueries.AssertExpectations(t)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries:        mockQueries,
			organizationID: "org-id",
			environmentID:  "env-id",
		}

		ctx := context.Background()

		mockQueries.On("DeleteAllExecutionSequences", ctx, sqlc.DeleteAllExecutionSequencesParams{OrganizationID: "org-id", EnvironmentID: "env-id"}).Return(errors.New("database error"))

		// Act
		err := repo.DeleteAllExecutionNumbers(ctx, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete all execution sequences")
		mockQueries.AssertExpectations(t)
	})
}
