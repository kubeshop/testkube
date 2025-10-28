package cronjob

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	intconfig "github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

const (
	// TestResourceURI is test resource uri for cron job call
	TestResourceURI = "tests"
	// TestSuiteResourceURI is test suite resource uri for cron job call
	TestSuiteResourceURI = "test-suites"
	// TestWorkflowResourceURI is test workflow resource uri for cron job call
	TestWorkflowResourceURI = "test-workflows"
)

const (
	watcherDelay = 200 * time.Millisecond
)

//go:generate mockgen -destination=./mock_scheduler.go -package=cronjob "github.com/kubeshop/testkube/pkg/cronjob" Interface
type Interface interface {
	Reconcile(ctx context.Context)
	ReconcileTestWorkflows(ctx context.Context) error
	ReconcileTestWorkflowTemplates(ctx context.Context) error
}

// Scheduler provide methods to schedule cron jobs
type Scheduler struct {
	testWorkflowClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient
	testWorkflowExecutor       testworkflowexecutor.TestWorkflowExecutor
	logger                     *zap.SugaredLogger
	proContext                 *intconfig.ProContext
	cronService                *cron.Cron
	testWorkflows              map[string]map[string]cron.EntryID
	lock                       sync.RWMutex
}

// New is a method to create new cron job scheduler
func New(testWorkflowClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	logger *zap.SugaredLogger,
	proContext *intconfig.ProContext) *Scheduler {
	return &Scheduler{
		testWorkflowClient:         testWorkflowClient,
		testWorkflowTemplateClient: testWorkflowTemplateClient,
		testWorkflowExecutor:       testWorkflowExecutor,
		logger:                     logger,
		cronService:                cron.New(),
		testWorkflows:              make(map[string]map[string]cron.EntryID),
		proContext:                 proContext,
	}
}

type Option func(*Scheduler)

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

	wg.Wait()
}

func (s *Scheduler) getEnvironmentId() string {
	if s.proContext != nil {
		return s.proContext.EnvID
	}

	return ""
}
