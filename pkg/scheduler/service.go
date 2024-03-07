package scheduler

import (
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/repository/config"

	executorsv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testsv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsourcesv1 "github.com/kubeshop/testkube-operator/pkg/client/testsources/v1"
	testsuiteexecutionsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testsuiteexecutions/v1"
	testsuitesv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/featureflags"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
)

type Scheduler struct {
	metrics                   v1.Metrics
	executor                  client.Executor
	containerExecutor         client.Executor
	testResults               result.Repository
	testsuiteResults          testresult.Repository
	executorsClient           executorsv1.Interface
	testsClient               testsv3.Interface
	testSuitesClient          testsuitesv3.Interface
	testSourcesClient         testsourcesv1.Interface
	secretClient              secret.Interface
	events                    *event.Emitter
	logger                    *zap.SugaredLogger
	configMap                 config.Repository
	configMapClient           configmap.Interface
	testSuiteExecutionsClient testsuiteexecutionsclientv1.Interface
	eventsBus                 bus.Bus
	dashboardURI              string
	featureFlags              featureflags.FeatureFlags
	logsStream                logsclient.Stream
	subscriptionChecker       checktcl.SubscriptionChecker
	namespace                 string
	agentAPITLSSecret         string
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
	testSuiteExecutionsClient testsuiteexecutionsclientv1.Interface,
	eventsBus bus.Bus,
	dashboardURI string,
	featureFlags featureflags.FeatureFlags,
	logsStream logsclient.Stream,
	namespace string,
	agentAPITLSSecret string,
) *Scheduler {
	return &Scheduler{
		metrics:                   metrics,
		executor:                  executor,
		containerExecutor:         containerExecutor,
		secretClient:              secretClient,
		testResults:               executionResults,
		testsuiteResults:          testExecutionResults,
		executorsClient:           executorsClient,
		testsClient:               testsClient,
		testSuitesClient:          testSuitesClient,
		testSourcesClient:         testSourcesClient,
		events:                    events,
		logger:                    logger,
		configMap:                 configMap,
		configMapClient:           configMapClient,
		testSuiteExecutionsClient: testSuiteExecutionsClient,
		eventsBus:                 eventsBus,
		dashboardURI:              dashboardURI,
		featureFlags:              featureFlags,
		logsStream:                logsStream,
		namespace:                 namespace,
		agentAPITLSSecret:         agentAPITLSSecret,
	}
}

// WithSubscriptionChecker sets subscription checker for the Scheduler
// This is used to check if Pro/Enterprise subscription is valid
func (s *Scheduler) WithSubscriptionChecker(subscriptionChecker checktcl.SubscriptionChecker) *Scheduler {
	s.subscriptionChecker = subscriptionChecker
	return s
}
