package executions

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// MapToSummary maps testkube.Executions to array of testkube.ExecutionSummary for lists without so many details
func MapToSummary(executions []testkube.Execution) []testkube.ExecutionSummary {
	result := make([]testkube.ExecutionSummary, len(executions))
	for i, s := range executions {
		result[i] = testkube.ExecutionSummary{
			Id:        s.Id,
			Name:      s.Name,
			TestName:  s.TestName,
			TestType:  s.TestType,
			Status:    s.ExecutionResult.Status,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
		}
	}

	return result
}
