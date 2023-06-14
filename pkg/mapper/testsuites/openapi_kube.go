package testsuites

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsuitesv3 "github.com/kubeshop/testkube-operator/apis/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/types"
)

// TODO move to testuites mapper
func MapToTestExecutionSummary(executions []testkube.TestSuiteExecution) []testkube.TestSuiteExecutionSummary {
	result := make([]testkube.TestSuiteExecutionSummary, len(executions))

	for i, execution := range executions {
		var executionsSummary []testkube.TestSuiteBatchStepExecutionSummary

		if len(execution.StepResults) != 0 {
			executionsSummary = make([]testkube.TestSuiteBatchStepExecutionSummary, len(execution.StepResults))
			for j, stepResult := range execution.StepResults {
				executionsSummary[j] = testkube.TestSuiteBatchStepExecutionSummary{
					Execute: []testkube.TestSuiteStepExecutionSummary{mapStepResultV2ToStepExecutionSummary(stepResult)},
				}
			}
		}

		if len(execution.ExecuteStepResults) != 0 {
			executionsSummary = make([]testkube.TestSuiteBatchStepExecutionSummary, len(execution.ExecuteStepResults))
			for j, stepResult := range execution.ExecuteStepResults {
				executionsSummary[j] = mapBatchStepResultToExecutionSummary(stepResult)
			}
		}

		result[i] = testkube.TestSuiteExecutionSummary{
			Id:            execution.Id,
			Name:          execution.Name,
			TestSuiteName: execution.TestSuite.Name,
			Status:        execution.Status,
			StartTime:     execution.StartTime,
			EndTime:       execution.EndTime,
			Duration:      types.FormatDuration(execution.Duration),
			DurationMs:    types.FormatDurationMs(execution.Duration),
			Execution:     executionsSummary,
			Labels:        execution.Labels,
		}
	}

	return result
}

func mapStepResultV2ToStepExecutionSummary(r testkube.TestSuiteStepExecutionResultV2) testkube.TestSuiteStepExecutionSummary {
	var id, testName, name string
	var status *testkube.ExecutionStatus = testkube.ExecutionStatusPassed
	var stepType *testkube.TestSuiteStepType

	if r.Test != nil {
		testName = r.Test.Name
	}

	if r.Execution != nil {
		id = r.Execution.Id
		if r.Execution.ExecutionResult != nil {
			status = r.Execution.ExecutionResult.Status
		}
	}

	if r.Step != nil {
		stepType = r.Step.Type()
		name = r.Step.FullName()
	}

	return testkube.TestSuiteStepExecutionSummary{
		Id:       id,
		Name:     name,
		TestName: testName,
		Status:   status,
		Type_:    stepType,
	}
}

func mapBatchStepResultToExecutionSummary(r testkube.TestSuiteBatchStepExecutionResult) testkube.TestSuiteBatchStepExecutionSummary {
	batch := make([]testkube.TestSuiteStepExecutionSummary, len(r.Execute))
	for i := range r.Execute {
		batch[i] = mapStepResultToStepExecutionSummary(r.Execute[i])
	}

	return testkube.TestSuiteBatchStepExecutionSummary{
		Execute: batch,
	}
}

func mapStepResultToStepExecutionSummary(r testkube.TestSuiteStepExecutionResult) testkube.TestSuiteStepExecutionSummary {
	var id, testName, name string
	var status *testkube.ExecutionStatus = testkube.ExecutionStatusPassed
	var stepType *testkube.TestSuiteStepType

	if r.Test != nil {
		testName = r.Test.Name
	}

	if r.Execution != nil {
		id = r.Execution.Id
		if r.Execution.ExecutionResult != nil {
			status = r.Execution.ExecutionResult.Status
		}
	}

	if r.Step != nil {
		stepType = r.Step.Type()
		name = r.Step.FullName()
	}

	return testkube.TestSuiteStepExecutionSummary{
		Id:       id,
		Name:     name,
		TestName: testName,
		Status:   status,
		Type_:    stepType,
	}
}

func MapTestSuiteUpsertRequestToTestCRD(request testkube.TestSuiteUpsertRequest) (testsuite testsuitesv3.TestSuite, err error) {
	before, err := mapTestBatchStepsToCRD(request.Before)
	if err != nil {
		return testsuite, err
	}

	steps, err := mapTestBatchStepsToCRD(request.Steps)
	if err != nil {
		return testsuite, err
	}

	after, err := mapTestBatchStepsToCRD(request.After)
	if err != nil {
		return testsuite, err
	}

	return testsuitesv3.TestSuite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsuitesv3.TestSuiteSpec{
			Repeats:          int(request.Repeats),
			Description:      request.Description,
			Before:           before,
			Steps:            steps,
			After:            after,
			Schedule:         request.Schedule,
			ExecutionRequest: MapExecutionRequestToSpecExecutionRequest(request.ExecutionRequest),
		},
	}, nil
}

func mapTestBatchStepsToCRD(batches []testkube.TestSuiteBatchStep) (out []testsuitesv3.TestSuiteBatchStep, err error) {
	for _, batch := range batches {
		steps := make([]testsuitesv3.TestSuiteStepSpec, len(batch.Execute))
		for i := range batch.Execute {
			steps[i], err = mapTestStepToCRD(batch.Execute[i])
			if err != nil {
				return nil, err
			}
		}

		out = append(out, testsuitesv3.TestSuiteBatchStep{
			StopOnFailure: batch.StopOnFailure,
			Execute:       steps,
		})
	}

	return out, nil
}

func mapTestStepToCRD(step testkube.TestSuiteStep) (stepSpec testsuitesv3.TestSuiteStepSpec, err error) {
	switch step.Type() {

	case testkube.TestSuiteStepTypeDelay:
		if step.Delay != "" {
			duration, err := time.ParseDuration(step.Delay)
			if err != nil {
				return stepSpec, err
			}

			stepSpec.Delay = metav1.Duration{Duration: duration}
		}
	case testkube.TestSuiteStepTypeExecuteTest:
		stepSpec.Test = step.Test
	}

	return stepSpec, nil
}

// MapExecutionRequestToSpecExecutionRequest maps ExecutionRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionRequestToSpecExecutionRequest(executionRequest *testkube.TestSuiteExecutionRequest) *testsuitesv3.TestSuiteExecutionRequest {
	if executionRequest == nil {
		return nil
	}

	return &testsuitesv3.TestSuiteExecutionRequest{
		Name:            executionRequest.Name,
		Labels:          executionRequest.Labels,
		ExecutionLabels: executionRequest.ExecutionLabels,
		Namespace:       executionRequest.Namespace,
		Variables:       MapCRDVariables(executionRequest.Variables),
		SecretUUID:      executionRequest.SecretUUID,
		Sync:            executionRequest.Sync,
		HttpProxy:       executionRequest.HttpProxy,
		HttpsProxy:      executionRequest.HttpsProxy,
		Timeout:         executionRequest.Timeout,
		CronJobTemplate: executionRequest.CronJobTemplate,
	}
}

// MapTestSuiteUpsertRequestToTestCRD maps TestSuiteUpdateRequest OpenAPI spec to TestSuite CRD spec
func MapTestSuiteUpdateRequestToTestCRD(request testkube.TestSuiteUpdateRequest,
	testSuite *testsuitesv3.TestSuite) (*testsuitesv3.TestSuite, error) {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&testSuite.Name,
		},
		{
			request.Namespace,
			&testSuite.Namespace,
		},
		{
			request.Description,
			&testSuite.Spec.Description,
		},
		{
			request.Schedule,
			&testSuite.Spec.Schedule,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	var err error
	if request.Before != nil {
		testSuite.Spec.Before, err = mapTestBatchStepsToCRD(*request.Before)
		if err != nil {
			return nil, err
		}
	}

	if request.Steps != nil {
		testSuite.Spec.Steps, err = mapTestBatchStepsToCRD(*request.Steps)
		if err != nil {
			return nil, err
		}
	}

	if request.After != nil {
		testSuite.Spec.After, err = mapTestBatchStepsToCRD(*request.After)
		if err != nil {
			return nil, err
		}
	}

	if request.Labels != nil {
		testSuite.Labels = *request.Labels
	}

	if request.Repeats != nil {
		testSuite.Spec.Repeats = int(*request.Repeats)
	}

	if request.ExecutionRequest != nil {
		testSuite.Spec.ExecutionRequest = MapExecutionUpdateRequestToSpecExecutionRequest(*request.ExecutionRequest, testSuite.Spec.ExecutionRequest)
	}

	return testSuite, nil
}

// MapExecutionUpdateRequestToSpecExecutionRequest maps ExecutionUpdateRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionUpdateRequestToSpecExecutionRequest(executionRequest *testkube.TestSuiteExecutionUpdateRequest,
	request *testsuitesv3.TestSuiteExecutionRequest) *testsuitesv3.TestSuiteExecutionRequest {
	if executionRequest == nil {
		return nil
	}

	if request == nil {
		request = &testsuitesv3.TestSuiteExecutionRequest{}
	}

	empty := true
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			executionRequest.Name,
			&request.Name,
		},
		{
			executionRequest.Namespace,
			&request.Namespace,
		},
		{
			executionRequest.SecretUUID,
			&request.SecretUUID,
		},
		{
			executionRequest.HttpProxy,
			&request.HttpProxy,
		},
		{
			executionRequest.HttpsProxy,
			&request.HttpsProxy,
		},
		{
			executionRequest.CronJobTemplate,
			&request.CronJobTemplate,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
			empty = false
		}
	}

	if executionRequest.Labels != nil {
		request.Labels = *executionRequest.Labels
		empty = false
	}

	if executionRequest.ExecutionLabels != nil {
		request.ExecutionLabels = *executionRequest.ExecutionLabels
		empty = false
	}

	if executionRequest.Sync != nil {
		request.Sync = *executionRequest.Sync
		empty = false
	}

	if executionRequest.Timeout != nil {
		request.Timeout = *executionRequest.Timeout
		empty = false
	}

	if executionRequest.Variables != nil {
		request.Variables = MapCRDVariables(*executionRequest.Variables)
		empty = false
	}

	if empty {
		return nil
	}

	return request
}

// MapStatusToSpec maps OpenAPI spec TestSuiteStatus to CRD
func MapStatusToSpec(testSuiteStatus *testkube.TestSuiteStatus) (specStatus testsuitesv3.TestSuiteStatus) {
	if testSuiteStatus == nil || testSuiteStatus.LatestExecution == nil {
		return specStatus
	}

	specStatus.LatestExecution = &testsuitesv3.TestSuiteExecutionCore{
		Id:     testSuiteStatus.LatestExecution.Id,
		Status: (*testsuitesv3.TestSuiteExecutionStatus)(testSuiteStatus.LatestExecution.Status),
	}

	specStatus.LatestExecution.StartTime.Time = testSuiteStatus.LatestExecution.StartTime
	specStatus.LatestExecution.EndTime.Time = testSuiteStatus.LatestExecution.EndTime

	return specStatus
}

// MapExecutionToTestSuiteStatus maps OpenAPI Execution to TestSuiteStatus CRD
func MapExecutionToTestSuiteStatus(execution *testkube.TestSuiteExecution) (specStatus testsuitesv3.TestSuiteStatus) {
	specStatus.LatestExecution = &testsuitesv3.TestSuiteExecutionCore{
		Id:     execution.Id,
		Status: (*testsuitesv3.TestSuiteExecutionStatus)(execution.Status),
	}

	specStatus.LatestExecution.StartTime.Time = execution.StartTime
	specStatus.LatestExecution.EndTime.Time = execution.EndTime

	return specStatus
}
