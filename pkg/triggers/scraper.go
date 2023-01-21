package triggers

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
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
					s.logger.Debugf("triggerStatus: %+v", *status)
					s.checkForRunningTestExecutions(ctx, status)
					s.checkForRunningTestSuiteExecutions(ctx, status)
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
	for _, id := range status.testExecutionIDs {
		execution, err := s.resultRepository.Get(ctx, id)
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
	for _, id := range status.testSuiteExecutionIDs {
		execution, err := s.testResultRepository.Get(ctx, id)
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
