// Package robfig provides a cronjob.Manager implementation based on the
// robfig/cron Go based cron scheduler.
package robfig

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cronjob"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
	cronjobtcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cronjob"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

type Executor interface {
	Execute(ctx context.Context, req *cloud.ScheduleRequest) ([]testkube.TestWorkflowExecution, error)
}

type Manager struct {
	proModeEnabled bool
	logger         *zap.SugaredLogger
	cron           *cron.Cron
	cronEntries    map[string]map[string]cron.EntryID
	executor       Executor
}

func New(logger *zap.SugaredLogger, executor Executor, proModeEnabled bool) Manager {
	return Manager{
		proModeEnabled: proModeEnabled,
		logger:         logger,
		cron:           cron.New(),
		cronEntries:    make(map[string]map[string]cron.EntryID),
		executor:       executor,
	}
}

// Start the cron manager in its own goroutine, or no-op if already started.
func (m Manager) Start() {
	m.logger.Infow("cron manager starting")
	m.cron.Start()
	m.logger.Infow("cron manager started")
}

// Stop stops the cron manager if it is running; otherwise it does nothing.
func (m Manager) Stop() {
	m.logger.Infow("cron manager stopping")
	m.cron.Stop()
	m.logger.Infow("cron manager stopped")
}

func cronSpec(config testkube.TestWorkflowCronJobConfig) string {
	spec := config.Cron
	if config.Timezone != nil {
		spec = fmt.Sprintf("CRON_TZ=%s %s", config.Timezone.Value, config.Cron)
	}
	return spec
}

func (m Manager) ReplaceWorkflowSchedules(ctx context.Context, workflow cronjob.Workflow, configs []testkube.TestWorkflowCronJobConfig) error {
	log := m.logger.With("workflow", workflow.Name)
	// Delete all existing schedules for this workflow.
	// This is because we may not know when a schedule is removed from
	// an object so we must recreate the entire schedule from scratch
	// each time there is a change.
	if _, exists := m.cronEntries[workflow.Name]; exists {
		log.Infow("removing existing schedules", "existing_entries", m.cronEntries[workflow.Name])
		for _, entryId := range m.cronEntries[workflow.Name] {
			log.Debugw("removing schedule entry", "entry_id", entryId)
			m.cron.Remove(entryId)
		}
		delete(m.cronEntries, workflow.Name)
	}
	m.cronEntries[workflow.Name] = make(map[string]cron.EntryID)

	for _, config := range configs {
		spec := cronSpec(config)

		if config.Timezone != nil {
			log.Infow("adding schedule",
				"spec", spec,
				"cron", config.Cron,
				"timezone", config.Timezone.Value,
			)
		} else {
			log.Infow("adding schedule",
				"spec", spec,
				"cron", config.Cron,
			)
		}
		entryId, err := m.cron.AddJob(spec, m.testWorkflowExecuteJob(ctx, workflow.Name, spec, config))
		if err != nil {
			m.logger.Errorw("Error adding cron for workflow, continuing processing",
				"cron", spec,
				"workflow", workflow,
				"err", err)
			continue
		}
		m.cronEntries[workflow.Name][spec] = entryId
		entry := m.cron.Entry(entryId)

		log.Infow("schedule registered",
			"entry_id", entryId,
			"spec", spec,
			"next_run", entry.Next.Format(time.RFC3339),
			"prev_run", entry.Prev.Format(time.RFC3339),
		)
	}
	log.Infow("ReplaceWorkflowSchedules finished")
	return nil
}

func (m Manager) testWorkflowExecuteJob(ctx context.Context, workflow, cronSpec string, config testkube.TestWorkflowCronJobConfig) cron.FuncJob {
	return cron.FuncJob(func() {
		var targets []*cloud.ExecutionTarget
		if config.Target != nil {
			targets = commonmapper.MapAllTargetsApiToGrpc([]testkube.ExecutionTarget{*config.Target})
		}

		request := &cloud.ScheduleRequest{
			Executions: []*cloud.ScheduleExecution{{
				Selector: &cloud.ScheduleResourceSelector{Name: workflow},
				Config:   config.Config,
				Targets:  targets,
			}},
		}

		// Pro edition only (tcl protected code)
		if m.proModeEnabled {
			request.RunningContext, _ = testworkflowexecutor.GetNewRunningContext(cronjobtcl.GetRunningContext(cronSpec), nil)
		}

		log := m.logger.With(
			"workflow", workflow,
			"schedule", cronSpec,
		)
		log.Info("executing scheduled workflow")

		results, err := m.executor.Execute(ctx, request)
		if err != nil {
			log.Errorw("unable to execute scheduled workflow",
				"error", err)
			return
		}

		executionID := ""
		if len(results) != 0 {
			executionID = results[0].Id
		}

		log.Debugw("started scheduled workflow execution",
			"execution id", executionID,
		)
	})
}
