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

//go:generate go tool mockgen -destination=./mock_service.go -package=cronjob "github.com/kubeshop/testkube/pkg/cronjob" ScheduleManager

// Workflow uniquely identifies a workflow in a multi-tenanted installation.
type Workflow struct {
	// Name of the workflow, this is expected to be unique within an environment.
	Name string
	// Identifier of the Environment, the named workflow is expected to be under
	// this environment.
	EnvId string
	// Identifier of the Organisation, the identified environment is expected to
	// be under this organisation. This value may be empty when the watcher is
	// operating in a single tenant mode.
	OrgId string
}

// ScheduleManager is responsible for managing the schedule and executing workflow on their schedule.
type ScheduleManager interface {
	// ReplaceWorkflowSchedules does pretty much what it says.
	// A manager is expected to replace all existing schedules for the passed workflow with only the schedules
	// passed in the function call.
	// In the event that the list of schedules is empty then this indicates that no schedules should exist
	// for this workflow.
	ReplaceWorkflowSchedules(ctx context.Context, workflow Workflow, schedules []testkube.TestWorkflowCronJobConfig) error
}

// Config of a workflow schedule
type Config struct {
	// Workflow is the unique identifier of a workflow that the schedule relates to.
	Workflow Workflow
	// Schedules are all the execution schedules for this workflow.
	// If it is empty then no schedules should exist for this workflow.
	Schedules []testkube.TestWorkflowCronJobConfig
}

type Watcher func(context.Context, chan<- Config)

type Service struct {
	logger   *zap.SugaredLogger
	cron     ScheduleManager
	watchers []Watcher
}

func NewService(logger *zap.SugaredLogger, mgr ScheduleManager, watchers ...Watcher) Service {
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
	s.logger.Infow("cronjob service starting",
		"watchers", len(s.watchers),
	)
	cronChan := make(chan Config)
	go func() {
		for {
			select {
			case <-ctx.Done():
				s.logger.Infow("cronjob service stopping (context done)")
				return
			case config := <-cronChan:
				s.logger.Infow("cronjob service received schedule config",
					"workflow", config.Workflow.Name,
					"org", config.Workflow.OrgId,
					"env", config.Workflow.EnvId,
					"cron_count", len(config.Schedules),
				)
				if err := s.cron.ReplaceWorkflowSchedules(ctx, config.Workflow, config.Schedules); err != nil {
					s.logger.Errorw("error modifying workflow execution schedule",
						"workflow name", config.Workflow.Name,
						"cron count", len(config.Schedules),
						"err", err)
				}
			}
		}
	}()
	for _, watcher := range s.watchers {
		s.logger.Infow("starting cronjob watcher")

		go watcher(ctx, cronChan)
	}
	// Run until context is complete.
	<-ctx.Done()
	close(cronChan)
}
