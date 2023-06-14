package testsuiteexecutions

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// MapToSummary maps testkube.TestSuiteExecution to testkube.TestSuiteExecutionSummary for lists without so many details.
func MapToSummary(execution *testkube.TestSuiteExecution) *testkube.TestSuiteExecutionSummary {
	var testSuiteName string
	if execution.TestSuite != nil {
		testSuiteName = execution.TestSuite.Name
	}

	var summary []testkube.TestSuiteBatchStepExecutionSummary

	if len(execution.StepResults) != 0 {
		summary = make([]testkube.TestSuiteBatchStepExecutionSummary, len(execution.StepResults))
		for i, step := range execution.StepResults {
			summary[i] = testkube.TestSuiteBatchStepExecutionSummary{
				Execute: []testkube.TestSuiteStepExecutionSummary{
					mapStepExecutionResultV2ToExecutionSummary(step),
				},
			}
		}
	}

	if len(execution.ExecuteStepResults) != 0 {
		summary = make([]testkube.TestSuiteBatchStepExecutionSummary, len(execution.ExecuteStepResults))
		for i, step := range execution.ExecuteStepResults {
			summary[i] = mapBatchStepExecutionResultToExecutionSummary(step)
		}
	}

	return &testkube.TestSuiteExecutionSummary{
		Id:            execution.Id,
		Name:          execution.Name,
		TestSuiteName: testSuiteName,
		Status:        execution.Status,
		StartTime:     execution.StartTime,
		EndTime:       execution.EndTime,
		Duration:      execution.Duration,
		DurationMs:    execution.DurationMs,
		Execution:     summary,
		Labels:        execution.Labels,
	}
}

func mapStepExecutionResultV2ToExecutionSummary(step testkube.TestSuiteStepExecutionResultV2) testkube.TestSuiteStepExecutionSummary {
	var id, name, testName string
	var status *testkube.ExecutionStatus
	var tp *testkube.TestSuiteStepType
	if step.Execution != nil {
		id = step.Execution.Id
		name = step.Execution.Name
		testName = step.Execution.TestName

		if step.Execution.ExecutionResult != nil {
			status = step.Execution.ExecutionResult.Status
		}
	}

	if step.Step != nil {
		if step.Step.Execute != nil {
			tp = testkube.TestSuiteStepTypeExecuteTest
		}

		if step.Step.Delay != nil {
			tp = testkube.TestSuiteStepTypeDelay
		}
	}

	return testkube.TestSuiteStepExecutionSummary{
		Id:       id,
		Name:     name,
		TestName: testName,
		Status:   status,
		Type_:    tp,
	}
}

func mapBatchStepExecutionResultToExecutionSummary(step testkube.TestSuiteBatchStepExecutionResult) testkube.TestSuiteBatchStepExecutionSummary {
	batch := make([]testkube.TestSuiteStepExecutionSummary, len(step.Execute))
	for i, step := range step.Execute {
		batch[i] = mapStepExecutionResultToExecutionSummary(step)
	}

	return testkube.TestSuiteBatchStepExecutionSummary{
		Execute: batch,
	}
}

func mapStepExecutionResultToExecutionSummary(step testkube.TestSuiteStepExecutionResult) testkube.TestSuiteStepExecutionSummary {
	var id, name, testName string
	var status *testkube.ExecutionStatus
	var tp *testkube.TestSuiteStepType
	if step.Execution != nil {
		id = step.Execution.Id
		name = step.Execution.Name
		testName = step.Execution.TestName

		if step.Execution.ExecutionResult != nil {
			status = step.Execution.ExecutionResult.Status
		}
	}

	if step.Step != nil {
		if step.Step.Test != "" {
			tp = testkube.TestSuiteStepTypeExecuteTest
		}

		if step.Step.Delay != "" {
			tp = testkube.TestSuiteStepTypeDelay
		}
	}

	return testkube.TestSuiteStepExecutionSummary{
		Id:       id,
		Name:     name,
		TestName: testName,
		Status:   status,
		Type_:    tp,
	}
}
