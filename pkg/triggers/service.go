package triggers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
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
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
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

// Service watches Kubernetes resources and TestTrigger CRDs in-cluster,
// matches events against registered triggers, and executes actions.
//
// Lock ordering (deadlock-avoidance): acquire triggerStatusMu before any
// per-status RWMutex. snapshotStatuses takes triggerStatusMu and releases it
// before callers touch a status's lock; updateTrigger holds triggerStatusMu
// across the inner setTestTrigger call. Never acquire the status lock first
// and then triggerStatusMu.
type Service struct {
	// informers is set on lease acquisition (runWatcher) and nil-ed on
	// release. informersMu guards the pointer so tests and external callers
	// can safely observe it; runInformers reads it from the same goroutine
	// that writes it and therefore doesn't need the lock.
	informers                     *k8sInformers
	informersMu                   sync.RWMutex
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
	// triggerStatus is mutated by informer event handlers (addTrigger /
	// updateTrigger / removeTrigger) and read by the matcher and scraper.
	// triggerStatusMu guards the map itself; concurrency on individual
	// *triggerStatus values is handled by their own embedded sync.RWMutex.
	triggerStatus                 map[statusKey]*triggerStatus
	triggerStatusMu               sync.RWMutex
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
	coordinator                   *leader.Coordinator
}

type Option func(*Service)

func NewService(
	agentName string,
	clientset kubernetes.Interface,
	testKubeClientset testkubeclientsetv1.Interface,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testTriggersClient testtriggerclient.TestTriggerClient,
	leaseBackend leasebackend.Repository,
	logger *zap.SugaredLogger,
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

	coordinatorLogger := logger.With("component", "trigger-service", "identifier", s.identifier)
	s.coordinator = leader.New(leaseBackend, s.identifier, s.clusterID, coordinatorLogger)

	s.coordinator.Register(leader.Task{
		Name: "trigger-watcher",
		Start: func(taskCtx context.Context) error {
			s.runWatcher(taskCtx)
			return nil
		},
	})

	s.coordinator.Register(leader.Task{
		Name: "trigger-scraper",
		Start: func(taskCtx context.Context) error {
			s.runExecutionScraper(taskCtx)
			return nil
		},
	})

	return s
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

func (s *Service) Run(ctx context.Context) {
	if s.coordinator == nil {
		<-ctx.Done()
		return
	}

	if err := s.coordinator.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		if s.logger != nil {
			s.logger.Errorw("trigger service: coordinator stopped unexpectedly", "error", err)
		}
	}
}

func (s *Service) addTrigger(t *testtriggersv1.TestTrigger) {
	key := newStatusKey(t.Namespace, t.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	s.triggerStatus[key] = newTriggerStatus(t)
}

func (s *Service) updateTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(target.Namespace, target.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	existing := s.triggerStatus[key]
	if existing == nil {
		s.triggerStatus[key] = newTriggerStatus(target)
		return
	}
	existing.setTestTrigger(target)
}

func (s *Service) removeTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(target.Namespace, target.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	delete(s.triggerStatus, key)
}

type triggerStatusEntry struct {
	key    statusKey
	status *triggerStatus
}

// snapshotStatuses returns a shallow copy so match/scraper can iterate without
// blocking high-volume informer events on the map lock.
func (s *Service) snapshotStatuses() []triggerStatusEntry {
	s.triggerStatusMu.RLock()
	defer s.triggerStatusMu.RUnlock()
	out := make([]triggerStatusEntry, 0, len(s.triggerStatus))
	for k, v := range s.triggerStatus {
		out = append(out, triggerStatusEntry{key: k, status: v})
	}
	return out
}
