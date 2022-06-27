package testsuite

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"go.uber.org/zap"

	testsuitesv1 "github.com/kubeshop/testkube-operator/apis/testsuite/v1"
	testsuitesclientv1 "github.com/kubeshop/testkube-operator/client/testsuites/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cronjob"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

const (
	// testResourceURI is test resource uri for cron job call
	testResourceURI = "tests"
	// testSuiteResourceURI is test suite resource uri for cron job call
	testSuiteResourceURI = "test-suites"
	// defaultConcurrencyLevel is a default concurrency level for worker pool
	defaultConcurrencyLevel = "10"
)

type TestsuitesService struct {
	TestsSuitesClient    *testsuitesclientv1.TestSuitesClient
	TestExecutionResults testresult.Repository
	CronJobClient        *cronjob.Client
	Log                  *zap.SugaredLogger
}

func (s TestsuitesService) Run(
	ctx context.Context,
	request testkube.TestSuiteExecutionRequest,
	name, namespace, selector, callback string,
	concurrencyLevel int) (results []testkube.TestSuiteExecution, err error) {

	s.Log.Debugw("getting test suite", "name", name, "selector", selector)

	var testSuites []testsuitesv1.TestSuite
	if name != "" {
		testSuite, err := s.TestsSuitesClient.Get(name)
		if err != nil {
			return results, err
		}

		testSuites = append(testSuites, *testSuite)
	} else {
		testSuiteList, err := s.TestsSuitesClient.List(selector)
		if err != nil {
			return results, err
		}

		testSuites = append(testSuites, testSuiteList.Items...)
	}

	var work []testsuitesv1.TestSuite
	for _, testSuite := range testSuites {
		if testSuite.Spec.Schedule == "" || c.Query("callback") != "" {
			work = append(work, testSuite)
			continue
		}

		data, err := json.Marshal(request)
		if err != nil {
			return results, err
		}

		options := cronjob.CronJobOptions{
			Schedule: testSuite.Spec.Schedule,
			Resource: testSuiteResourceURI,
			Data:     string(data),
			Labels:   testSuite.Labels,
		}
		if err = s.CronJobClient.Apply(testSuite.Name, cronjob.GetMetadataName(testSuite.Name, testSuiteResourceURI), options); err != nil {
			return results, err
		}

		results = append(results, testkube.NewQueuedTestSuiteExecution(name, namespace))
	}

	if len(work) != 0 {
		workerpoolService := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](concurrencyLevel)

		go workerpoolService.SendRequests(s.prepareTestSuiteRequests(work, request))
		go workerpoolService.Run(ctx)

		for r := range workerpoolService.GetResponses() {
			results = append(results, r.Result)
		}
	}

	s.Log.Debugw("executing test", "name", name, "selector", selector)
	if name != "" && len(results) != 0 {
		if results[0].IsFailed() {
			return results, fmt.Errorf("test suite failed %v", name)
		}

		return results, nil
	}

	return results, nil
}

func (s TestsuitesService) prepareTestSuiteRequests(work []testsuitesv1.TestSuite, request testkube.TestSuiteExecutionRequest) []workerpool.Request[
	testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution] {
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

func (s TestsuitesService) executeTestSuite(ctx context.Context, testSuite testkube.TestSuite, request testkube.TestSuiteExecutionRequest) (
	testsuiteExecution testkube.TestSuiteExecution, err error) {
	s.Log.Debugw("Got test to execute", "test", testSuite)

	testsuiteExecution = testkube.NewStartedTestSuiteExecution(testSuite, request)
	err = s.TestExecutionResults.Insert(ctx, testsuiteExecution)
	if err != nil {
		s.Log.Infow("Inserting test execution", "error", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(testsuiteExecution *testkube.TestSuiteExecution, request testkube.TestSuiteExecutionRequest) {
		defer func(testExecution *testkube.TestSuiteExecution) {
			duration := testExecution.CalculateDuration()
			testExecution.EndTime = time.Now()
			testExecution.Duration = duration.String()

			err = s.TestExecutionResults.EndExecution(ctx, testExecution.Id, testExecution.EndTime, duration)
			if err != nil {
				s.Log.Errorw("error setting end time", "error", err.Error())
			}

			wg.Done()
		}(testsuiteExecution)

		hasFailedSteps := false
		cancellSteps := false
		for i := range testsuiteExecution.StepResults {
			if cancellSteps {
				testsuiteExecution.StepResults[i].Execution.ExecutionResult.Cancel()
				continue
			}

			// start execution of given step
			testsuiteExecution.StepResults[i].Execution.ExecutionResult.InProgress()
			err = s.TestExecutionResults.Update(ctx, *testsuiteExecution)
			if err != nil {
				s.Log.Infow("Updating test execution", "error", err)
			}

			s.executeTestStep(ctx, *testsuiteExecution, request, &testsuiteExecution.StepResults[i])

			err := s.TestExecutionResults.Update(ctx, *testsuiteExecution)
			if err != nil {
				hasFailedSteps = true
				s.Log.Errorw("saving test suite execution results error", "error", err)
				continue
			}

			if testsuiteExecution.StepResults[i].IsFailed() {
				hasFailedSteps = true
				if testsuiteExecution.StepResults[i].Step.StopTestOnFailure {
					cancellSteps = true
					continue
				}
			}
		}

		testsuiteExecution.Status = testkube.TestSuiteExecutionStatusPassed
		if hasFailedSteps {
			testsuiteExecution.Status = testkube.TestSuiteExecutionStatusFailed
		}

		s.Metrics.IncExecuteTestSuite(*testsuiteExecution)

		err := s.TestExecutionResults.Update(ctx, *testsuiteExecution)
		if err != nil {
			s.Log.Errorw("saving final test suite execution result error", "error", err)
		}

	}(&testsuiteExecution, request)

	// wait for sync test suite execution
	if request.Sync {
		wg.Wait()
	}

	return testsuiteExecution, nil
}
