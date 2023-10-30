package newman

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapMetadataToResult(newmanResult NewmanExecutionResult) testkube.ExecutionResult {
	status := testkube.StatusPtr(testkube.PASSED_ExecutionStatus)
	if len(newmanResult.Metadata.Run.Failures) > 0 {
		status = testkube.StatusPtr(testkube.FAILED_ExecutionStatus)
	}

	result := testkube.ExecutionResult{
		Output:     newmanResult.Output,
		OutputType: "text/plain",
		Status:     status,
	}

	runHasFailedAssertions := false
	for _, execution := range newmanResult.Metadata.Run.Executions {

		duration := time.Duration(execution.Response.ResponseTime) * time.Millisecond
		step := testkube.ExecutionStepResult{
			Name:     execution.Item.Name,
			Status:   "passed",
			Duration: duration.String(),
		}

		executionHasFailedAssertions := false
		for _, assertion := range execution.Assertions {
			assertionResult := testkube.AssertionResult{
				Name:   assertion.Assertion,
				Status: "passed",
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
		result.Status = testkube.StatusPtr(testkube.FAILED_ExecutionStatus)
	}

	return result
}

func makeSuccessExecution(out []byte) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status
	result.Output = string(out)
	result.OutputType = "text/plain"

	return result
}
