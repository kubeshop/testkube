package triggers

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

func (s *Service) runExecutionScraper(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	s.l.Debugf("trigger service: starting execution scraper")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.l.Debugf("trigger service: execution scraper component: starting new ticker iteration")
			for triggerName, status := range s.triggerStatus {
				s.l.Debugf("triggerStatus: %+v", *status)
				s.checkForRunningExecutions(ctx, status, triggerName)
			}
		}
	}
}

func (s *Service) checkForRunningExecutions(ctx context.Context, status *TriggerStatus, triggerNamespacedName StatusKey) {
	if status.ActiveTests {
		if len(status.ExecutionIDs) == 0 {
			s.l.Debugf("marking status as finished for testtrigger %s/%s", triggerNamespacedName)
			status.Finish()
			return
		}
		for _, id := range status.ExecutionIDs {
			execution, err := s.tk.ExecutionResults.Get(ctx, id)
			if err == mongo.ErrNoDocuments {
				s.l.Warnf("trigger service: execution scraper component: no execution found for id %s", id)
				status.RemoveExecutionID(id)
				continue
			} else if err != nil {
				s.l.Errorf("trigger service: execution scraper component: error fetching execution result: %v", err)
				continue
			}
			if !execution.IsRunning() {
				s.l.Debugf("trigger service: execution scraper component: execution %s is finished", id)
				status.RemoveExecutionID(id)
			}
		}
	}
}

func (s *Service) checkForRunningTestSuiteExecutions(ctx context.Context, status *TriggerStatus, triggerNamespacedName StatusKey) {
	if status.ActiveTests {
		if len(status.TestSuiteExecutionIDs) == 0 {
			s.l.Debugf("marking status as finished for testtrigger %s", triggerNamespacedName)
			status.Finish()
			return
		}
		for _, id := range status.TestSuiteExecutionIDs {
			execution, err := s.tk.TestExecutionResults.Get(ctx, id)
			if err == mongo.ErrNoDocuments {
				s.l.Warnf("trigger service: execution scraper component: no testsuite execution found for id %s", id)
				status.RemoveExecutionID(id)
				continue
			} else if err != nil {
				s.l.Errorf("trigger service: execution scraper component: error fetching testsuite execution result: %v", err)
				continue
			}
			if !execution.IsRunning() {
				s.l.Debugf("trigger service: execution scraper component: testsuite execution %s is finished", id)
				status.RemoveExecutionID(id)
			}
		}
	}
}
