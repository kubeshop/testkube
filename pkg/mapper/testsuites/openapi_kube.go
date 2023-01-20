package testsuites

import (
	testsuitesv3 "github.com/kubeshop/testkube-operator/apis/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					Batch: []testkube.TestSuiteStepExecutionSummary{mapStepResultToStepExecutionSummary(stepResult)},
				}
			}
		}

		if len(execution.BatchStepResults) != 0 {
			executionsSummary = make([]testkube.TestSuiteBatchStepExecutionSummary, len(execution.BatchStepResults))
			for j, stepResult := range execution.BatchStepResults {
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

func mapBatchStepResultToExecutionSummary(r testkube.TestSuiteBatchStepExecutionResult) testkube.TestSuiteBatchStepExecutionSummary {
	batch := make([]testkube.TestSuiteStepExecutionSummary, len(r.Batch))
	for i := range r.Batch {
		batch[i] = mapStepResultToStepExecutionSummary(r.Batch[i])
	}

	return testkube.TestSuiteBatchStepExecutionSummary{
		Batch: batch,
	}
}

func MapTestSuiteUpsertRequestToTestCRD(request testkube.TestSuiteUpsertRequest) testsuitesv3.TestSuite {
	return testsuitesv3.TestSuite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsuitesv3.TestSuiteSpec{
			Repeats:          int(request.Repeats),
			Description:      request.Description,
			Before:           mapTestBatchStepsToCRD(request.Before),
			Steps:            mapTestBatchStepsToCRD(request.Steps),
			After:            mapTestBatchStepsToCRD(request.After),
			Schedule:         request.Schedule,
			ExecutionRequest: MapExecutionRequestToSpecExecutionRequest(request.ExecutionRequest),
		},
	}
}

func mapTestBatchStepsToCRD(batches []testkube.TestSuiteBatchStep) (out []testsuitesv3.TestSuiteBatchStep) {
	for _, batch := range batches {
		steps := make([]testsuitesv3.TestSuiteStepSpec, len(batch.Batch))
		for i := range batch.Batch {
			steps[i] = mapTestStepToCRD(batch.Batch[i])
		}

		out = append(out, testsuitesv3.TestSuiteBatchStep{
			StopOnFailure: batch.StopOnFailure,
			Batch:         steps,
		})
	}

	return out
}

func mapTestStepToCRD(step testkube.TestSuiteStep) (stepSpec testsuitesv3.TestSuiteStepSpec) {
	switch step.Type() {

	case testkube.TestSuiteStepTypeDelay:
		stepSpec.Delay = &testsuitesv3.TestSuiteStepDelay{
			Duration: step.Delay.Duration,
		}

	case testkube.TestSuiteStepTypeExecuteTest:
		s := step.Execute
		stepSpec.Execute = &testsuitesv3.TestSuiteStepExecute{
			Namespace: s.Namespace,
			Name:      s.Name,
		}
	}

	return
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
	}
}

// MapTestSuiteUpsertRequestToTestCRD maps TestSuiteUpdateRequest OpenAPI spec to TestSuite CRD spec
func MapTestSuiteUpdateRequestToTestCRD(request testkube.TestSuiteUpdateRequest, testSuite *testsuitesv3.TestSuite) *testsuitesv3.TestSuite {
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

	if request.Before != nil {
		testSuite.Spec.Before = mapTestBatchStepsToCRD(*request.Before)
	}

	if request.Steps != nil {
		testSuite.Spec.Steps = mapTestBatchStepsToCRD(*request.Steps)
	}

	if request.After != nil {
		testSuite.Spec.After = mapTestBatchStepsToCRD(*request.After)
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

	return testSuite
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
