package testsuites

import (
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO move to testuites mapper
func MapToTestExecutionSummary(executions []testkube.TestSuiteExecution) []testkube.TestSuiteExecutionSummary {
	result := make([]testkube.TestSuiteExecutionSummary, len(executions))

	for i, execution := range executions {
		executionsSummary := make([]testkube.TestSuiteStepExecutionSummary, len(execution.StepResults))
		for j, stepResult := range execution.StepResults {
			executionsSummary[j] = mapStepResultToExecutionSummary(stepResult)
		}

		result[i] = testkube.TestSuiteExecutionSummary{
			Id:            execution.Id,
			Name:          execution.Name,
			TestSuiteName: execution.TestSuite.Name,
			Status:        execution.Status,
			StartTime:     execution.StartTime,
			EndTime:       execution.EndTime,
			Duration:      types.FormatDuration(execution.Duration),
			Execution:     executionsSummary,
			Labels:        execution.Labels,
		}
	}

	return result
}

func mapStepResultToExecutionSummary(r testkube.TestSuiteStepExecutionResult) testkube.TestSuiteStepExecutionSummary {
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

func MapTestSuiteUpsertRequestToTestCRD(request testkube.TestSuiteUpsertRequest) testsuitesv2.TestSuite {
	return testsuitesv2.TestSuite{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsuitesv2.TestSuiteSpec{
			Repeats:          int(request.Repeats),
			Description:      request.Description,
			Before:           mapTestStepsToCRD(request.Before),
			Steps:            mapTestStepsToCRD(request.Steps),
			After:            mapTestStepsToCRD(request.After),
			Schedule:         request.Schedule,
			ExecutionRequest: MapExecutionRequestToSpecExecutionRequest(request.ExecutionRequest),
		},
	}
}

func mapTestStepsToCRD(steps []testkube.TestSuiteStep) (out []testsuitesv2.TestSuiteStepSpec) {
	for _, step := range steps {
		out = append(out, mapTestStepToCRD(step))
	}

	return
}

func mapTestStepToCRD(step testkube.TestSuiteStep) (stepSpec testsuitesv2.TestSuiteStepSpec) {
	switch step.Type() {

	case testkube.TestSuiteStepTypeDelay:
		stepSpec.Delay = &testsuitesv2.TestSuiteStepDelay{
			Duration: step.Delay.Duration,
		}

	case testkube.TestSuiteStepTypeExecuteTest:
		s := step.Execute
		stepSpec.Execute = &testsuitesv2.TestSuiteStepExecute{
			Namespace: s.Namespace,
			Name:      s.Name,
			// TODO move StopOnFailure level up in operator model to mimic this one
			StopOnFailure: step.StopTestOnFailure,
		}
	}

	return
}

// MapExecutionRequestToSpecExecutionRequest maps ExecutionRequest OpenAPI spec to ExecutionRequest CRD spec
func MapExecutionRequestToSpecExecutionRequest(executionRequest *testkube.TestSuiteExecutionRequest) *testsuitesv2.TestSuiteExecutionRequest {
	if executionRequest == nil {
		return nil
	}

	return &testsuitesv2.TestSuiteExecutionRequest{
		Name:            executionRequest.Name,
		Labels:          executionRequest.Labels,
		ExecutionLabels: executionRequest.ExecutionLabels,
		Namespace:       executionRequest.Namespace,
		Variables:       MapCRDVariables(executionRequest.Variables),
		SecretUUID:      executionRequest.SecretUUID,
		Sync:            executionRequest.Sync,
		HttpProxy:       executionRequest.HttpProxy,
		HttpsProxy:      executionRequest.HttpsProxy,
	}
}
