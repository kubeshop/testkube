package cronjob

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/kubeshop/testkube-operator/pkg/client/common"
)

// ReconcileTests is watching for test change and schedule test cron jobs
func (s *Scheduler) ReconcileTests(ctx context.Context) error {
	includeInitialData := true
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			watcher := s.testRESTClient.WatchUpdates(ctx, s.getEnvironmentId(), includeInitialData)
			for obj := range watcher.Channel() {
				if obj.Resource == nil {
					continue
				}

				var err error
				switch obj.Type {
				case common.EventTypeCreate:
					err = s.addTestCronJob(ctx, obj.Resource.Name, obj.Resource.Spec.Schedule)
				case common.EventTypeUpdate:
					err = s.changeTestCronJob(ctx, obj.Resource.Name, obj.Resource.Spec.Schedule)
				case common.EventTypeDelete:
					s.removeTestCronJob(obj.Resource.Name)
				}

				if err == nil {
					s.logger.Infow("cron job scheduler: reconciler component: scheduled Test to cron jobs", "name", obj.Resource.Name)
				} else {
					s.logger.Errorw("cron job scheduler: reconciler component: failed to watch Tests", "error", err)
				}
			}

			if watcher.Err() != nil {
				s.logger.Errorw("cron job scheduler: reconciler component: failed to watch Tests", "error", watcher.Err())
			} else {
				includeInitialData = false
			}

			time.Sleep(watcherDelay)
		}
	}
}

func (s *Scheduler) addTestCronJob(ctx context.Context, testName, schedule string) error {
	if schedule == "" {
		return nil
	}

	if _, ok := s.tests[testName]; !ok {
		entryID, err := s.cronService.AddJob(schedule,
			cron.FuncJob(func() { s.executeTest(ctx, testName, schedule) }))
		if err != nil {
			return err
		}

		s.tests[testName] = scheduleEntry{Schedule: schedule, EntryID: entryID}
	}

	return nil
}

func (s *Scheduler) changeTestCronJob(ctx context.Context, testName, schedule string) error {
	if schedule != "" {
		skip := false
		if scheduleEntry, ok := s.tests[testName]; ok {
			if scheduleEntry.Schedule != schedule {
				s.removeTestCronJob(testName)
			} else {
				skip = true
			}
		}

		if !skip {
			if err := s.addTestCronJob(ctx, testName, schedule); err != nil {
				return err
			}
		}
	} else {
		s.removeTestCronJob(testName)
	}

	return nil
}

func (s *Scheduler) removeTestCronJob(testName string) {
	if scheduleEntry, ok := s.tests[testName]; ok {
		s.cronService.Remove(scheduleEntry.EntryID)
		delete(s.tests, testName)
	}
}
