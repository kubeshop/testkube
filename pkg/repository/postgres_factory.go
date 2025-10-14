package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	leasebackendpostgres "github.com/kubeshop/testkube/pkg/repository/leasebackend/postgres"
	"github.com/kubeshop/testkube/pkg/repository/result"
	resultpostgres "github.com/kubeshop/testkube/pkg/repository/result/postgres"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	testresultpostgres "github.com/kubeshop/testkube/pkg/repository/testresult/postgres"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	testworkflowpostgres "github.com/kubeshop/testkube/pkg/repository/testworkflow/postgres"
)

// PostgreSQL Factory Implementation
type PostgreSQLFactory struct {
	db               *pgxpool.Pool
	leaseBackendRepo leasebackend.Repository
	resultRepo       result.Repository
	testResultRepo   testresult.Repository
	testWorkflowRepo testworkflow.Repository
}

type PostgreSQLFactoryConfig struct {
	Database *pgxpool.Pool
}

func NewPostgreSQLFactory(config PostgreSQLFactoryConfig) *PostgreSQLFactory {
	factory := &PostgreSQLFactory{
		db: config.Database,
	}

	return factory
}

func (f *PostgreSQLFactory) NewLeaseBackendRepository() leasebackend.Repository {
	if f.leaseBackendRepo == nil {
		f.leaseBackendRepo = leasebackendpostgres.NewPostgresLeaseBackend(f.db)
	}
	return f.leaseBackendRepo
}

func (f *PostgreSQLFactory) NewResultRepository() result.Repository {
	if f.resultRepo == nil {
		f.resultRepo = resultpostgres.NewPostgresRepository(
			f.db,
		)
	}
	return f.resultRepo
}

func (f *PostgreSQLFactory) NewTestResultRepository() testresult.Repository {
	if f.testResultRepo == nil {
		f.testResultRepo = testresultpostgres.NewPostgresRepository(
			f.db,
		)
	}
	return f.testResultRepo
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
	return scheduling.NewPostgresScheduler()
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
