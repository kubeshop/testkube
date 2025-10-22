package cronjob

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
	cronjobtcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cronjob"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

const (
	concurrencyLevel = 10
)

func (s *Scheduler) executeTestWorkflow(ctx context.Context, testWorkflowName string, cron *testkube.TestWorkflowCronJobConfig) {
	var targets []*cloud.ExecutionTarget
	if cron.Target != nil {
		targets = commonmapper.MapAllTargetsApiToGrpc([]testkube.ExecutionTarget{*cron.Target})
	}

	request := &cloud.ScheduleRequest{
		Executions: []*cloud.ScheduleExecution{{
			Selector: &cloud.ScheduleResourceSelector{Name: testWorkflowName},
			Config:   cron.Config,
			Targets:  targets,
		},
		},
	}

	cronName := cron.Cron
	if cron.Timezone != nil {
		cronName = fmt.Sprintf("CRON_TZ=%s %s", cron.Timezone.Value, cron.Cron)
	}

	// Pro edition only (tcl protected code)
	if s.proContext != nil && s.proContext.APIKey != "" {
		request.RunningContext, _ = testworkflowexecutor.GetNewRunningContext(cronjobtcl.GetRunningContext(cronName), nil)
	}

	s.logger.Infof(
		"cron job scheduler: executor component: scheduling testworkflow execution for %s/%s",
		testWorkflowName, cronName,
	)

	results, err := s.testWorkflowExecutor.Execute(ctx, request)
	if err != nil {
		s.logger.Errorw(fmt.Sprintf("cron job scheduler: executor component: error executing testworkflow for cron %s/%s", testWorkflowName, cronName), "error", err)
		return
	}

	executionID := ""
	if len(results) != 0 {
		executionID = results[0].Id
	}

	s.logger.Debugf("cron job scheduler: executor component: started test workflow execution for cron %s/%s/%s", testWorkflowName, cron, executionID)
}
