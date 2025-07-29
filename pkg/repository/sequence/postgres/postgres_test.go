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

func (m *MockQueriesInterface) GetExecutionSequence(ctx context.Context, name string) (sqlc.ExecutionSequence, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(sqlc.ExecutionSequence), args.Error(1)
}

func (m *MockQueriesInterface) UpsertAndIncrementExecutionSequence(ctx context.Context, name string) (sqlc.ExecutionSequence, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(sqlc.ExecutionSequence), args.Error(1)
}

func (m *MockQueriesInterface) DeleteExecutionSequence(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockQueriesInterface) DeleteExecutionSequences(ctx context.Context, names []string) error {
	args := m.Called(ctx, names)
	return args.Error(0)
}

func (m *MockQueriesInterface) DeleteAllExecutionSequences(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQueriesInterface) GetAllExecutionSequences(ctx context.Context) ([]sqlc.ExecutionSequence, error) {
	args := m.Called(ctx)
	return args.Get(0).([]sqlc.ExecutionSequence), args.Error(1)
}

func (m *MockQueriesInterface) GetExecutionSequencesByNames(ctx context.Context, names []string) ([]sqlc.ExecutionSequence, error) {
	args := m.Called(ctx, names)
	return args.Get(0).([]sqlc.ExecutionSequence), args.Error(1)
}

func (m *MockQueriesInterface) CountExecutionSequences(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestPostgresRepository_GetNextExecutionNumber(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockQueries := &MockQueriesInterface{}
		repo := &PostgresRepository{
			queries: mockQueries,
		}

		ctx := context.Background()
		name := "test-execution"

		expectedResult := sqlc.ExecutionSequence{
			Name:   name,
			Number: 5,
		}
		mockQueries.On("UpsertAndIncrementExecutionSequence", ctx, name).Return(expectedResult, nil)

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
			queries: mockQueries,
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("UpsertAndIncrementExecutionSequence", ctx, name).Return(sqlc.ExecutionSequence{}, errors.New("database error"))

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
			queries: mockQueries,
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("DeleteExecutionSequence", ctx, name).Return(nil)

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
			queries: mockQueries,
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("DeleteExecutionSequence", ctx, name).Return(pgx.ErrNoRows)

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
			queries: mockQueries,
		}

		ctx := context.Background()
		name := "test-execution"

		mockQueries.On("DeleteExecutionSequence", ctx, name).Return(errors.New("database error"))

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
			queries: mockQueries,
		}

		ctx := context.Background()
		names := []string{"exec1", "exec2"}

		mockQueries.On("DeleteExecutionSequences", ctx, names).Return(nil)

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
			queries: mockQueries,
		}

		ctx := context.Background()
		names := []string{"exec1", "exec2"}

		mockQueries.On("DeleteExecutionSequences", ctx, names).Return(errors.New("database error"))

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
			queries: mockQueries,
		}

		ctx := context.Background()

		mockQueries.On("DeleteAllExecutionSequences", ctx).Return(nil)

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
			queries: mockQueries,
		}

		ctx := context.Background()

		mockQueries.On("DeleteAllExecutionSequences", ctx).Return(errors.New("database error"))

		// Act
		err := repo.DeleteAllExecutionNumbers(ctx, sequence.ExecutionTypeTestWorkflow)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete all execution sequences")
		mockQueries.AssertExpectations(t)
	})
}
