package executions

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// MapToSummary maps testkube.Execution to testkube.ExecutionSummary for lists without so many details.
func MapToSummary(execution *testkube.Execution) *testkube.ExecutionSummary {
	var status *testkube.ExecutionStatus
	if execution.ExecutionResult != nil {
		status = execution.ExecutionResult.Status
	}

	return &testkube.ExecutionSummary{
		Id:        execution.Id,
		Name:      execution.Name,
		Number:    execution.Number,
		TestName:  execution.TestName,
		TestType:  execution.TestType,
		Status:    status,
		StartTime: execution.StartTime,
		EndTime:   execution.EndTime,
	}
}
