package executions

import "github.com/kubeshop/kubtest/pkg/api/kubtest"

func MapToSummary(executions []kubtest.ScriptExecution) []kubtest.ExecutionSummary {
	result := make([]kubtest.ExecutionSummary, len(executions))
	for i, s := range executions {
		result[i] = kubtest.ExecutionSummary{
			Id:         s.Id,
			Name:       s.Name,
			ScriptName: s.ScriptName,
			ScriptType: s.ScriptType,
			Status:     s.Execution.Status,
			StartTime:  s.Execution.StartTime,
			EndTime:    s.Execution.EndTime,
		}
	}

	return result
}
