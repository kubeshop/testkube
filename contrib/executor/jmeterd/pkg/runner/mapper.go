package runner

import (
	"fmt"

	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/parser"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func mapResultsToExecutionResults(out []byte, results parser.Results) (result testkube.ExecutionResult) {
	result.Status = testkube.ExecutionStatusPassed
	if results.HasError {
		result.Status = testkube.ExecutionStatusFailed
		result.ErrorMessage = results.LastErrorMessage
	}

	result.Output = string(out)
	result.OutputType = "text/plain"

	for _, r := range results.Results {
		result.Steps = append(
			result.Steps,
			testkube.ExecutionStepResult{
				Name:     r.Label,
				Duration: r.Duration.String(),
				Status:   mapResultStatus(r),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: mapResultStatus(r),
				}},
			})
	}

	return result
}

func mapTestResultsToExecutionResults(out []byte, results parser.TestResults) (result testkube.ExecutionResult) {
	result.Status = testkube.ExecutionStatusPassed

	result.Output = string(out)
	result.OutputType = "text/plain"

	samples := append(results.HTTPSamples, results.Samples...)
	for _, r := range samples {
		if !r.Success {
			result.Status = testkube.ExecutionStatusFailed
			if r.AssertionResult != nil {
				result.ErrorMessage = r.AssertionResult.FailureMessage
			}
		}

		result.Steps = append(
			result.Steps,
			testkube.ExecutionStepResult{
				Name:     r.Label,
				Duration: fmt.Sprintf("%dms", r.Time),
				Status:   mapTestResultStatus(r.Success),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: mapTestResultStatus(r.Success),
				}},
			})
	}

	return result
}

func mapResultStatus(result parser.Result) string {
	if result.Success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}

func mapTestResultStatus(success bool) string {
	if success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}
