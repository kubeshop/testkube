package scheduler

import (
	"context"
	"fmt"
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"
	"github.com/kubeshop/testkube/pkg/workerpool"
	"github.com/pkg/errors"
	"sync"
	"time"
)

func (s *Scheduler) PrepareTestSuiteRequests(work []testsuitesv2.TestSuite, request testkube.TestSuiteExecutionRequest) []workerpool.Request[
	testkube.TestSuite,
	testkube.TestSuiteExecutionRequest,
	testkube.TestSuiteExecution,
] {
	requests := make([]workerpool.Request[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution], len(work))
	for i := range work {
		requests[i] = workerpool.Request[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution]{
			Object:  testsuitesmapper.MapCRToAPI(work[i]),
			Options: request,
			ExecFn:  s.executeTestSuite,
		}
	}
	return requests
}

func (s *Scheduler) executeTestSuite(ctx context.Context, testSuite testkube.TestSuite, request testkube.TestSuiteExecutionRequest) (
	testsuiteExecution testkube.TestSuiteExecution, err error) {
	s.logger.Debugw("Got testsuite to execute", "test", testSuite)
	secretUUID, err := s.testSuitesClient.GetCurrentSecretUUID(testSuite.Name)
	if err != nil {
		return testsuiteExecution, err
	}

	request.SecretUUID = secretUUID
	if testSuite.ExecutionRequest != nil {
		if request.Name == "" && testSuite.ExecutionRequest.Name != "" {
			request.Name = testSuite.ExecutionRequest.Name
		}
		if request.HttpProxy == "" && testSuite.ExecutionRequest.HttpProxy != "" {
			request.HttpProxy = testSuite.ExecutionRequest.HttpProxy
		}

		if request.HttpsProxy == "" && testSuite.ExecutionRequest.HttpsProxy != "" {
			request.HttpsProxy = testSuite.ExecutionRequest.HttpsProxy
		}
	}

	request.Number = s.getNextExecutionNumber("ts-" + testSuite.Name)
	if request.Name == "" {
		request.Name = fmt.Sprintf("ts-%s-%d", testSuite.Name, request.Number)
	}

	testsuiteExecution = testkube.NewStartedTestSuiteExecution(testSuite, request)
	err = s.testExecutionResults.Insert(ctx, testsuiteExecution)
	if err != nil {
		s.logger.Infow("Inserting test execution", "error", err)
	}

	s.events.Notify(testkube.NewEventStartTestSuite(&testsuiteExecution))

	var wg sync.WaitGroup
	wg.Add(1)
	go func(testsuiteExecution *testkube.TestSuiteExecution, request testkube.TestSuiteExecutionRequest) {
		defer func(testExecution *testkube.TestSuiteExecution) {
			duration := testExecution.CalculateDuration()
			testExecution.EndTime = time.Now()
			testExecution.Duration = duration.String()

			err = s.testExecutionResults.EndExecution(ctx, testExecution.Id, testExecution.EndTime, duration)
			if err != nil {
				s.logger.Errorw("error setting end time", "error", err.Error())
			}

			wg.Done()
		}(testsuiteExecution)

		hasFailedSteps := false
		cancelSteps := false
		for i := range testsuiteExecution.StepResults {
			if cancelSteps {
				testsuiteExecution.StepResults[i].Execution.ExecutionResult.Cancel()
				continue
			}

			// start execution of given step
			testsuiteExecution.StepResults[i].Execution.ExecutionResult.InProgress()
			err = s.testExecutionResults.Update(ctx, *testsuiteExecution)
			if err != nil {
				s.logger.Infow("Updating test execution", "error", err)
			}

			s.executeTestStep(ctx, *testsuiteExecution, request, &testsuiteExecution.StepResults[i])

			err := s.testExecutionResults.Update(ctx, *testsuiteExecution)
			if err != nil {
				hasFailedSteps = true

				s.logger.Errorw("saving test suite execution results error", "error", err)
				continue
			}

			if testsuiteExecution.StepResults[i].IsFailed() {
				hasFailedSteps = true
				if testsuiteExecution.StepResults[i].Step.StopTestOnFailure {
					cancelSteps = true
					continue
				}
			}
		}

		testsuiteExecution.Status = testkube.TestSuiteExecutionStatusPassed
		if hasFailedSteps {
			testsuiteExecution.Status = testkube.TestSuiteExecutionStatusFailed
			s.events.Notify(testkube.NewEventEndTestSuiteFailed(testsuiteExecution))
		} else {
			s.events.Notify(testkube.NewEventEndTestSuiteSuccess(testsuiteExecution))
		}

		s.metrics.IncExecuteTestSuite(*testsuiteExecution)

		err := s.testExecutionResults.Update(ctx, *testsuiteExecution)
		if err != nil {
			s.logger.Errorw("saving final test suite execution result error", "error", err)
		}

	}(&testsuiteExecution, request)

	// wait for sync test suite execution
	if request.Sync {
		wg.Wait()
	}

	return testsuiteExecution, nil
}

func (s *Scheduler) executeTestStep(ctx context.Context, testsuiteExecution testkube.TestSuiteExecution,
	request testkube.TestSuiteExecutionRequest, result *testkube.TestSuiteStepExecutionResult) {

	var testSuiteName string
	if testsuiteExecution.TestSuite != nil {
		testSuiteName = testsuiteExecution.TestSuite.Name
	}

	step := result.Step

	l := s.logger.With("type", step.Type(), "testSuiteName", testSuiteName, "name", step.FullName())

	switch step.Type() {

	case testkube.TestSuiteStepTypeExecuteTest:
		executeTestStep := step.Execute
		request := testkube.ExecutionRequest{
			Name:                fmt.Sprintf("%s-%s", testSuiteName, executeTestStep.Name),
			TestSuiteName:       testSuiteName,
			Namespace:           executeTestStep.Namespace,
			Variables:           testsuiteExecution.Variables,
			TestSuiteSecretUUID: request.SecretUUID,
			Sync:                true,
			HttpProxy:           request.HttpProxy,
			HttpsProxy:          request.HttpsProxy,
			ExecutionLabels:     request.ExecutionLabels,
		}

		l.Info("executing test", "variables", testsuiteExecution.Variables, "request", request)
		execution, err := s.executeTest(ctx, testkube.Test{Name: executeTestStep.Name}, request)
		if err != nil {
			result.Err(err)
			return
		}
		result.Execution = &execution

	case testkube.TestSuiteStepTypeDelay:
		l.Debug("delaying execution")
		time.Sleep(time.Millisecond * time.Duration(step.Delay.Duration))
		result.Execution.ExecutionResult.Success()

	default:
		result.Err(errors.Errorf("can't find handler for execution step type: '%v'", step.Type()))
	}
}
