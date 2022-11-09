package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"
	"github.com/kubeshop/testkube/pkg/workerpool"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
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
	go s.runSteps(ctx, &wg, &testsuiteExecution, request)

	// wait for sync test suite execution
	if request.Sync {
		wg.Wait()
	}

	return testsuiteExecution, nil
}

func (s *Scheduler) runSteps(ctx context.Context, wg *sync.WaitGroup, testsuiteExecution *testkube.TestSuiteExecution, request testkube.TestSuiteExecutionRequest) {
	defer func(testExecution *testkube.TestSuiteExecution) {
		testExecution.Stop()
		err := s.testExecutionResults.EndExecution(ctx, *testExecution)
		if err != nil {
			s.logger.Errorw("error setting end time", "error", err.Error())
		}

		wg.Done()
	}(testsuiteExecution)
	s.logger.Infow("Running steps", "test", testsuiteExecution.Name)
	hasFailedSteps := false
	cancelSteps := false
	var stepResult *testkube.TestSuiteStepExecutionResult
	abortChan := make(chan bool)

	go s.abortionCheck(ctx, testsuiteExecution, abortChan)
	for i := range testsuiteExecution.StepResults {
		stepResult = &testsuiteExecution.StepResults[i]
		select {
		case <-abortChan:
			s.logger.Infow("Aborting test suite execution", "execution", testsuiteExecution.Id, "i", i)
			cancelSteps = true
			stepResult.Execution.ExecutionResult.Abort()
			testsuiteExecution.Status = testkube.TestSuiteExecutionStatusAborting
		default:
			s.logger.Debugw("Running step", "step", testsuiteExecution.StepResults[i].Step, "i", i)
			if cancelSteps {
				stepResult.Execution.ExecutionResult.Abort()
				s.logger.Debugw("Aborting step", "step", testsuiteExecution.StepResults[i].Step, "i", i)
				continue
			}

			// start execution of given step
			stepResult.Execution.ExecutionResult.InProgress()
			err := s.testExecutionResults.Update(ctx, *testsuiteExecution)
			if err != nil {
				s.logger.Infow("Updating test execution", "error", err)
			}

			s.executeTestStep(ctx, *testsuiteExecution, request, stepResult)

			s.logger.Debugw("Step execution result", "step", testsuiteExecution.StepResults[i].Step, "result", stepResult.Execution.ExecutionResult)
			err = s.testExecutionResults.Update(ctx, *testsuiteExecution)
			if err != nil {
				hasFailedSteps = true

				s.logger.Errorw("saving test suite execution results error", "error", err)
				continue
			}

			if stepResult.IsFailed() {
				hasFailedSteps = true
				if stepResult.Step.StopTestOnFailure {
					cancelSteps = true
					continue
				}
			}
		}
	}

	if *testsuiteExecution.Status == testkube.ABORTING_TestSuiteExecutionStatus {
		s.events.Notify(testkube.NewEventEndTestSuiteAborted(testsuiteExecution))
		testsuiteExecution.Status = testkube.TestSuiteExecutionStatusAborted
	} else if hasFailedSteps {
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

}

// abortionCheck is polling database to see if the user aborted the test suite execution
func (s *Scheduler) abortionCheck(ctx context.Context, testsuiteExecution *testkube.TestSuiteExecution, abortChan chan bool) {
	const abortionPollingInterval = 100 * time.Millisecond
	s.logger.Debugw("Abortion check started", "test", testsuiteExecution.Name)
	ticker := time.NewTicker(abortionPollingInterval)
	for testsuiteExecution.Status == testkube.TestSuiteExecutionStatusRunning {
		<-ticker.C
		if s.wasTestSuiteAborted(ctx, testsuiteExecution.Id) {
			s.logger.Debugw("Aborting test suite execution", "execution", testsuiteExecution.Id)
			abortChan <- true
			break
		}
	}
	s.logger.Debugw("Abortion check finished", "test", testsuiteExecution.Name)
}

func (s *Scheduler) wasTestSuiteAborted(ctx context.Context, id string) bool {
	execution, err := s.testExecutionResults.Get(ctx, id)
	if err == mongo.ErrNoDocuments {
		execution, err = s.testExecutionResults.GetByName(ctx, id)
	}
	if err != nil {
		s.logger.Errorw("getting test execution", "error", err)
		return false
	}

	s.logger.Infow("Checking if test suite execution was aborted", "id", id, "status", execution.Status)
	return *execution.Status == testkube.ABORTING_TestSuiteExecutionStatus
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
		l.Infow("delaying execution", "step", step.FullName(), "delay", step.Delay.Duration)
		duration := time.Millisecond * time.Duration(step.Delay.Duration)
		s.delayWithAbortionCheck(duration, testsuiteExecution.Id, result)
	default:
		result.Err(errors.Errorf("can't find handler for execution step type: '%v'", step.Type()))
	}
}

func (s *Scheduler) delayWithAbortionCheck(duration time.Duration, testSuiteId string, result *testkube.TestSuiteStepExecutionResult) {
	timer := time.NewTimer(duration)
	const abortionPollingInterval = 100 * time.Millisecond
	ticker := time.NewTicker(abortionPollingInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			s.logger.Infow("delay finished", "testSuiteId", testSuiteId, "duration", duration)
			result.Execution.ExecutionResult.Success()
			return
		case <-ticker.C:
			if s.wasTestSuiteAborted(context.Background(), testSuiteId) {
				s.logger.Infow("delay aborted", "testSuiteId", testSuiteId, "duration", duration)
				result.Execution.ExecutionResult.Abort()
				return
			}
		}
	}
}
