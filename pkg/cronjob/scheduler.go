package cronjob

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	testsclientv3 "github.com/kubeshop/testkube/pkg/operator/client/tests/v3"
	testsuitesclientv3 "github.com/kubeshop/testkube/pkg/operator/client/testsuites/v3"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

type scheduleEntry struct {
	Schedule string
	EntryID  cron.EntryID
}

const (
	watcherDelay = 200 * time.Millisecond
)

//go:generate mockgen -destination=./mock_scheduler.go -package=cronjob "github.com/kubeshop/testkube/pkg/cronjob" Interface
type Interface interface {
	Reconcile(ctx context.Context)
	ReconcileTestWorkflows(ctx context.Context) error
	ReconcileTestWorkflowTemplates(ctx context.Context) error
	ReconcileTests(ctx context.Context) error
	ReconcileTestSuites(ctx context.Context) error
}

// Scheduler provide methods to schedule cron jobs
type Scheduler struct {
	testWorkflowClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient
	testWorkflowExecutor       testworkflowexecutor.TestWorkflowExecutor
	testClient                 testsclientv3.Interface
	testSuiteClient            testsuitesclientv3.Interface
	testRESTClient             testsclientv3.RESTInterface
	testSuiteRESTClient        testsuitesclientv3.RESTInterface
	executeTestFn              workerpool.ExecuteFn[testkube.Test, testkube.ExecutionRequest, testkube.Execution]
	executeTestSuiteFn         workerpool.ExecuteFn[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution]
	logger                     *zap.SugaredLogger
	proContext                 *intconfig.ProContext
	cronService                *cron.Cron
	testWorklows               map[string]map[string]cron.EntryID
	tests                      map[string]scheduleEntry
	testSuites                 map[string]scheduleEntry
	lock                       sync.RWMutex
}

// New is a method to create new cron job scheduler
func New(testWorkflowClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	logger *zap.SugaredLogger,
	opts ...Option) *Scheduler {
	s := &Scheduler{
		testWorkflowClient:         testWorkflowClient,
		testWorkflowTemplateClient: testWorkflowTemplateClient,
		testWorkflowExecutor:       testWorkflowExecutor,
		logger:                     logger,
		cronService:                cron.New(),
		testWorklows:               make(map[string]map[string]cron.EntryID),
		tests:                      make(map[string]scheduleEntry),
		testSuites:                 make(map[string]scheduleEntry),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Option func(*Scheduler)

func WithTestClient(testClient testsclientv3.Interface) Option {
	return func(s *Scheduler) {
		s.testClient = testClient
	}
}

func WithTestSuiteClient(testSuiteClient testsuitesclientv3.Interface) Option {
	return func(s *Scheduler) {
		s.testSuiteClient = testSuiteClient
	}
}

func WithTestRESTClient(testRESTClient testsclientv3.RESTInterface) Option {
	return func(s *Scheduler) {
		s.testRESTClient = testRESTClient
	}
}

func WithTestSuiteRESTClient(testSuiteRESTClient testsuitesclientv3.RESTInterface) Option {
	return func(s *Scheduler) {
		s.testSuiteRESTClient = testSuiteRESTClient
	}
}

func WithExecuteTestFn(executeTestFn workerpool.ExecuteFn[testkube.Test, testkube.ExecutionRequest, testkube.Execution]) Option {
	return func(s *Scheduler) {
		s.executeTestFn = executeTestFn
	}
}

func WithExecuteTestSuiteFn(executeTestSuiteFn workerpool.ExecuteFn[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution]) Option {
	return func(s *Scheduler) {
		s.executeTestSuiteFn = executeTestSuiteFn
	}
}

func WithProContext(proContext *intconfig.ProContext) Option {
	return func(s *Scheduler) {
		s.proContext = proContext
	}
}

// Reconcile is reconciling cron jobs
func (s *Scheduler) Reconcile(ctx context.Context) {
	s.cronService.Start()
	defer s.cronService.Stop()

	var wg sync.WaitGroup

	s.logger.Infow("cron job scheduler: reconciler component: starting reconciler")

	wg.Add(2)
	go func() {
		defer wg.Done()

		if err := s.ReconcileTestWorkflows(ctx); err != nil {
			s.logger.Errorw("cron job scheduler: reconciler component: failed to reconcile TestWorkflows", "error", err)
		}
	}()

	go func() {
		defer wg.Done()

		if err := s.ReconcileTestWorkflowTemplates(ctx); err != nil {
			s.logger.Errorw("cron job scheduler: reconciler component: failed to reconcile TestWorkflowTemplates", "error", err)
		}
	}()

	if s.testClient != nil && s.testRESTClient != nil && s.executeTestFn != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := s.ReconcileTests(ctx); err != nil {
				s.logger.Errorw("cron job scheduler: reconciler component: failed to reconcile Tests", "error", err)
			}
		}()

	}

	if s.testSuiteClient != nil && s.testSuiteRESTClient != nil && s.executeTestSuiteFn != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := s.ReconcileTestSuites(ctx); err != nil {
				s.logger.Errorw("cron job scheduler: reconciler component: failed to reconcile TestSuites", "error", err)
			}
		}()

	}

	wg.Wait()
}

func (s *Scheduler) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}

	return ""
}
