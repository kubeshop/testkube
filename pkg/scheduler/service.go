package scheduler

import (
	executorsv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	testsv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsuitesv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	v1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/secret"
	"go.uber.org/zap"
)

type Scheduler struct {
	metrics              v1.Metrics
	executor             client.Executor
	executionResults     result.Repository
	testExecutionResults testresult.Repository
	executorsClient      executorsv1.Interface
	testsClient          testsv3.Interface
	testSuitesClient     testsuitesv2.Interface
	secretClient         secret.Interface
	events               *event.Emitter
	logger               *zap.SugaredLogger
}

func NewRunner(
	executor client.Executor,
	executionResults result.Repository,
	testExecutionResults testresult.Repository,
	executorsClient executorsv1.Interface,
	testsClient testsv3.Interface,
	testSuitesClient testsuitesv2.Interface,
	secretClient secret.Interface,
	events *event.Emitter,
	logger *zap.SugaredLogger,
) *Scheduler {
	return &Scheduler{
		executor:             executor,
		secretClient:         secretClient,
		executionResults:     executionResults,
		testExecutionResults: testExecutionResults,
		executorsClient:      executorsClient,
		testsClient:          testsClient,
		testSuitesClient:     testSuitesClient,
		events:               events,
		logger:               logger,
	}
}
