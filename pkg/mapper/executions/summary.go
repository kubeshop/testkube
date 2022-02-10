package executions

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

func MapToSummary(executions []testkube.Execution) []testkube.ExecutionSummary {
	result := make([]testkube.ExecutionSummary, len(executions))
	for i, s := range executions {
		result[i] = testkube.ExecutionSummary{
			Id:        s.Id,
			Name:      s.Name,
			TestName:  s.ScriptName,
			TestType:  s.ScriptType,
			Status:    s.ExecutionResult.Status,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
		}
	}

	return result
}
