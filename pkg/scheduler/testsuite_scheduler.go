package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"

	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	testsuiteexecutionsmapper "github.com/kubeshop/testkube/pkg/mapper/testsuiteexecutions"
	testsuitesmapper "github.com/kubeshop/testkube/pkg/mapper/testsuites"

	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/version"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

const (
	// DefaultConcurrencyLevel is a default concurrency level for worker pool
	DefaultConcurrencyLevel = 10
)

type testTuple struct {
	test        testkube.Test
	executionID string
	stepRequest *testkube.TestSuiteStepExecutionRequest
}

func (s *Scheduler) PrepareTestSuiteRequests(work []testsuitesv3.TestSuite, request testkube.TestSuiteExecutionRequest) []workerpool.Request[
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
		if request.Timeout == 0 && testSuite.ExecutionRequest.Timeout != 0 {
			request.Timeout = testSuite.ExecutionRequest.Timeout
		}

		var fields = []struct {
			source      string
			destination *string
		}{
			{
				testSuite.ExecutionRequest.Name,
				&request.Name,
			},
			{
				testSuite.ExecutionRequest.HttpProxy,
				&request.HttpProxy,
			},
			{
				testSuite.ExecutionRequest.HttpsProxy,
				&request.HttpsProxy,
			},
			{
				testSuite.ExecutionRequest.JobTemplate,
				&request.JobTemplate,
			},
			{
				testSuite.ExecutionRequest.JobTemplateReference,
				&request.JobTemplateReference,
			},
			{
				testSuite.ExecutionRequest.ScraperTemplate,
				&request.ScraperTemplate,
			},
			{
				testSuite.ExecutionRequest.ScraperTemplateReference,
				&request.ScraperTemplateReference,
			},
			{
				testSuite.ExecutionRequest.PvcTemplate,
				&request.PvcTemplate,
			},
			{
				testSuite.ExecutionRequest.PvcTemplateReference,
				&request.PvcTemplateReference,
			},
		}

		for _, field := range fields {
			if *field.destination == "" && field.source != "" {
				*field.destination = field.source
			}
		}
	}

	s.logger.Infow("Executing testsuite", "test", testSuite.Name, "request", request, "ExecutionRequest", testSuite.ExecutionRequest)

	request.Number = s.getNextExecutionNumber("ts-" + testSuite.Name)
	if request.Name == "" {
		request.Name = fmt.Sprintf("ts-%s-%d", testSuite.Name, request.Number)
	}

	testsuiteExecution = testkube.NewStartedTestSuiteExecution(testSuite, request)
	err = s.testsuiteResults.Insert(ctx, testsuiteExecution)
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
	defer s.runAfterEachStep(ctx, testsuiteExecution, wg)

	s.logger.Infow("Running steps", "test", testsuiteExecution.Name)

	statusChan := make(chan *testkube.TestSuiteExecutionStatus)
	hasFailedSteps := false
	cancelSteps := false
	var batchStepResult *testkube.TestSuiteBatchStepExecutionResult

	var abortionStatus *testkube.TestSuiteExecutionStatus

	go s.timeoutCheck(ctx, testsuiteExecution, request.Timeout)

	err := s.eventsBus.SubscribeTopic(bus.InternalSubscribeTopic, testsuiteExecution.Name, func(event testkube.Event) error {
		s.logger.Infow("test suite abortion event in runSteps", "event", event)
		if event.TestSuiteExecution != nil &&
			event.TestSuiteExecution.Id == testsuiteExecution.Id &&
			event.Type_ != nil &&
			(*event.Type_ == testkube.END_TESTSUITE_ABORTED_EventType || *event.Type_ == testkube.END_TESTSUITE_TIMEOUT_EventType) {
			s.logger.Infow("Aborting test suite execution", "execution", testsuiteExecution.Id)

			status := testkube.TestSuiteExecutionStatusAborting
			if *event.Type_ == testkube.END_TESTSUITE_TIMEOUT_EventType {
				status = testkube.TestSuiteExecutionStatusTimeout
			}
			statusChan <- status
		}
		return nil
	})

	if err != nil {
		s.logger.Errorw("error subscribing to event", "error", err)
	}

	for i := range testsuiteExecution.ExecuteStepResults {
		batchStepResult = &testsuiteExecution.ExecuteStepResults[i]
		s.logger.Debugw("Running batch step", "step", batchStepResult.Execute, "i", i)

		select {
		case status := <-statusChan:
			abortionStatus = status
			cancelSteps = true
		default:
		}

		if cancelSteps {
			s.logger.Infow("Aborting batch step", "step", batchStepResult.Execute, "i", i)
			for j := range batchStepResult.Execute {
				if batchStepResult.Execute[j].Execution != nil && batchStepResult.Execute[j].Execution.ExecutionResult != nil {
					batchStepResult.Execute[j].Execution.ExecutionResult.Abort()
				}
			}

			testsuiteExecution.Status = testkube.TestSuiteExecutionStatusAborting

			for j := range batchStepResult.Execute {
				if batchStepResult.Execute[j].Execution != nil && batchStepResult.Execute[j].Execution.ExecutionResult != nil {
					batchStepResult.Execute[j].Execution.ExecutionResult.Abort()
				}
			}

			continue
		}

		// start execution of given step
		for j := range batchStepResult.Execute {
			if batchStepResult.Execute[j].Execution != nil && batchStepResult.Execute[j].Execution.ExecutionResult != nil {
				batchStepResult.Execute[j].Execution.ExecutionResult.InProgress()
			}
		}

		err := s.testsuiteResults.Update(ctx, *testsuiteExecution)
		if err != nil {
			s.logger.Infow("Updating test execution", "error", err)
		}

		s.executeTestStep(ctx, *testsuiteExecution, request, batchStepResult, testsuiteExecution.ExecuteStepResults[:i])

		var results []*testkube.ExecutionResult
		for j := range batchStepResult.Execute {
			if batchStepResult.Execute[j].Execution != nil && batchStepResult.Execute[j].Execution.ExecutionResult != nil {
				results = append(results, batchStepResult.Execute[j].Execution.ExecutionResult)
			}
		}

		s.logger.Debugw("Batch step execution result", "step", batchStepResult.Execute, "results", results)

		err = s.testsuiteResults.Update(ctx, *testsuiteExecution)
		if err != nil {
			s.logger.Errorw("saving test suite execution results error", "error", err)

			hasFailedSteps = true
			continue
		}

		for j := range batchStepResult.Execute {
			if batchStepResult.Execute[j].IsFailed() {
				hasFailedSteps = true
				if batchStepResult.Step != nil && batchStepResult.Step.StopOnFailure {
					cancelSteps = true
					break
				}
			}
		}
	}
	s.logger.Infow("Finished running steps", "test", testsuiteExecution.Name, "hasFailedSteps", hasFailedSteps, "cancelSteps", cancelSteps, "status", testsuiteExecution.Status)

	if testsuiteExecution.Status != nil && *testsuiteExecution.Status == testkube.ABORTING_TestSuiteExecutionStatus {
		if abortionStatus != nil && *abortionStatus == testkube.TIMEOUT_TestSuiteExecutionStatus {
			s.events.Notify(testkube.NewEventEndTestSuiteTimeout(testsuiteExecution))
			testsuiteExecution.Status = testkube.TestSuiteExecutionStatusTimeout
		} else {
			s.events.Notify(testkube.NewEventEndTestSuiteAborted(testsuiteExecution))
			testsuiteExecution.Status = testkube.TestSuiteExecutionStatusAborted
		}
	} else if hasFailedSteps {
		testsuiteExecution.Status = testkube.TestSuiteExecutionStatusFailed
		s.events.Notify(testkube.NewEventEndTestSuiteFailed(testsuiteExecution))
	} else {
		testsuiteExecution.Status = testkube.TestSuiteExecutionStatusPassed
		s.events.Notify(testkube.NewEventEndTestSuiteSuccess(testsuiteExecution))
	}

	s.metrics.IncAndObserveExecuteTestSuite(*testsuiteExecution, s.dashboardURI)

	err = s.testsuiteResults.Update(ctx, *testsuiteExecution)
	if err != nil {
		s.logger.Errorw("saving final test suite execution result error", "error", err)
	}

	s.eventsBus.Unsubscribe(testsuiteExecution.Name)
}

func (s *Scheduler) runAfterEachStep(ctx context.Context, execution *testkube.TestSuiteExecution, wg *sync.WaitGroup) {
	execution.Stop()
	err := s.testsuiteResults.EndExecution(ctx, *execution)
	if err != nil {
		s.logger.Errorw("error setting end time", "error", err.Error())
	}

	wg.Done()

	if execution.TestSuite != nil {
		testSuite, err := s.testSuitesClient.Get(execution.TestSuite.Name)
		if err != nil {
			s.logger.Errorw("getting test suite error", "error", err)
		}

		if testSuite != nil {
			testSuite.Status = testsuitesmapper.MapExecutionToTestSuiteStatus(execution)
			if err = s.testSuitesClient.UpdateStatus(testSuite); err != nil {
				s.logger.Errorw("updating test suite error", "error", err)
			}

			if execution.TestSuiteExecutionName != "" {
				testSuiteExecution, err := s.testSuiteExecutionsClient.Get(execution.TestSuiteExecutionName)
				if err != nil {
					s.logger.Errorw("getting test suite execution error", "error", err)
				}

				if testSuiteExecution != nil {
					testSuiteExecution.Status = testsuiteexecutionsmapper.MapAPIToCRD(execution, testSuiteExecution.Generation)
					if err = s.testSuiteExecutionsClient.UpdateStatus(testSuiteExecution); err != nil {
						s.logger.Errorw("updating test suite execution error", "error", err)
					}
				}
			}
		}
	}

	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		s.logger.Debugw("getting telemetry enabled error", "error", err)
	}

	if !telemetryEnabled {
		return
	}

	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		s.logger.Debugw("getting cluster id error", "error", err)
	}

	host, err := os.Hostname()
	if err != nil {
		s.logger.Debugw("getting hostname error", "hostname", host, "error", err)
	}

	status := ""
	if execution.Status != nil {
		status = string(*execution.Status)
	}

	out, err := telemetry.SendRunEvent("testkube_api_run_test_suite", telemetry.RunParams{
		AppVersion: version.Version,
		Host:       host,
		ClusterID:  clusterID,
		DurationMs: execution.DurationMs,
		Status:     status,
	})

	if err != nil {
		s.logger.Debugw("sending run test suite telemetry event error", "error", err)
	} else {
		s.logger.Debugw("sending run test suite telemetry event", "output", out)
	}
}

// timeoutCheck is checking if the testsuite has timed out
func (s *Scheduler) timeoutCheck(ctx context.Context, testsuiteExecution *testkube.TestSuiteExecution, timeout int32) {
	s.logger.Infow("timeout check started", "test", testsuiteExecution.Name, "timeout", timeout)

	timer := time.NewTimer(time.Duration(timeout) * time.Second)

	defer func() {
		timer.Stop()
	}()

	for testsuiteExecution.Status == testkube.TestSuiteExecutionStatusRunning {
		select {
		case <-timer.C:
			s.logger.Debugw("testsuite timeout occured", "test suite", testsuiteExecution.Name)

			if timeout > 0 {
				s.logger.Debugw("aborting test suite execution due to timeout", "execution", testsuiteExecution.Id)

				err := s.eventsBus.PublishTopic(bus.InternalPublishTopic, testkube.NewEventEndTestSuiteTimeout(testsuiteExecution))
				if err != nil {
					s.logger.Errorw("error publishing event", "error", err)
				}
				return
			}
		case <-ctx.Done():
			return
		}
	}

	s.logger.Debugw("Timeout check, finished checking", "test", testsuiteExecution.Name)
}

func (s *Scheduler) executeTestStep(ctx context.Context, testsuiteExecution testkube.TestSuiteExecution,
	request testkube.TestSuiteExecutionRequest, result *testkube.TestSuiteBatchStepExecutionResult,
	previousSteps []testkube.TestSuiteBatchStepExecutionResult) {

	var testSuiteName string
	if testsuiteExecution.TestSuite != nil {
		testSuiteName = testsuiteExecution.TestSuite.Name
	}

	ids := make(map[string]struct{})
	testNames := make(map[string]struct{})
	if result.Step != nil && result.Step.DownloadArtifacts != nil {
		for _, testName := range result.Step.DownloadArtifacts.PreviousTestNames {
			testNames[testName] = struct{}{}
		}

		for i := range previousSteps {
			for j := range previousSteps[i].Execute {
				if previousSteps[i].Execute[j].Execution != nil &&
					previousSteps[i].Execute[j].Step != nil && previousSteps[i].Execute[j].Step.Test != "" {
					if previousSteps[i].Execute[j].Execution.IsPassed() || previousSteps[i].Execute[j].Execution.IsFailed() {
						if result.Step.DownloadArtifacts.AllPreviousSteps {
							ids[previousSteps[i].Execute[j].Execution.Id] = struct{}{}
						} else {
							for _, n := range result.Step.DownloadArtifacts.PreviousStepNumbers {
								if n == int32(i+1) {
									ids[previousSteps[i].Execute[j].Execution.Id] = struct{}{}
									break
								}
							}

							if _, ok := testNames[previousSteps[i].Execute[j].Step.Test]; ok {
								ids[previousSteps[i].Execute[j].Execution.Id] = struct{}{}
							}
						}
					}
				}
			}
		}
	}

	var testTuples []testTuple
	var duration time.Duration
	for i := range result.Execute {
		step := result.Execute[i].Step
		if step == nil {
			continue
		}

		l := s.logger.With("type", step.Type(), "testSuiteName", testSuiteName, "name", step.FullName())

		switch step.Type() {
		case testkube.TestSuiteStepTypeExecuteTest:
			executeTestStep := step.Test
			if executeTestStep == "" {
				continue
			}

			execution := result.Execute[i].Execution
			if execution == nil {
				continue
			}

			l.Info("executing test", "variables", testsuiteExecution.Variables, "request", request)

			testTuples = append(testTuples, testTuple{
				test:        testkube.Test{Name: executeTestStep, Namespace: testsuiteExecution.TestSuite.Namespace},
				executionID: execution.Id,
				stepRequest: step.ExecutionRequest,
			})
		case testkube.TestSuiteStepTypeDelay:
			if step.Delay == "" {
				continue
			}

			l.Infow("delaying execution", "step", step.FullName(), "delay", step.Delay)

			delay, err := time.ParseDuration(step.Delay)
			if err != nil {
				result.Execute[i].Err(err)
				continue
			}

			if delay > duration {
				duration = delay
			}
		default:
			result.Execute[i].Err(errors.Errorf("can't find handler for execution step type: '%v'", step.Type()))
		}
	}

	concurrencyLevel := DefaultConcurrencyLevel
	if request.ConcurrencyLevel != 0 {
		concurrencyLevel = int(request.ConcurrencyLevel)
	}

	workerpoolService := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](concurrencyLevel)

	if len(testTuples) != 0 {
		var executionIDs []string
		for id := range ids {
			executionIDs = append(executionIDs, id)
		}

		req := testkube.ExecutionRequest{
			TestSuiteName:         testSuiteName,
			Variables:             testsuiteExecution.Variables,
			TestSuiteSecretUUID:   request.SecretUUID,
			Sync:                  true,
			HttpProxy:             request.HttpProxy,
			HttpsProxy:            request.HttpsProxy,
			ExecutionLabels:       request.ExecutionLabels,
			ActiveDeadlineSeconds: int64(request.Timeout),
			ContentRequest:        request.ContentRequest,
			RunningContext: &testkube.RunningContext{
				Type_:   string(testkube.RunningContextTypeTestSuite),
				Context: testsuiteExecution.Name,
			},
			JobTemplate:                  request.JobTemplate,
			JobTemplateReference:         request.JobTemplateReference,
			ScraperTemplate:              request.ScraperTemplate,
			ScraperTemplateReference:     request.ScraperTemplateReference,
			PvcTemplate:                  request.PvcTemplate,
			PvcTemplateReference:         request.PvcTemplateReference,
			DownloadArtifactExecutionIDs: executionIDs,
		}

		requests := make([]workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution], len(testTuples))
		for i := range testTuples {
			req.Name = fmt.Sprintf("%s-%s", testSuiteName, testTuples[i].test.Name)
			req.Id = testTuples[i].executionID
			req = MergeStepRequest(testTuples[i].stepRequest, req)
			requests[i] = workerpool.Request[testkube.Test, testkube.ExecutionRequest, testkube.Execution]{
				Object:  testTuples[i].test,
				Options: req,
				ExecFn:  s.executeTest,
			}
		}

		go workerpoolService.SendRequests(requests)
		go workerpoolService.Run(ctx)
	}

	result.Start()
	if err := s.testsuiteResults.Update(ctx, testsuiteExecution); err != nil {
		s.logger.Errorw("saving test suite execution start time error", "error", err)
	}

	if duration != 0 {
		s.delayWithAbortionCheck(duration, testsuiteExecution.Id, result)
	}

	if len(testTuples) != 0 {
		for r := range workerpoolService.GetResponses() {
			status := ""
			if r.Result.ExecutionResult != nil && r.Result.ExecutionResult.Status != nil {
				status = string(*r.Result.ExecutionResult.Status)
			}

			s.logger.Infow("execution result", "id", r.Result.Id, "status", status)
			value := r.Result
			for i := range result.Execute {
				if result.Execute[i].Execution == nil {
					continue
				}

				if result.Execute[i].Execution.Id == r.Result.Id {
					result.Execute[i].Execution = &value

					if err := s.testsuiteResults.Update(ctx, testsuiteExecution); err != nil {
						s.logger.Errorw("saving test suite execution results error", "error", err)
					}
				}
			}
		}
	}

	result.Stop()
	if err := s.testsuiteResults.Update(ctx, testsuiteExecution); err != nil {
		s.logger.Errorw("saving test suite execution end time error", "error", err)
	}
}

func (s *Scheduler) delayWithAbortionCheck(duration time.Duration, testSuiteId string, result *testkube.TestSuiteBatchStepExecutionResult) {
	timer := time.NewTimer(duration)

	defer func() {
		timer.Stop()
	}()

	abortChan := make(chan bool)

	err := s.eventsBus.SubscribeTopic(bus.InternalSubscribeTopic, testSuiteId, func(event testkube.Event) error {
		s.logger.Infow("test suite abortion event in delay handling", "event", event)
		if event.TestSuiteExecution != nil &&
			event.TestSuiteExecution.Id == testSuiteId &&
			event.Type_ != nil &&
			*event.Type_ == testkube.END_TESTSUITE_ABORTED_EventType {

			s.logger.Infow("delay aborted", "testSuiteId", testSuiteId, "duration", duration)
			abortChan <- true
		}
		return nil
	})

	if err != nil {
		s.logger.Errorw("error subscribing to event", "error", err)
	}

	for {
		select {
		case <-timer.C:
			s.logger.Infow("delay finished", "testSuiteId", testSuiteId, "duration", duration)

			for i := range result.Execute {
				if result.Execute[i].Step != nil && result.Execute[i].Step.Delay != "" &&
					result.Execute[i].Execution != nil && result.Execute[i].Execution.ExecutionResult != nil {
					result.Execute[i].Execution.ExecutionResult.Success()
				}
			}

			return
		case <-abortChan:

			for i := range result.Execute {
				if result.Execute[i].Step != nil && result.Execute[i].Step.Delay != "" &&
					result.Execute[i].Execution != nil && result.Execute[i].Execution.ExecutionResult != nil {
					delay, err := time.ParseDuration(result.Execute[i].Step.Delay)
					if err != nil {
						result.Execute[i].Err(err)
						continue
					}

					if delay < duration {
						result.Execute[i].Execution.ExecutionResult.Success()
						continue
					}

					result.Execute[i].Execution.ExecutionResult.Abort()
				}
			}
			return
		}
	}
}

// MergeStepRequest inherits step request fields with execution request
func MergeStepRequest(stepRequest *testkube.TestSuiteStepExecutionRequest, executionRequest testkube.ExecutionRequest) testkube.ExecutionRequest {
	if stepRequest == nil {
		return executionRequest
	}
	if stepRequest.ExecutionLabels != nil {
		executionRequest.ExecutionLabels = stepRequest.ExecutionLabels
	}

	if stepRequest.Variables != nil {
		executionRequest.Variables = mergeVariables(executionRequest.Variables, stepRequest.Variables)
	}

	if len(stepRequest.Args) != 0 {
		if stepRequest.ArgsMode == string(testkube.ArgsModeTypeAppend) || stepRequest.ArgsMode == "" {
			executionRequest.Args = append(executionRequest.Args, stepRequest.Args...)
		}

		if stepRequest.ArgsMode == string(testkube.ArgsModeTypeOverride) || stepRequest.ArgsMode == string(testkube.ArgsModeTypeReplace) {
			executionRequest.Args = stepRequest.Args
		}
	}

	if stepRequest.Command != nil {
		executionRequest.Command = stepRequest.Command
	}
	executionRequest.HttpProxy = setStringField(executionRequest.HttpProxy, stepRequest.HttpProxy)
	executionRequest.HttpsProxy = setStringField(executionRequest.HttpsProxy, stepRequest.HttpsProxy)
	executionRequest.CronJobTemplate = setStringField(executionRequest.CronJobTemplate, stepRequest.CronJobTemplate)
	executionRequest.CronJobTemplateReference = setStringField(executionRequest.CronJobTemplateReference, stepRequest.CronJobTemplateReference)
	executionRequest.JobTemplate = setStringField(executionRequest.JobTemplate, stepRequest.JobTemplate)
	executionRequest.JobTemplateReference = setStringField(executionRequest.JobTemplateReference, stepRequest.JobTemplateReference)
	executionRequest.ScraperTemplate = setStringField(executionRequest.ScraperTemplate, stepRequest.ScraperTemplate)
	executionRequest.ScraperTemplateReference = setStringField(executionRequest.ScraperTemplateReference, stepRequest.ScraperTemplateReference)
	executionRequest.PvcTemplate = setStringField(executionRequest.PvcTemplate, stepRequest.PvcTemplate)
	executionRequest.PvcTemplateReference = setStringField(executionRequest.PvcTemplate, stepRequest.PvcTemplateReference)

	if stepRequest.RunningContext != nil {
		executionRequest.RunningContext = &testkube.RunningContext{
			Type_:   string(stepRequest.RunningContext.Type_),
			Context: stepRequest.RunningContext.Context,
		}
	}

	return executionRequest
}

func setStringField(oldValue string, newValue string) string {
	if newValue != "" {
		return newValue
	}
	return oldValue
}
