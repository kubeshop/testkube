package newman

import (
	"time"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

func MapMetadataToResult(metadata ExecutionJSONResult) kubetest.ExecutionResult {

	startTime := time.Unix(metadata.Run.Timings.Started, 0)
	endTime := time.Unix(metadata.Run.Timings.Completed, 0)

	status := "success"
	if len(metadata.Run.Failures) > 0 {
		status = "failed"
	}

	result := kubetest.ExecutionResult{
		StartTime: startTime,
		EndTime:   endTime,
		Status:    status,
	}

	return result
}
