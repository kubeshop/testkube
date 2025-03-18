package cronjob

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/kubeshop/testkube-operator/pkg/client/common"
)

// ReconcileTestSuites is watching for testsuite change and schedule testsuite cron jobs
func (s *Scheduler) ReconcileTestSuites(ctx context.Context) error {
	includeInitialData := true
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			watcher := s.testSuiteRESTClient.WatchUpdates(ctx, s.getEnvironmentId(), includeInitialData)
			for obj := range watcher.Channel() {
				if obj.Resource == nil {
					continue
				}

				var err error
				switch obj.Type {
				case common.EventTypeCreate:
					err = s.addTestSuiteCronJob(ctx, obj.Resource.Name, obj.Resource.Spec.Schedule)
				case common.EventTypeUpdate:
					err = s.changeTestSuiteCronJob(ctx, obj.Resource.Name, obj.Resource.Spec.Schedule)
				case common.EventTypeDelete:
					s.removeTestSuiteCronJob(obj.Resource.Name)
				}

				if err == nil {
					s.logger.Infow("cron job scheduler: reconciler component: scheduled TestSuite to cron jobs", "name", obj.Resource.Name)
				} else {
					s.logger.Errorw("cron job scheduler: reconciler component: failed to watch TestSuites", "error", err)
				}
			}

			if watcher.Err() != nil {
				s.logger.Errorw("cron job scheduler: reconciler component: failed to watch TestSuites", "error", watcher.Err())
			} else {
				includeInitialData = false
			}

			time.Sleep(watcherDelay)
		}
	}
}

func (s *Scheduler) addTestSuiteCronJob(ctx context.Context, testSuiteName, schedule string) error {
	if schedule == "" {
		return nil
	}

	if _, ok := s.testSuites[testSuiteName]; !ok {
		entryID, err := s.cronService.AddJob(schedule,
			cron.FuncJob(func() { s.executeTestSuite(ctx, testSuiteName, schedule) }))
		if err != nil {
			return err
		}

		s.testSuites[testSuiteName] = scheduleEntry{Schedule: schedule, EntryID: entryID}
	}

	return nil
}

func (s *Scheduler) changeTestSuiteCronJob(ctx context.Context, testSuiteName, schedule string) error {
	if schedule != "" {
		skip := false
		if scheduleEntry, ok := s.testSuites[testSuiteName]; ok {
			if scheduleEntry.Schedule != schedule {
				s.removeTestSuiteCronJob(testSuiteName)
			} else {
				skip = true
			}
		}

		if !skip {
			if err := s.addTestSuiteCronJob(ctx, testSuiteName, schedule); err != nil {
				return err
			}
		}
	} else {
		s.removeTestSuiteCronJob(testSuiteName)
	}

	return nil
}

func (s *Scheduler) removeTestSuiteCronJob(testSuiteName string) {
	if scheduleEntry, ok := s.testSuites[testSuiteName]; ok {
		s.cronService.Remove(scheduleEntry.EntryID)
		delete(s.testSuites, testSuiteName)
	}
}
