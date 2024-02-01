package parser

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func mapCSVResultsToExecutionResults(out []byte, results CSVResults) (result testkube.ExecutionResult) {
	result = MakeSuccessExecution(out)
	// TODO: Is it enough to just disable it here?
	//if results.HasError {
	//	result.Status = testkube.ExecutionStatusFailed
	//	result.ErrorMessage = results.LastErrorMessage
	//}

	for _, r := range results.Results {
		result.Steps = append(
			result.Steps,
			testkube.ExecutionStepResult{
				Name:     r.Label,
				Duration: r.Duration.String(),
				Status:   mapCSVResultStatus(r),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: mapCSVResultStatus(r),
				}},
			})
	}

	return result
}

func mapCSVResultStatus(result CSVResult) string {
	if result.Success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}

func mapXMLResultsToExecutionResults(out []byte, results XMLResults) (result testkube.ExecutionResult) {
	result = MakeSuccessExecution(out)

	samples := append(results.HTTPSamples, results.Samples...)
	for _, r := range samples {
		// TODO: Is it enough to disable it here?
		//if !r.Success {
		//	result.Status = testkube.ExecutionStatusFailed
		//	if r.AssertionResult != nil {
		//		result.ErrorMessage = r.AssertionResult.FailureMessage
		//	}
		//}

		result.Steps = append(
			result.Steps,
			testkube.ExecutionStepResult{
				Name:     r.Label,
				Duration: fmt.Sprintf("%dms", r.Time),
				Status:   mapXMLResultStatus(r.Success),
				AssertionResults: []testkube.AssertionResult{{
					Name:   r.Label,
					Status: mapXMLResultStatus(r.Success),
				}},
			})
	}

	return result
}

func mapXMLResultStatus(success bool) string {
	if success {
		return string(testkube.PASSED_ExecutionStatus)
	}

	return string(testkube.FAILED_ExecutionStatus)
}

func MakeSuccessExecution(out []byte) (result testkube.ExecutionResult) {
	status := testkube.PASSED_ExecutionStatus
	result.Status = &status
	result.Output = string(out)
	result.OutputType = "text/plain"

	return result
}
