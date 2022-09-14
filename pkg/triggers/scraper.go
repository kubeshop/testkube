package triggers

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func (s *Service) runExecutionScraper(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	s.l.Debugf("trigger service: starting execution scraper")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.l.Debugf("trigger service: execution scraper component: starting new ticker iteration")
			for triggerName, status := range s.triggerStatus {
				if status.ActiveTests {
					if len(status.ExecutionIDs) == 0 && len(status.TestSuiteExecutionIDs) == 0 {
						s.l.Debugf("marking status as finished for testtrigger %s", triggerName)
						status.finish()
						return
					}
				}
				s.l.Debugf("triggerStatus: %+v", *status)
				s.checkForRunningExecutions(ctx, status)
				s.checkForRunningTestSuiteExecutions(ctx, status)
			}
		}
	}
}

func (s *Service) checkForRunningExecutions(ctx context.Context, status *triggerStatus) {
	for _, id := range status.ExecutionIDs {
		execution, err := s.tk.ExecutionResults.Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.l.Warnf("trigger service: execution scraper component: no execution found for id %s", id)
			status.removeExecutionID(id)
			continue
		} else if err != nil {
			s.l.Errorf("trigger service: execution scraper component: error fetching execution result: %v", err)
			continue
		}
		if !execution.IsRunning() {
			s.l.Debugf("trigger service: execution scraper component: execution %s is finished", id)
			status.removeExecutionID(id)
		}
	}
}

func (s *Service) checkForRunningTestSuiteExecutions(ctx context.Context, status *triggerStatus) {
	for _, id := range status.TestSuiteExecutionIDs {
		execution, err := s.tk.TestExecutionResults.Get(ctx, id)
		if err == mongo.ErrNoDocuments {
			s.l.Warnf("trigger service: execution scraper component: no testsuite execution found for id %s", id)
			status.removeExecutionID(id)
			continue
		} else if err != nil {
			s.l.Errorf("trigger service: execution scraper component: error fetching testsuite execution result: %v", err)
			continue
		}
		if !execution.IsRunning() {
			s.l.Debugf("trigger service: execution scraper component: testsuite execution %s is finished", id)
			status.removeExecutionID(id)
		}
	}
}
