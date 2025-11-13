// Package cronjob contains a pluggable service for managing
// TestWorkflow execution schedules, sourcing schedules from
// a number of different possible sources, for example from
// both TestWorkflow and TestWorkflowTemplate resources in
// a Kubernetes cluster.
package cronjob

import (
	"context"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:generate go tool mockgen -destination=./mock_service.go -package=cronjob "github.com/kubeshop/testkube/pkg/cronjob" CronManager

// CronManager is responsible for managing the schedule and executing workflow on their schedule.
type CronManager interface {
	// CreateOrUpdate a workflow execution schedule.
	CreateOrUpdate(context.Context, string, testkube.TestWorkflowCronJobConfig) error
	// Delete a workflow execution schedule.
	Delete(context.Context, string, testkube.TestWorkflowCronJobConfig) error
}

// Config of a workflow schedule
type Config struct {
	// WorkflowName is the name of the workflow that should be executed on this schedule.
	WorkflowName string
	// CronJob is the schedule configuration itself.
	CronJob testkube.TestWorkflowCronJobConfig
	// Remove, when set to true will cause the associated schedule to be removed from the manager.
	Remove bool
}

type Watcher func(context.Context, chan<- Config)

type Service struct {
	logger   *zap.SugaredLogger
	cron     CronManager
	watchers []Watcher
}

func NewService(logger *zap.SugaredLogger, mgr CronManager, watchers ...Watcher) Service {
	return Service{
		logger:   logger,
		cron:     mgr,
		watchers: watchers,
	}
}

// Run starts the cronjob Service, causing it to wait for
// updates of configured listeners and use those to generate
// cron scheduled test executions.
func (s Service) Run(ctx context.Context) {
	cronChan := make(chan Config)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case config := <-cronChan:
				var err error
				switch {
				case config.Remove:
					err = s.cron.Delete(ctx, config.WorkflowName, config.CronJob)
				default:
					err = s.cron.CreateOrUpdate(ctx, config.WorkflowName, config.CronJob)
				}
				if err != nil {
					s.logger.Errorf("error modifying workflow execution schedule",
						"workflow name", config.WorkflowName,
						"delete action", config.Remove,
						"cron", config.CronJob.Cron,
						"err", err)
				}
			}
		}
	}()
	for _, watcher := range s.watchers {
		go watcher(ctx, cronChan)
	}
	// Run until context is complete.
	<-ctx.Done()
	close(cronChan)
}
