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
	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"

	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/coordination/leader"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/http"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/workflowtriggerclient"
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
// per-status RWMutex. snapshotStatuses takes triggerStatusMu, copies each
// status's trigger pointer into the returned entry, and releases the lock
// before callers touch a status's own per-status lock. updateTrigger writes
// the trigger pointer directly while holding triggerStatusMu.Lock().
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
	workflowTriggersClient        workflowtriggerclient.WorkflowTriggerClient
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
	dynamicClient                 dynamic.Interface
	dynamicManager                *dynamicInformerManager
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

// WithWorkflowTriggersClient injects the client used for control-plane-backed
// WorkflowTrigger v2 polling. When set alongside testTriggerControlPlane=true
// the dynamic informer is bypassed and this client is polled instead.
func WithWorkflowTriggersClient(client workflowtriggerclient.WorkflowTriggerClient) Option {
	return func(s *Service) {
		s.workflowTriggersClient = client
	}
}

func WithEventLabels(eventLabels map[string]string) Option {
	return func(s *Service) {
		s.eventLabels = eventLabels
	}
}

func WithDynamicClient(client dynamic.Interface) Option {
	return func(s *Service) {
		s.dynamicClient = client
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

func (s *Service) addTrigger(ctx context.Context, t *testtriggersv1.TestTrigger) {
	key := newStatusKey(triggerSourceV1, t.Namespace, t.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	if _, exists := s.triggerStatus[key]; exists {
		// Informer resyncs replay AddFunc for every existing object. Skip to
		// avoid re-registering the same trigger's dynamic informer reference.
		return
	}
	s.triggerStatus[key] = newTriggerStatusFromV1(t)
	s.ensureDynamicInformerForTrigger(ctx, t, key)
}

func (s *Service) updateTrigger(ctx context.Context, target *testtriggersv1.TestTrigger) {
	key := newStatusKey(triggerSourceV1, target.Namespace, target.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	if s.triggerStatus[key] != nil {
		old := s.triggerStatus[key].trigger
		newInternal := convertV1ToInternal(target)

		// Only restart dynamic informer if the GVR actually changed
		gvrChanged := old != nil && (old.ResourceGroup != newInternal.ResourceGroup ||
			old.ResourceVersion != newInternal.ResourceVersion ||
			old.ResourceKind != newInternal.ResourceKind)

		if gvrChanged && old.ResourceKind != "" {
			s.releaseDynamicInformerForTrigger(&testtriggersv1.TestTrigger{
				Spec: testtriggersv1.TestTriggerSpec{ResourceRef: &testtriggersv1.TestTriggerResourceRef{
					Group: old.ResourceGroup, Version: old.ResourceVersion, Kind: old.ResourceKind,
				}},
			}, key)
		}

		s.triggerStatus[key].trigger = newInternal

		if gvrChanged {
			s.ensureDynamicInformerForTrigger(ctx, target, key)
		}
	} else {
		s.triggerStatus[key] = newTriggerStatusFromV1(target)
		s.ensureDynamicInformerForTrigger(ctx, target, key)
	}
}

func (s *Service) removeTrigger(target *testtriggersv1.TestTrigger) {
	key := newStatusKey(triggerSourceV1, target.Namespace, target.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	delete(s.triggerStatus, key)
	s.releaseDynamicInformerForTrigger(target, key)
}

// ensureDynamicInformerForTrigger starts a dynamic informer if the trigger
// uses resourceRef pointing to a custom resource. Built-in types already
// have typed informers and must NOT get a dynamic informer to avoid double-firing.
// The caller's ctx is propagated so in-flight probe/condition waits cancel
// when the trigger service loses its lease.
func (s *Service) ensureDynamicInformerForTrigger(ctx context.Context, t *testtriggersv1.TestTrigger, key statusKey) {
	if t.Spec.ResourceRef == nil {
		return
	}
	if s.dynamicManager == nil {
		s.logger.Warnf(
			"trigger service: TestTrigger %s/%s has resourceRef=%s but no dynamic client is configured; custom-resource events will be ignored",
			t.Namespace, t.Name, t.Spec.ResourceRef.Kind,
		)
		return
	}
	if isBuiltinResource(t.Spec.ResourceRef.Kind) {
		s.logger.Debugf("trigger service: skipping dynamic informer for built-in type %s", t.Spec.ResourceRef.Kind)
		return
	}

	gvr, err := resolveGVR(s.dynamicManager.mapper, t.Spec.ResourceRef.Group, t.Spec.ResourceRef.Version, t.Spec.ResourceRef.Kind)
	if err != nil {
		s.logger.Errorf("trigger service: failed to resolve GVR for %s: %v", t.Spec.ResourceRef.Kind, err)
		return
	}

	s.dynamicManager.ensureInformer(ctx, gvr, string(key), s.dynamicEventHandler(ctx, gvr))
}

func (s *Service) releaseDynamicInformerForTrigger(t *testtriggersv1.TestTrigger, key statusKey) {
	if t.Spec.ResourceRef == nil {
		return
	}
	s.releaseDynamicInformerByGVK(t.Spec.ResourceRef.Group, t.Spec.ResourceRef.Version, t.Spec.ResourceRef.Kind, key)
}

// releaseDynamicInformerByGVK resolves the GVR for the given kind and releases
// the dynamic informer reference held under the given key. Safe to call when
// the CRD has already been deleted — resolveGVR fails and we skip at Debug
// level because that's an expected cleanup ordering, not an operator-actionable
// condition.
func (s *Service) releaseDynamicInformerByGVK(group, version, kind string, key statusKey) {
	if s.dynamicManager == nil || isBuiltinResource(kind) {
		return
	}
	gvr, err := resolveGVR(s.dynamicManager.mapper, group, version, kind)
	if err != nil {
		s.logger.Debugf("trigger service: informer cleanup skipped for %s/%s/%s (key=%s): %v",
			group, version, kind, key, err)
		return
	}
	s.dynamicManager.releaseInformer(gvr, string(key))
}

func (s *Service) addWorkflowTrigger(ctx context.Context, t *workflowtriggersv1.WorkflowTrigger) {
	key := newStatusKey(triggerSourceV2, t.Namespace, t.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	if _, exists := s.triggerStatus[key]; exists {
		// Informer resyncs replay AddFunc. Skip to avoid re-registering dynamic informers.
		return
	}
	s.triggerStatus[key] = &triggerStatus{trigger: convertV2ToInternal(t)}
	s.ensureDynamicInformerForWorkflowTrigger(ctx, t, key)
}

func (s *Service) updateWorkflowTrigger(ctx context.Context, target *workflowtriggersv1.WorkflowTrigger) {
	key := newStatusKey(triggerSourceV2, target.Namespace, target.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	if s.triggerStatus[key] != nil {
		old := s.triggerStatus[key].trigger
		newInternal := convertV2ToInternal(target)

		gvrChanged := old != nil && (old.ResourceGroup != newInternal.ResourceGroup ||
			old.ResourceVersion != newInternal.ResourceVersion ||
			old.ResourceKind != newInternal.ResourceKind)

		if gvrChanged && old.ResourceKind != "" {
			s.releaseDynamicInformerByGVK(old.ResourceGroup, old.ResourceVersion, old.ResourceKind, key)
		}

		s.triggerStatus[key].trigger = newInternal

		if gvrChanged {
			s.ensureDynamicInformerForWorkflowTrigger(ctx, target, key)
		}
	} else {
		s.triggerStatus[key] = &triggerStatus{trigger: convertV2ToInternal(target)}
		s.ensureDynamicInformerForWorkflowTrigger(ctx, target, key)
	}
}

func (s *Service) removeWorkflowTrigger(target *workflowtriggersv1.WorkflowTrigger) {
	key := newStatusKey(triggerSourceV2, target.Namespace, target.Name)
	s.triggerStatusMu.Lock()
	defer s.triggerStatusMu.Unlock()
	delete(s.triggerStatus, key)
	if target.Spec.Watch == nil {
		return
	}
	s.releaseDynamicInformerByGVK(target.Spec.Watch.Resource.Group, target.Spec.Watch.Resource.Version, target.Spec.Watch.Resource.Kind, key)
}

// ensureDynamicInformerForWorkflowTrigger starts a dynamic informer for the
// custom resource kind the trigger watches. Built-in types already have typed
// informers in the trigger service; skipping them here prevents double-firing.
// The caller's ctx is propagated so in-flight probe/condition waits cancel
// when the trigger service loses its lease.
func (s *Service) ensureDynamicInformerForWorkflowTrigger(ctx context.Context, t *workflowtriggersv1.WorkflowTrigger, key statusKey) {
	if t.Spec.Watch == nil || t.Spec.Watch.Resource.Kind == "" {
		return
	}
	if s.dynamicManager == nil {
		s.logger.Warnf(
			"trigger service: WorkflowTrigger %s/%s watches %s but no dynamic client is configured; events will be ignored",
			t.Namespace, t.Name, t.Spec.Watch.Resource.Kind,
		)
		return
	}
	if isBuiltinResource(t.Spec.Watch.Resource.Kind) {
		s.logger.Debugf("trigger service: skipping dynamic informer for built-in type %s", t.Spec.Watch.Resource.Kind)
		return
	}
	gvr, err := resolveGVR(s.dynamicManager.mapper, t.Spec.Watch.Resource.Group, t.Spec.Watch.Resource.Version, t.Spec.Watch.Resource.Kind)
	if err != nil {
		s.logger.Errorf("trigger service: failed to resolve GVR for %s: %v", t.Spec.Watch.Resource.Kind, err)
		return
	}
	s.dynamicManager.ensureInformer(ctx, gvr, string(key), s.dynamicEventHandler(ctx, gvr))
}

type triggerStatusEntry struct {
	key    statusKey
	status *triggerStatus
	// trigger is captured under triggerStatusMu so matcher reads don't race
	// with updateTrigger's direct assignment to status.trigger.
	trigger *internalTrigger
}

// snapshotStatuses returns a shallow copy so match/scraper can iterate without
// blocking high-volume informer events on the map lock.
func (s *Service) snapshotStatuses() []triggerStatusEntry {
	s.triggerStatusMu.RLock()
	defer s.triggerStatusMu.RUnlock()
	out := make([]triggerStatusEntry, 0, len(s.triggerStatus))
	for k, v := range s.triggerStatus {
		out = append(out, triggerStatusEntry{key: k, status: v, trigger: v.trigger})
	}
	return out
}
