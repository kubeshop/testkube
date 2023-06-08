package scheduler

import (
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/repository/config"

	executorsv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	testsv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsourcesv1 "github.com/kubeshop/testkube-operator/client/testsources/v1"
	testsuitesv3 "github.com/kubeshop/testkube-operator/client/testsuites/v3"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/secret"
)

type Scheduler struct {
	metrics              v1.Metrics
	executor             client.Executor
	containerExecutor    client.Executor
	executionResults     result.Repository
	testExecutionResults testresult.Repository
	executorsClient      executorsv1.Interface
	testsClient          testsv3.Interface
	testSuitesClient     testsuitesv3.Interface
	testSourcesClient    testsourcesv1.Interface
	secretClient         secret.Interface
	events               *event.Emitter
	logger               *zap.SugaredLogger
	configMap            config.Repository
	configMapClient      configmap.Interface
}

func NewScheduler(
	metrics v1.Metrics,
	executor client.Executor,
	containerExecutor client.Executor,
	executionResults result.Repository,
	testExecutionResults testresult.Repository,
	executorsClient executorsv1.Interface,
	testsClient testsv3.Interface,
	testSuitesClient testsuitesv3.Interface,
	testSourcesClient testsourcesv1.Interface,
	secretClient secret.Interface,
	events *event.Emitter,
	logger *zap.SugaredLogger,
	configMap config.Repository,
	configMapClient configmap.Interface,
) *Scheduler {
	return &Scheduler{
		metrics:              metrics,
		executor:             executor,
		containerExecutor:    containerExecutor,
		secretClient:         secretClient,
		executionResults:     executionResults,
		testExecutionResults: testExecutionResults,
		executorsClient:      executorsClient,
		testsClient:          testsClient,
		testSuitesClient:     testSuitesClient,
		testSourcesClient:    testSourcesClient,
		events:               events,
		logger:               logger,
		configMap:            configMap,
		configMapClient:      configMapClient,
	}
}
