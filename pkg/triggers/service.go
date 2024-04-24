package triggers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsuitev3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	testkubeclientsetv1 "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	defaultScraperInterval        = 5 * time.Second
	defaultLeaseCheckInterval     = 5 * time.Second
	defaultMaxLeaseDuration       = 1 * time.Minute
	defaultConditionsCheckBackoff = 1 * time.Second
	defaultConditionsCheckTimeout = 60 * time.Second
	defaultProbesCheckBackoff     = 1 * time.Second
	defaultProbesCheckTimeout     = 60 * time.Second
	defaultClusterID              = "testkube-api"
	defaultIdentifierFormat       = "testkube-api-%s"
)

type Service struct {
	informers                     *k8sInformers
	leaseBackend                  LeaseBackend
	identifier                    string
	clusterID                     string
	triggerExecutor               ExecutorF
	scraperInterval               time.Duration
	leaseCheckInterval            time.Duration
	maxLeaseDuration              time.Duration
	defaultConditionsCheckTimeout time.Duration
	defaultConditionsCheckBackoff time.Duration
	defaultProbesCheckTimeout     time.Duration
	defaultProbesCheckBackoff     time.Duration
	watchFromDate                 time.Time
	triggerStatus                 map[statusKey]*triggerStatus
	scheduler                     *scheduler.Scheduler
	clientset                     kubernetes.Interface
	testKubeClientset             testkubeclientsetv1.Interface
	testSuitesClient              testsuitesclientv3.Interface
	testsClient                   testsclientv3.Interface
	testWorkflowsClient           testworkflowsclientv1.Interface
	resultRepository              result.Repository
	testResultRepository          testresult.Repository
	logger                        *zap.SugaredLogger
	configMap                     config.Repository
	executorsClient               executorsclientv1.Interface
	httpClient                    http.HttpClient
	testExecutor                  client.Executor
	eventsBus                     bus.Bus
	metrics                       metrics.Metrics
	testWorkflowExecutor          testworkflowexecutor.TestWorkflowExecutor
	testWorkflowResultsRepository testworkflow.Repository
	testkubeNamespace             string
	watcherNamespaces             []string
	disableSecretCreation         bool
}

type Option func(*Service)

func NewService(
	scheduler *scheduler.Scheduler,
	clientset kubernetes.Interface,
	testKubeClientset testkubeclientsetv1.Interface,
	testSuitesClient testsuitesclientv3.Interface,
	testsClient testsclientv3.Interface,
	testWorkflowsClient testworkflowsclientv1.Interface,
	resultRepository result.Repository,
	testResultRepository testresult.Repository,
	leaseBackend LeaseBackend,
	logger *zap.SugaredLogger,
	configMap config.Repository,
	executorsClient executorsclientv1.Interface,
	testExecutor client.Executor,
	eventsBus bus.Bus,
	metrics metrics.Metrics,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	testWorkflowResultsRepository testworkflow.Repository,
	opts ...Option,
) *Service {
	identifier := fmt.Sprintf(defaultIdentifierFormat, utils.RandAlphanum(10))
	s := &Service{
		identifier:                    identifier,
		clusterID:                     defaultClusterID,
		scraperInterval:               defaultScraperInterval,
		leaseCheckInterval:            defaultLeaseCheckInterval,
		maxLeaseDuration:              defaultMaxLeaseDuration,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		scheduler:                     scheduler,
		clientset:                     clientset,
		testKubeClientset:             testKubeClientset,
		testSuitesClient:              testSuitesClient,
		testWorkflowsClient:           testWorkflowsClient,
		testsClient:                   testsClient,
		resultRepository:              resultRepository,
		testResultRepository:          testResultRepository,
		leaseBackend:                  leaseBackend,
		logger:                        logger,
		configMap:                     configMap,
		executorsClient:               executorsClient,
		testExecutor:                  testExecutor,
		eventsBus:                     eventsBus,
		metrics:                       metrics,
		testWorkflowExecutor:          testWorkflowExecutor,
		testWorkflowResultsRepository: testWorkflowResultsRepository,
		httpClient:                    http.NewClient(),
		watchFromDate:                 time.Now(),
		triggerStatus:                 make(map[statusKey]*triggerStatus),
	}
	if s.triggerExecutor == nil {
		s.triggerExecutor = s.execute
	}

	for _, opt := range opts {
		opt(s)
	}

	s.informers = newK8sInformers(clientset, testKubeClientset, s.testkubeNamespace, s.watcherNamespaces)

	return s
}

func WithIdentifier(id string) Option {
	return func(s *Service) {
		s.identifier = id
	}
}

func WithHostnameIdentifier() Option {
	return func(s *Service) {
		identifier, err := os.Hostname()
		if err == nil {
			s.identifier = identifier
		}
	}
}

func WithClusterID(id string) Option {
	return func(s *Service) {
		s.clusterID = id
	}
}

func WithWatchFromDate(from time.Time) Option {
	return func(s *Service) {
		s.watchFromDate = from
	}
}

func WithLeaseCheckerInterval(interval time.Duration) Option {
	return func(s *Service) {
		s.leaseCheckInterval = interval
	}
}

func WithScraperInterval(interval time.Duration) Option {
	return func(s *Service) {
		s.scraperInterval = interval
	}
}

func WithExecutor(triggerExecutor ExecutorF) Option {
	return func(s *Service) {
		s.triggerExecutor = triggerExecutor
	}
}

func WithTestkubeNamespace(namespace string) Option {
	return func(s *Service) {
		s.testkubeNamespace = namespace
	}
}

func WithWatcherNamespaces(namespaces string) Option {
	return func(s *Service) {
		for _, namespace := range strings.Split(namespaces, ",") {
			value := strings.TrimSpace(namespace)
			if value != "" {
				s.watcherNamespaces = append(s.watcherNamespaces, value)
			}
		}
	}
}

func WithDisableSecretCreation(disableSecretCreation bool) Option {
	return func(s *Service) {
		s.disableSecretCreation = disableSecretCreation
	}
}

func (s *Service) Run(ctx context.Context) {
	leaseChan := make(chan bool)

	go s.runLeaseChecker(ctx, leaseChan)

	go s.runWatcher(ctx, leaseChan)

	go s.runExecutionScraper(ctx)
}

func (s *Service) addTrigger(t *testtriggersv1.TestTrigger) {
	key := newStatusKey(t.Namespace, t.Name)
	s.triggerStatus[key] = newTriggerStatus(t)
}

func (s *Service) updateTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(target.Namespace, target.Name)
	if s.triggerStatus[key] != nil {
		s.triggerStatus[key].testTrigger = target
	} else {
		s.triggerStatus[key] = newTriggerStatus(target)
	}
}

func (s *Service) removeTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(target.Namespace, target.Name)
	delete(s.triggerStatus, key)
}

func (s *Service) addTest(test *testsv3.Test) {
	ctx := context.Background()
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		s.logger.Debugw("getting telemetry enabled error", "error", err)
	}

	if !telemetryEnabled {
		return
	}

	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		s.logger.Debugw("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		s.logger.Debugw("getting hostname error", "hostname", host, "error", err)
	}

	var dataSource string
	if test.Spec.Content != nil {
		dataSource = string(test.Spec.Content.Type_)
	}

	out, err := telemetry.SendCreateEvent("testkube_api_create_test", telemetry.CreateParams{
		AppVersion: version.Version,
		DataSource: dataSource,
		Host:       host,
		ClusterID:  clusterID,
		TestType:   test.Spec.Type_,
		TestSource: test.Spec.Source,
	})
	if err != nil {
		s.logger.Debugw("sending create test telemetry event error", "error", err)
	} else {
		s.logger.Debugw("sending create test telemetry event", "output", out)
	}

	if test.Labels == nil {
		test.Labels = make(map[string]string)
	}

	test.Labels[testkube.TestLabelTestType] = utils.SanitizeName(test.Spec.Type_)
	executorCR, err := s.executorsClient.GetByType(test.Spec.Type_)
	if err == nil {
		test.Labels[testkube.TestLabelExecutor] = executorCR.Name
	} else {
		s.logger.Debugw("can't get executor spec", "error", err)
	}

	if _, err = s.testsClient.Update(test, s.disableSecretCreation); err != nil {
		s.logger.Debugw("can't update test spec", "error", err)
	}
}

func (s *Service) updateTest(test *testsv3.Test) {
	changed := false
	if test.Labels == nil {
		test.Labels = make(map[string]string)
	}

	testType := utils.SanitizeName(test.Spec.Type_)
	if test.Labels[testkube.TestLabelTestType] != testType {
		test.Labels[testkube.TestLabelTestType] = testType
		changed = true
	}

	executorCR, err := s.executorsClient.GetByType(test.Spec.Type_)
	if err == nil {
		if test.Labels[testkube.TestLabelExecutor] != executorCR.Name {
			test.Labels[testkube.TestLabelExecutor] = executorCR.Name
			changed = true
		}
	} else {
		s.logger.Debugw("can't get executor spec", "error", err)
	}

	if changed {
		if _, err = s.testsClient.Update(test, s.disableSecretCreation); err != nil {
			s.logger.Debugw("can't update test spec", "error", err)
		}
	}
}

func (s *Service) addTestSuite(testSuite *testsuitev3.TestSuite) {
	ctx := context.Background()
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		s.logger.Debugw("getting telemetry enabled error", "error", err)
	}

	if !telemetryEnabled {
		return
	}

	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		s.logger.Debugw("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		s.logger.Debugw("getting hostname error", "hostname", host, "error", err)
	}

	out, err := telemetry.SendCreateEvent("testkube_api_create_test_suite", telemetry.CreateParams{
		AppVersion:     version.Version,
		Host:           host,
		ClusterID:      clusterID,
		TestSuiteSteps: int32(len(testSuite.Spec.Before) + len(testSuite.Spec.Steps) + len(testSuite.Spec.After)),
	})
	if err != nil {
		s.logger.Debugw("sending create test suite telemetry event error", "error", err)
	} else {
		s.logger.Debugw("sending create test suite telemetry event", "output", out)
	}
}
