package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	leasebackendmongo "github.com/kubeshop/testkube/pkg/repository/leasebackend/mongo"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/result/minio"
	resultmongo "github.com/kubeshop/testkube/pkg/repository/result/mongo"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
	sequencemongo "github.com/kubeshop/testkube/pkg/repository/sequence/mongo"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	testresultmongo "github.com/kubeshop/testkube/pkg/repository/testresult/mongo"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	testworkflowmongo "github.com/kubeshop/testkube/pkg/repository/testworkflow/mongo"
)

// MongoDB Factory Implementation
type MongoDBFactory struct {
	db               *mongo.Database
	allowDiskUse     bool
	isDocDb          bool
	logGrpcClient    logsclient.StreamGetter
	sequenceRepo     sequence.Repository
	outputRepository *minio.MinioRepository
	leaseBackendRepo leasebackend.Repository
	resultRepo       result.Repository
	testResultRepo   testresult.Repository
	testWorkflowRepo testworkflow.Repository
}

type MongoDBFactoryConfig struct {
	Database         *mongo.Database
	AllowDiskUse     bool
	IsDocDb          bool
	LogGrpcClient    logsclient.StreamGetter
	OutputRepository *minio.MinioRepository
}

func NewMongoDBFactory(config MongoDBFactoryConfig) *MongoDBFactory {
	factory := &MongoDBFactory{
		db:               config.Database,
		allowDiskUse:     config.AllowDiskUse,
		isDocDb:          config.IsDocDb,
		logGrpcClient:    config.LogGrpcClient,
		outputRepository: config.OutputRepository,
	}

	// Initialize sequence repository first as it's used by other repositories
	factory.sequenceRepo = sequencemongo.NewMongoRepository(config.Database)

	return factory
}

func (f *MongoDBFactory) NewLeaseBackendRepository() leasebackend.Repository {
	if f.leaseBackendRepo == nil {
		f.leaseBackendRepo = leasebackendmongo.NewMongoLeaseBackend(f.db)
	}
	return f.leaseBackendRepo
}

func (f *MongoDBFactory) NewResultRepository() result.Repository {
	if f.resultRepo == nil {
		opts := []resultmongo.MongoRepositoryOpt{
			resultmongo.WithLogsClient(f.logGrpcClient),
			resultmongo.WithMongoRepositorySequence(f.sequenceRepo),
		}

		if f.outputRepository != nil {
			opts = append(opts, resultmongo.WithMinioOutputRepository(f.outputRepository))
		}

		f.resultRepo = resultmongo.NewMongoRepository(
			f.db,
			f.allowDiskUse,
			f.isDocDb,
			opts...,
		)
	}
	return f.resultRepo
}

func (f *MongoDBFactory) NewTestResultRepository() testresult.Repository {
	if f.testResultRepo == nil {
		f.testResultRepo = testresultmongo.NewMongoRepository(
			f.db,
			f.allowDiskUse,
			f.isDocDb,
			testresultmongo.WithMongoRepositorySequence(f.sequenceRepo),
		)
	}
	return f.testResultRepo
}

func (f *MongoDBFactory) NewTestWorkflowRepository() testworkflow.Repository {
	if f.testWorkflowRepo == nil {
		f.testWorkflowRepo = testworkflowmongo.NewMongoRepository(
			f.db,
			f.allowDiskUse,
			testworkflowmongo.WithMongoRepositorySequence(f.sequenceRepo),
		)
	}
	return f.testWorkflowRepo
}

func (f *MongoDBFactory) NewScheduler() scheduling.Scheduler {
	executionsCollection := f.db.Collection(testworkflowmongo.CollectionName)
	return scheduling.NewMongoScheduler(executionsCollection)
}

func (f *MongoDBFactory) NewExecutionController() scheduling.Controller {
	executionsCollection := f.db.Collection(testworkflowmongo.CollectionName)
	return scheduling.NewMongoExecutionController(executionsCollection)
}

func (f *MongoDBFactory) NewExecutionQuerier() scheduling.ExecutionQuerier {
	executionsCollection := f.db.Collection(testworkflowmongo.CollectionName)
	return scheduling.NewMongoExecutionQuerier(executionsCollection)
}

func (f *MongoDBFactory) NewSequenceRepository() sequence.Repository {
	return f.sequenceRepo
}

func (f *MongoDBFactory) GetDatabaseType() DatabaseType {
	return DatabaseTypeMongoDB
}

func (f *MongoDBFactory) Close(ctx context.Context) error {
	return f.db.Client().Disconnect(ctx)
}

func (f *MongoDBFactory) HealthCheck(ctx context.Context) error {
	return f.db.Client().Ping(ctx, nil)
}
