package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	leasebackendpostgres "github.com/kubeshop/testkube/pkg/repository/leasebackend/postgres"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	testworkflowpostgres "github.com/kubeshop/testkube/pkg/repository/testworkflow/postgres"
)

// PostgreSQL Factory Implementation
type PostgreSQLFactory struct {
	db               *pgxpool.Pool
	schedulerDb      *database.DB
	leaseBackendRepo leasebackend.Repository
	testWorkflowRepo testworkflow.Repository
}

type PostgreSQLFactoryConfig struct {
	Database    *pgxpool.Pool
	SchedulerDb *database.DB
}

func NewPostgreSQLFactory(config PostgreSQLFactoryConfig) *PostgreSQLFactory {
	factory := &PostgreSQLFactory{
		db:          config.Database,
		schedulerDb: config.SchedulerDb,
	}

	return factory
}

func (f *PostgreSQLFactory) NewLeaseBackendRepository() leasebackend.Repository {
	if f.leaseBackendRepo == nil {
		f.leaseBackendRepo = leasebackendpostgres.NewPostgresLeaseBackend(f.db)
	}
	return f.leaseBackendRepo
}

func (f *PostgreSQLFactory) NewTestWorkflowRepository() testworkflow.Repository {
	if f.testWorkflowRepo == nil {
		f.testWorkflowRepo = testworkflowpostgres.NewPostgresRepository(
			f.db,
		)
	}
	return f.testWorkflowRepo
}

func (f *PostgreSQLFactory) NewScheduler() scheduling.Scheduler {
	return scheduling.NewPostgresScheduler(f.schedulerDb)
}

func (f *PostgreSQLFactory) NewExecutionController() scheduling.Controller {
	return scheduling.NewPostgresExecutionController(f.schedulerDb)
}

func (f *PostgreSQLFactory) NewExecutionQuerier() scheduling.ExecutionQuerier {
	return scheduling.NewPostgresExecutionQuerier(f.schedulerDb)
}

func (f *PostgreSQLFactory) GetDatabaseType() DatabaseType {
	return DatabaseTypePostgreSQL
}

func (f *PostgreSQLFactory) Close(ctx context.Context) error {
	f.db.Close()
	return nil
}

func (f *PostgreSQLFactory) HealthCheck(ctx context.Context) error {
	return f.db.Ping(ctx)
}
