package executions

import "github.com/kubeshop/kubtest/pkg/api/v1/kubtest"

func MapToSummary(executions []kubtest.Execution) []kubtest.ExecutionSummary {
	result := make([]kubtest.ExecutionSummary, len(executions))
	for i, s := range executions {
		result[i] = kubtest.ExecutionSummary{
			Id:         s.Id,
			Name:       s.Name,
			ScriptName: s.ScriptName,
			ScriptType: s.ScriptType,
			Status:     s.ExecutionResult.Status,
			StartTime:  s.ExecutionResult.StartTime,
			EndTime:    s.ExecutionResult.EndTime,
		}
	}

	return result
}
