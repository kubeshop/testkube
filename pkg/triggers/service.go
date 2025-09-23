package triggers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsuitev3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	testkubeclientsetv1 "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	defaultScraperInterval        = 5 * time.Second
	defaultLeaseCheckInterval     = 5 * time.Second
	defaultConditionsCheckBackoff = 1 * time.Second
	defaultConditionsCheckTimeout = 60 * time.Second
	defaultProbesCheckBackoff     = 1 * time.Second
	defaultProbesCheckTimeout     = 60 * time.Second
	defaultClusterID              = "testkube-api"
	defaultIdentifierFormat       = "testkube-api-%s"
)

type Service struct {
	informers                     *k8sInformers
	leaseBackend                  leasebackend.Repository
	identifier                    string
	clusterID                     string
	agentName                     string
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
	clientset                     kubernetes.Interface
	testKubeClientset             testkubeclientsetv1.Interface
	testWorkflowsClient           testworkflowclient.TestWorkflowClient
	testTriggersClient            testtriggerclient.TestTriggerClient
	logger                        *zap.SugaredLogger
	configMap                     config.Repository
	httpClient                    http.HttpClient
	eventsBus                     bus.Bus
	metrics                       metrics.Metrics
	executionWorkerClient         executionworkertypes.Worker
	testWorkflowExecutor          testworkflowexecutor.TestWorkflowExecutor
	testWorkflowResultsRepository testworkflow.Repository
	testkubeNamespace             string
	watcherNamespaces             []string
	disableSecretCreation         bool
	deprecatedSystem              *services.DeprecatedSystem
	proContext                    *intconfig.ProContext
	testTriggerControlPlane       bool
	eventLabels                   map[string]string
	Agent                         watcherAgent
}

type Option func(*Service)

func NewService(
	agentName string,
	deprecatedSystem *services.DeprecatedSystem,
	clientset kubernetes.Interface,
	testKubeClientset testkubeclientsetv1.Interface,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testTriggersClient testtriggerclient.TestTriggerClient,
	leaseBackend leasebackend.Repository,
	logger *zap.SugaredLogger,
	configMap config.Repository,
	eventsBus bus.Bus,
	metrics metrics.Metrics,
	executionWorkerClient executionworkertypes.Worker,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	testWorkflowResultsRepository testworkflow.Repository,
	proContext *intconfig.ProContext,
	opts ...Option,
) *Service {
	identifier := fmt.Sprintf(defaultIdentifierFormat, utils.RandAlphanum(10))
	s := &Service{
		identifier:                    identifier,
		clusterID:                     defaultClusterID,
		agentName:                     agentName,
		scraperInterval:               defaultScraperInterval,
		leaseCheckInterval:            defaultLeaseCheckInterval,
		maxLeaseDuration:              leasebackend.DefaultMaxLeaseDuration,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		clientset:                     clientset,
		testKubeClientset:             testKubeClientset,
		testWorkflowsClient:           testWorkflowsClient,
		testTriggersClient:            testTriggersClient,
		leaseBackend:                  leaseBackend,
		logger:                        logger,
		configMap:                     configMap,
		eventsBus:                     eventsBus,
		metrics:                       metrics,
		executionWorkerClient:         executionWorkerClient,
		testWorkflowExecutor:          testWorkflowExecutor,
		testWorkflowResultsRepository: testWorkflowResultsRepository,
		httpClient:                    http.NewClient(),
		watchFromDate:                 time.Now(),
		triggerStatus:                 make(map[statusKey]*triggerStatus),
		deprecatedSystem:              deprecatedSystem,
		proContext:                    proContext,
	}
	if s.triggerExecutor == nil {
		s.triggerExecutor = s.execute
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize agent snapshot from proContext if available
	s.Agent = watcherAgent{}
	if s.proContext != nil {
		s.Agent.Name = s.proContext.Agent.Name
		s.Agent.Labels = s.proContext.Agent.Labels
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

// WithTestTriggerControlPlane enables Control Plane-backed trigger watching
func WithTestTriggerControlPlane(enabled bool) Option {
	return func(s *Service) {
		s.testTriggerControlPlane = enabled
	}
}

func WithEventLabels(eventLabels map[string]string) Option {
	return func(s *Service) {
		s.eventLabels = eventLabels
	}
}

func (s *Service) Run(ctx context.Context) {
	leaseChan := make(chan bool)

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		s.runLeaseChecker(ctx, leaseChan)
		wg.Done()
	}()
	go func() {
		s.runWatcher(ctx, leaseChan)
		wg.Done()
	}()
	go func() {
		s.runExecutionScraper(ctx)
		wg.Done()
	}()
	wg.Wait()
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
	executorCR, err := s.deprecatedSystem.Clients.Executors().GetByType(test.Spec.Type_)
	if err == nil {
		test.Labels[testkube.TestLabelExecutor] = executorCR.Name
	} else {
		s.logger.Debugw("can't get executor spec", "error", err)
	}

	if _, err = s.deprecatedSystem.Clients.Tests().Update(test, s.disableSecretCreation); err != nil {
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

	executorCR, err := s.deprecatedSystem.Clients.Executors().GetByType(test.Spec.Type_)
	if err == nil {
		if test.Labels[testkube.TestLabelExecutor] != executorCR.Name {
			test.Labels[testkube.TestLabelExecutor] = executorCR.Name
			changed = true
		}
	} else {
		s.logger.Debugw("can't get executor spec", "error", err)
	}

	if changed {
		if _, err = s.deprecatedSystem.Clients.Tests().Update(test, s.disableSecretCreation); err != nil {
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
