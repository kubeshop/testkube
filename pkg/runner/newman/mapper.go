package newman

import (
	"time"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

func MapMetadataToResult(newmanResult NewmanExecutionResult) kubtest.ExecutionResult {

	startTime := time.Unix(0, newmanResult.Metadata.Run.Timings.Started*int64(time.Millisecond))
	endTime := time.Unix(0, newmanResult.Metadata.Run.Timings.Completed*int64(time.Millisecond))

	status := "success"
	if len(newmanResult.Metadata.Run.Failures) > 0 {
		status = "failed"
	}

	result := kubtest.ExecutionResult{
		RawOutput:     newmanResult.RawOutput,
		RawOutputType: "text/plain",
		StartTime:     startTime,
		EndTime:       endTime,
		Status:        status,
	}

	runHasFailedAssertions := false
	for _, execution := range newmanResult.Metadata.Run.Executions {

		duration := time.Duration(execution.Response.ResponseTime) * time.Millisecond
		step := kubtest.ExecutionStepResult{
			Name:     execution.Item.Name,
			Status:   "success",
			Duration: duration.String(),
		}

		executionHasFailedAssertions := false
		for _, assertion := range execution.Assertions {
			assertionResult := kubtest.AssertionResult{
				Name:   assertion.Assertion,
				Status: "success",
			}

			if assertion.Error != nil {

				assertionResult.ErrorMessage = assertion.Error.Message
				assertionResult.Status = "failed"
				executionHasFailedAssertions = true
			}

			step.AssertionResults = append(step.AssertionResults, assertionResult)
		}

		if executionHasFailedAssertions {
			step.Status = "failed"
			runHasFailedAssertions = true
		}

		result.Steps = append(result.Steps, step)
	}

	if runHasFailedAssertions {
		result.Status = "failed"
	}

	return result
}
