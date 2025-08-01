package repository

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func TestFactoryBuilder(t *testing.T) {
	t.Run("MongoDB Factory", func(t *testing.T) {
		// This would require a real MongoDB connection in integration tests
		// For unit tests, we'd mock the database
		t.Skip("Requires MongoDB connection")
	})

	t.Run("PostgreSQL Factory", func(t *testing.T) {
		// This would require a real PostgreSQL connection in integration tests
		// For unit tests, we'd mock the database
		t.Skip("Requires PostgreSQL connection")
	})

	t.Run("Builder Validation", func(t *testing.T) {
		builder := NewFactoryBuilder()

		// Should fail without configuration
		_, err := builder.Build()
		assert.Error(t, err)

		// Should fail with invalid database type
		builder.databaseType = "invalid"
		_, err = builder.Build()
		assert.Error(t, err)
	})
}

func TestRepositoryManager(t *testing.T) {
	// Mock factory for testing

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFactory := &MockRepositoryFactory{ctr: mockCtrl}
	manager := NewRepositoryManager(mockFactory)

	t.Run("Repository Access", func(t *testing.T) {
		// Test that manager provides access to all repositories
		assert.NotNil(t, manager.LeaseBackend())
		assert.NotNil(t, manager.Result())
		assert.NotNil(t, manager.TestResult())
		assert.NotNil(t, manager.TestWorkflow())
	})

	t.Run("Database Type", func(t *testing.T) {
		mockFactory.On("GetDatabaseType").Return(DatabaseTypePostgreSQL)
		assert.Equal(t, DatabaseTypePostgreSQL, manager.GetDatabaseType())
	})
}

// Mock implementations for testing
type MockRepositoryFactory struct {
	ctr *gomock.Controller
	mock.Mock
}

func (m *MockRepositoryFactory) NewLeaseBackendRepository() leasebackend.Repository {
	return leasebackend.NewMockRepository(m.ctr)
}

func (m *MockRepositoryFactory) NewResultRepository() result.Repository {
	return result.NewMockRepository(m.ctr)
}

func (m *MockRepositoryFactory) NewTestResultRepository() testresult.Repository {
	return testresult.NewMockRepository(m.ctr)
}

func (m *MockRepositoryFactory) NewTestWorkflowRepository() testworkflow.Repository {
	return testworkflow.NewMockRepository(m.ctr)
}

func (m *MockRepositoryFactory) GetDatabaseType() DatabaseType {
	args := m.Called()
	return args.Get(0).(DatabaseType)
}

func (m *MockRepositoryFactory) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepositoryFactory) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
