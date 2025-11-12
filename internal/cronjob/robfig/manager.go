// Package robfig provides a cronjob.Manager implementation based on the
// robfig/cron Go based cron scheduler.
package robfig

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
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
	m.cron.Start()
}

// Stop stops the cron manager if it is running; otherwise it does nothing.
func (m Manager) Stop() {
	m.cron.Stop()
}

func cronSpec(config testkube.TestWorkflowCronJobConfig) string {
	spec := config.Cron
	if config.Timezone != nil {
		spec = fmt.Sprintf("CRON_TZ=%s %s", config.Timezone.Value, config.Cron)
	}
	return spec
}

// CreateOrUpdate a schedule in the manager.
func (m Manager) CreateOrUpdate(ctx context.Context, workflow string, config testkube.TestWorkflowCronJobConfig) error {
	if _, ok := m.cronEntries[workflow]; !ok {
		m.cronEntries[workflow] = make(map[string]cron.EntryID, 0)
	}

	spec := cronSpec(config)
	if _, ok := m.cronEntries[workflow][spec]; !ok {
		entryId, err := m.cron.AddJob(spec, m.testWorkflowExecuteJob(ctx, workflow, spec, config))
		if err != nil {
			return fmt.Errorf("adding cron %q for workflow %q: %w", spec, workflow, err)
		}
		m.cronEntries[workflow][spec] = entryId
	}

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

		log.Debugf("started scheduled workflow execution",
			"execution id", executionID,
		)
	})
}

// Delete a schedule from the manager.
func (m Manager) Delete(_ context.Context, workflow string, config testkube.TestWorkflowCronJobConfig) error {
	if _, ok := m.cronEntries[workflow]; !ok {
		// Already gone, mission failed successfully!
		return nil
	}

	spec := cronSpec(config)
	// Only need to remove if it isn't in the entries.
	if _, ok := m.cronEntries[workflow][spec]; ok {
		m.cron.Remove(m.cronEntries[workflow][spec])
		delete(m.cronEntries[workflow], spec)
	}

	return nil
}
