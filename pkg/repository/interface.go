package repository

import (
	"context"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// DatabaseType represents the type of database backend
type DatabaseType string

const (
	DatabaseTypeMongoDB    DatabaseType = "mongodb"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
)

// RepositoryFactory defines the interface for creating repository instances
type RepositoryFactory interface {
	// LeaseBackend Repository
	NewLeaseBackendRepository() leasebackend.Repository

	// Result Repository (Test Executions)
	NewResultRepository() result.Repository

	// TestResult Repository (Test Suite Executions)
	NewTestResultRepository() testresult.Repository

	// TestWorkflow Repository (Test Workflow Executions)
	NewTestWorkflowRepository() testworkflow.Repository

	// TestWorkflow Execution Scheduler
	NewScheduler() scheduling.Scheduler

	// TestWorkflow Execution Querier & Controller (Pausing, Aborting, etc)
	NewExecutionController() scheduling.Controller
	NewExecutionQuerier() scheduling.ExecutionQuerier

	// Utility methods
	GetDatabaseType() DatabaseType
	Close(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

// DatabaseRepository defines the interface for database repository
type DatabaseRepository interface {
	// LeaseBackend Repository
	LeaseBackend() leasebackend.Repository

	// Result Repository (Test Executions)
	Result() result.Repository

	// TestResult Repository (Test Suite Executions)
	TestResult() testresult.Repository

	// TestWorkflow Repository (Test Workflow Executions)
	TestWorkflow() testworkflow.Repository

	// Utility methods
	GetDatabaseType() DatabaseType
	Close(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}
