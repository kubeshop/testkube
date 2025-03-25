package cronjob

import (
	"context"
	"fmt"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"
	cronjobtcl "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/cronjob"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/workerpool"
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
		return
	}

	executionID := ""
	if len(results) != 0 {
		executionID = results[0].Id
	}

	s.logger.Debugf("cron job scheduler: executor component: started test workflow execution for cron %s/%s/%s", testWorkflowName, cron, executionID)
}

func (s *Scheduler) executeTest(ctx context.Context, testName string, schedule string) error {
	s.logger.Debugf("cron job scheduler: executor component: fetching testsv3.Test with name %s", testName)

	test, err := s.testClient.Get(testName)
	if err != nil {
		return err
	}

	request := testkube.ExecutionRequest{
		RunningContext: &testkube.RunningContext{
			Type_:   string(testkube.RunningContextTypeScheduler),
			Context: schedule,
		},
	}

	wp := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](concurrencyLevel)
	go func() {
		s.logger.Infof(
			"cron job scheduler: executor component: scheduling test execution for %s/%s",
			testName, schedule,
		)

		go wp.SendRequests(s.prepareTestRequest(*test, request))
		go wp.Run(ctx)
	}()

	executionID := ""
	for r := range wp.GetResponses() {
		executionID = r.Result.Id
		err = r.Err
	}

	if err != nil {
		s.logger.Errorw(fmt.Sprintf("cron job scheduler: executor component: error executing test for schedule %s/%s", testName, schedule), "error", err)
		return nil
	}

	s.logger.Debugf("cron job scheduler: executor component: started test execution for schedule %s/%s/%s", testName, schedule, executionID)
	return nil
}

func (s *Scheduler) executeTestSuite(ctx context.Context, testSuiteName string, schedule string) error {
	s.logger.Debugf("cron job scheduler: executor component: fetching testsuitesv3.TestSuite with name %s", testSuiteName)

	testSuite, err := s.testSuiteClient.Get(testSuiteName)
	if err != nil {
		return err
	}

	request := testkube.TestSuiteExecutionRequest{
		RunningContext: &testkube.RunningContext{
			Type_:   string(testkube.RunningContextTypeScheduler),
			Context: schedule,
		},
	}

	wp := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](concurrencyLevel)
	go func() {
		s.logger.Infof(
			"cron job scheduler: executor component: scheduling test suite execution for %s/%s",
			testSuiteName, schedule,
		)

		go wp.SendRequests(s.prepareTestSuiteRequest(*testSuite, request))
		go wp.Run(ctx)
	}()

	executionID := ""
	for r := range wp.GetResponses() {
		executionID = r.Result.Id
		err = r.Err
	}

	if err != nil {
		s.logger.Errorw(fmt.Sprintf("cron job scheduler: executor component: error executing test suite for schedule %s/%s", testSuiteName, schedule), "error", err)
		return nil
	}

	s.logger.Debugf("cron job scheduler: executor component: started test suite execution for schedule %s/%s/%s", testSuiteName, schedule, executionID)
	return nil
}

func (s *Scheduler) prepareTestRequest(test testsv3.Test, request testkube.ExecutionRequest) []workerpool.Request[
	testkube.Test, testkube.ExecutionRequest, testkube.Execution] {
	requests := []workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution]{
		{
			Object:  testsmapper.MapTestCRToAPI(test),
			Options: request,
			ExecFn:  s.executeTestFn,
		},
	}
	return requests
}

func (s *Scheduler) prepareTestSuiteRequest(testSuite testsuitesv3.TestSuite, request testkube.TestSuiteExecutionRequest) []workerpool.Request[
	testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution] {
	requests := []workerpool.Request[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution]{
		{
			Object:  testsuitesmapper.MapCRToAPI(testSuite),
			Options: request,
			ExecFn:  s.executeTestSuiteFn,
		},
	}
	return requests
}
