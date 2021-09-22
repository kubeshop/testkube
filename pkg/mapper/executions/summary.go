package executions

import "github.com/kubeshop/kubtest/pkg/api/kubtest"

func MapToSummary(executions []kubtest.Execution) []kubtest.ExecutionSummary {
	result := make([]kubtest.ExecutionSummary, len(executions))
	for i, s := range executions {
		result[i] = kubtest.ExecutionSummary{
			Id:         s.Id,
			Name:       s.Name,
			ScriptName: s.ScriptName,
			ScriptType: s.ScriptType,
			Status:     s.Result.Status,
			StartTime:  s.Result.StartTime,
			EndTime:    s.Result.EndTime,
		}
	}

	return result
}
