package newman

import (
	"time"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

func MapMetadataToResult(newmanResult NewmanExecutionResult) kubetest.ExecutionResult {

	startTime := time.Unix(newmanResult.Metadata.Run.Timings.Started, 0)
	endTime := time.Unix(newmanResult.Metadata.Run.Timings.Completed, 0)

	status := "success"
	if len(newmanResult.Metadata.Run.Failures) > 0 {
		status = "failed"
	}

	result := kubetest.ExecutionResult{
		RawOutput:     newmanResult.RawOutput,
		RawOutputType: "text/plain",
		StartTime:     startTime,
		EndTime:       endTime,
		Status:        status,
	}

	return result
}
