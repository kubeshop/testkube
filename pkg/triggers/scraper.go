package triggers

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func (s *Service) runExecutionScraper(ctx context.Context) {
	ticker := time.NewTicker(s.scraperInterval)
	s.logger.Debugf("trigger service: starting execution scraper")

	for {
		select {
		case <-ctx.Done():
			s.logger.Infof("trigger service: stopping scraper component")
			return
		case <-ticker.C:
			s.logger.Debugf("trigger service: execution scraper component: starting new ticker iteration")
			for triggerName, status := range s.triggerStatus {
				if status.hasActiveTests() {
					if s.deprecatedSystem != nil {
						s.checkForRunningTestExecutions(ctx, status)
						s.checkForRunningTestSuiteExecutions(ctx, status)
					}
					s.checkForRunningTestWorkflowExecutions(ctx, status)
					if !status.hasActiveTests() {
						s.logger.Debugf("marking status as finished for testtrigger %s", triggerName)
						status.done()
					}
				}
			}
		}
	}
}

func (s *Service) checkForRunningTestExecutions(ctx context.Context, status *triggerStatus) {
	testExecutionIDs := status.getExecutionIDs()

	for _, id := range testExecutionIDs {
		execution, err := s.deprecatedSystem.Repositories.TestResults().Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.logger.Warnf("trigger service: execution scraper component: no test execution found for id %s", id)
			status.removeExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching test execution result: %v", err)
			continue
		}
		if !execution.IsRunning() && !execution.IsQueued() {
			s.logger.Debugf("trigger service: execution scraper component: test execution %s is finished", id)
			status.removeExecutionID(id)
		}
	}
}

func (s *Service) checkForRunningTestSuiteExecutions(ctx context.Context, status *triggerStatus) {
	testSuiteExecutionIDs := status.getTestSuiteExecutionIDs()

	for _, id := range testSuiteExecutionIDs {
		execution, err := s.deprecatedSystem.Repositories.TestSuiteResults().Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.logger.Warnf("trigger service: execution scraper component: no testsuite execution found for id %s", id)
			status.removeTestSuiteExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching testsuite execution result: %v", err)
			continue
		}
		if !execution.IsRunning() && !execution.IsQueued() {
			s.logger.Debugf("trigger service: execution scraper component: testsuite execution %s is finished", id)
			status.removeTestSuiteExecutionID(id)
		}
	}
}

func (s *Service) checkForRunningTestWorkflowExecutions(ctx context.Context, status *triggerStatus) {
	testWorkflowExecutionIDs := status.getTestWorkflowExecutionIDs()

	for _, id := range testWorkflowExecutionIDs {
		// Pro edition only (tcl protected code)
		execution, err := s.testWorkflowResultsRepository.Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.logger.Warnf("trigger service: execution scraper component: no testworkflow execution found for id %s", id)
			status.removeTestWorkflowExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching testworkflow execution result: %v", err)
			continue
		}
		if execution.Result != nil && !(execution.Result.IsRunning() || execution.Result.IsQueued() || execution.Result.IsPaused()) {
			s.logger.Debugf("trigger service: execution scraper component: testworkflow execution %s is finished", id)
			status.removeTestWorkflowExecutionID(id)
		}
	}
}

func (s *Service) abortExecutions(ctx context.Context, testTriggerName string, status *triggerStatus) {
	s.logger.Debugf("trigger service: abort executions")
	if s.deprecatedSystem != nil {
		s.abortRunningTestExecutions(ctx, status)
		s.abortRunningTestSuiteExecutions(ctx, status)
	}
	s.abortRunningTestWorkflowExecutions(ctx, status)
	if !status.hasActiveTests() {
		s.logger.Debugf("marking status as finished for testtrigger %s", testTriggerName)
		status.done()
	}
}

func (s *Service) abortRunningTestExecutions(ctx context.Context, status *triggerStatus) {
	testExecutionIDs := status.getExecutionIDs()

	for _, id := range testExecutionIDs {
		execution, err := s.deprecatedSystem.Repositories.TestResults().Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.logger.Warnf("trigger service: execution scraper component: no test execution found for id %s", id)
			status.removeExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching test execution result: %v", err)
			continue
		}
		if execution.IsRunning() || execution.IsQueued() {
			res, err := s.deprecatedSystem.JobExecutor.Abort(ctx, &execution)
			if err != nil {
				s.logger.Errorf("trigger service: execution scraper component: error aborting test execution: %v", err)
				continue
			}
			s.metrics.IncAbortTest(execution.TestType, res.IsFailed())

			s.logger.Debugf("trigger service: execution scraper component: test execution %s is aborted", id)
			status.removeExecutionID(id)
		}
	}
}

func (s *Service) abortRunningTestSuiteExecutions(ctx context.Context, status *triggerStatus) {
	testSuiteExecutionIDs := status.getTestSuiteExecutionIDs()

	for _, id := range testSuiteExecutionIDs {
		execution, err := s.deprecatedSystem.Repositories.TestSuiteResults().Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.logger.Warnf("trigger service: execution scraper component: no testsuite execution found for id %s", id)
			status.removeTestSuiteExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching testsuite execution result: %v", err)
			continue
		}
		if execution.IsRunning() || execution.IsQueued() {
			err := s.eventsBus.PublishTopic(bus.InternalPublishTopic, testkube.NewEventEndTestSuiteAborted(&execution))
			if err != nil {
				s.logger.Errorf("trigger service: execution scraper component: error aborting test suite execution: %v", err)
				continue
			}
			s.metrics.IncAbortTestSuite()

			s.logger.Debugf("trigger service: execution scraper component: testsuite execution %s is aborted", id)
			status.removeTestSuiteExecutionID(id)
		}
	}
}

func (s *Service) abortRunningTestWorkflowExecutions(ctx context.Context, status *triggerStatus) {
	testWorkflowExecutionIDs := status.getTestWorkflowExecutionIDs()

	for _, id := range testWorkflowExecutionIDs {
		// Pro edition only (tcl protected code)
		execution, err := s.testWorkflowResultsRepository.Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.logger.Warnf("trigger service: execution scraper component: no testworkflow execution found for id %s", id)
			status.removeTestWorkflowExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching testworkflow execution result: %v", err)
			continue
		}
		if execution.Result != nil && (execution.Result.IsRunning() || execution.Result.IsQueued() || execution.Result.IsPaused()) {
			// Pro edition only (tcl protected code)
			// Obtain the controller
			err = s.executionWorkerClient.Abort(ctx, execution.Id, executionworkertypes.DestroyOptions{
				Namespace: s.testkubeNamespace,
			})
			if err != nil {
				s.logger.Errorf("trigger service: execution scraper component: error aborting test workflow execution: %v", err)
				continue
			}
			s.metrics.IncAbortTestWorkflow()

			s.logger.Debugf("trigger service: execution scraper component: testworkflow execution %s is aborted", id)
			status.removeTestWorkflowExecutionID(id)
		}
	}
}
