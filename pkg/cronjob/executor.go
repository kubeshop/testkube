package cronjob

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cronjobtcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cronjob"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func (s *Scheduler) execute(ctx context.Context, testWorkflowName string, cron *testkube.TestWorkflowCronJobConfig) error {
	request := &cloud.ScheduleRequest{
		Executions: []*cloud.ScheduleExecution{{
			Selector: &cloud.ScheduleResourceSelector{Name: testWorkflowName},
			Config:   cron.Config,
		},
		},
	}

	// Pro edition only (tcl protected code)
	if s.proContext != nil && s.proContext.APIKey != "" {
		request.RunningContext, _ = testworkflowexecutor.GetNewRunningContext(cronjobtcl.GetRunningContext(cron.Cron), nil)
	}

	s.logger.Infof(
		"cron job scheduler: executor component: scheduling testworkflow execution for %s/%s",
		testWorkflowName, cron.Cron,
	)

	resp := s.testWorkflowExecutor.Execute(ctx, "", request)

	results := make([]testkube.TestWorkflowExecution, 0)
	for v := range resp.Channel() {
		results = append(results, *v)
	}

	if resp.Error() != nil {
		s.logger.Errorw(fmt.Sprintf("cron job scheduler: executor component: error executing testworkflow for cron %s/%s", testWorkflowName, cron.Cron), "error", resp.Error())
		return nil
	}

	executionID := ""
	if len(results) != 0 {
		executionID = results[0].Id
	}

	s.logger.Debugf("cron job scheduler: executor component: started test workflow execution for cron %s/%s/%s", testWorkflowName, cron, executionID)
	return nil
}
