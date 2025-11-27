package triggers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/coordination/leader"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	testkubeclientsetv1 "github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	defaultScraperInterval        = 5 * time.Second
	defaultConditionsCheckBackoff = 1 * time.Second
	defaultConditionsCheckTimeout = 60 * time.Second
	defaultProbesCheckBackoff     = 1 * time.Second
	defaultProbesCheckTimeout     = 60 * time.Second
	defaultClusterID              = "testkube-api"
	defaultIdentifierFormat       = "testkube-api-%s"
)

type Service struct {
	informers                     *k8sInformers
	identifier                    string
	clusterID                     string
	agentName                     string
	triggerExecutor               ExecutorF
	scraperInterval               time.Duration
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
	httpClient                    http.HttpClient
	eventsBus                     bus.Bus
	metrics                       metrics.Metrics
	executionWorkerClient         executionworkertypes.Worker
	testWorkflowExecutor          testworkflowexecutor.TestWorkflowExecutor
	testWorkflowResultsRepository testworkflow.Repository
	testkubeNamespace             string
	watcherNamespaces             []string
	proContext                    *intconfig.ProContext
	testTriggerControlPlane       bool
	eventLabels                   map[string]string
	Agent                         watcherAgent
}

type Option func(*Service)

// NewService builds leader-only tasks for trigger watching and scraping.
func NewService(
	agentName string,
	clientset kubernetes.Interface,
	testKubeClientset testkubeclientsetv1.Interface,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testTriggersClient testtriggerclient.TestTriggerClient,
	logger *zap.SugaredLogger,
	eventsBus bus.Bus,
	metrics metrics.Metrics,
	executionWorkerClient executionworkertypes.Worker,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	testWorkflowResultsRepository testworkflow.Repository,
	proContext *intconfig.ProContext,
	opts ...Option,
) []leader.Task {
	identifier := fmt.Sprintf(defaultIdentifierFormat, utils.RandAlphanum(10))
	service := &Service{
		identifier:                    identifier,
		clusterID:                     defaultClusterID,
		agentName:                     agentName,
		scraperInterval:               defaultScraperInterval,
		defaultConditionsCheckTimeout: defaultConditionsCheckTimeout,
		defaultConditionsCheckBackoff: defaultConditionsCheckBackoff,
		defaultProbesCheckTimeout:     defaultProbesCheckTimeout,
		defaultProbesCheckBackoff:     defaultProbesCheckBackoff,
		clientset:                     clientset,
		testKubeClientset:             testKubeClientset,
		testWorkflowsClient:           testWorkflowsClient,
		testTriggersClient:            testTriggersClient,
		logger:                        logger,
		eventsBus:                     eventsBus,
		metrics:                       metrics,
		executionWorkerClient:         executionWorkerClient,
		testWorkflowExecutor:          testWorkflowExecutor,
		testWorkflowResultsRepository: testWorkflowResultsRepository,
		httpClient:                    http.NewClient(),
		watchFromDate:                 time.Now(),
		triggerStatus:                 make(map[statusKey]*triggerStatus),
		proContext:                    proContext,
	}
	if service.triggerExecutor == nil {
		service.triggerExecutor = service.execute
	}

	for _, opt := range opts {
		opt(service)
	}

	// Initialize agent snapshot from proContext if available
	service.Agent = watcherAgent{}
	if service.proContext != nil {
		service.Agent.Name = service.proContext.Agent.Name
		service.Agent.Labels = service.proContext.Agent.Labels
	}

	return []leader.Task{
		{
			Name: "trigger-watcher",
			Start: func(taskCtx context.Context) error {
				service.runWatcher(taskCtx)
				return nil
			},
		},
		{
			Name: "trigger-scraper",
			Start: func(taskCtx context.Context) error {
				service.runExecutionScraper(taskCtx)
				return nil
			},
		},
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

// WithCoordinator is retained for compatibility but is a no-op: tasks are expected
// to be registered on an external coordinator by the caller.
func WithCoordinator(coordinator *leader.Coordinator) Option {
	return func(s *Service) {
	}
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
