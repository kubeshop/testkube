package triggers

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/mongo"

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

func (s *Service) checkForRunningTestWorkflowExecutions(ctx context.Context, status *triggerStatus) {
	testWorkflowExecutionIDs := status.getTestWorkflowExecutionIDs()

	for _, id := range testWorkflowExecutionIDs {
		// Pro edition only (tcl protected code)
		execution, err := s.testWorkflowResultsRepository.Get(ctx, id)
		if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warnf("trigger service: execution scraper component: no testworkflow execution found for id %s", id)
			status.removeTestWorkflowExecutionID(id)
			continue
		} else if err != nil {
			s.logger.Errorf("trigger service: execution scraper component: error fetching testworkflow execution result: %v", err)
			continue
		}
		if execution.Result != nil && (!execution.Result.IsRunning() && !execution.Result.IsQueued() && !execution.Result.IsPaused()) {
			s.logger.Debugf("trigger service: execution scraper component: testworkflow execution %s is finished", id)
			status.removeTestWorkflowExecutionID(id)
		}
	}
}

func (s *Service) abortExecutions(ctx context.Context, testTriggerName string, status *triggerStatus) {
	s.logger.Debugf("trigger service: abort executions")
	s.abortRunningTestWorkflowExecutions(ctx, status)
	if !status.hasActiveTests() {
		s.logger.Debugf("marking status as finished for testtrigger %s", testTriggerName)
		status.done()
	}
}

func (s *Service) abortRunningTestWorkflowExecutions(ctx context.Context, status *triggerStatus) {
	testWorkflowExecutionIDs := status.getTestWorkflowExecutionIDs()

	for _, id := range testWorkflowExecutionIDs {
		// Pro edition only (tcl protected code)
		execution, err := s.testWorkflowResultsRepository.Get(ctx, id)
		if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, pgx.ErrNoRows) {
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
