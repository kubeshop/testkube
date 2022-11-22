package testsuiteexecutions

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// MapToSummary maps testkube.TestSuiteExecution to testkube.TestSuiteExecutionSummary for lists without so many details.
func MapToSummary(execution *testkube.TestSuiteExecution) *testkube.TestSuiteExecutionSummary {
	var testSuiteName string
	if execution.TestSuite != nil {
		testSuiteName = execution.TestSuite.Name
	}

	var summary []testkube.TestSuiteStepExecutionSummary
	for _, step := range execution.StepResults {
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

		summary = append(summary, testkube.TestSuiteStepExecutionSummary{
			Id:       id,
			Name:     name,
			TestName: testName,
			Status:   status,
			Type_:    tp,
		})
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
