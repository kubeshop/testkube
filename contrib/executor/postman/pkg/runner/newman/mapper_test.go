package newman

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapNewmanMetadataToResult(t *testing.T) {

	t.Run("timings", func(t *testing.T) {

		newmanResult := NewmanExecutionResult{
			Output: "output",
			Metadata: ExecutionJSONResult{
				Run: Run{
					Timings: RunTimings{
						Started:   1,
						Completed: 60,
					},
					Failures: []Failure{},
				},
			},
		}

		result := MapMetadataToResult(newmanResult)

		assert.Equal(t, testkube.PASSED_ExecutionStatus, *result.Status)
		assert.Equal(t, newmanResult.Output, result.Output)
		assert.Equal(t, "text/plain", result.OutputType)
	})

	t.Run("check success result", func(t *testing.T) {

		newmanResult := NewmanExecutionResult{
			Output: "some text result",
			Metadata: ExecutionJSONResult{
				Run: Run{
					Timings: RunTimings{
						Started:   1,
						Completed: 60,
					},
					Failures: []Failure{},
				},
			},
		}

		result := MapMetadataToResult(newmanResult)

		assert.Equal(t, "passed", string(*result.Status), "no failures, expecting success status")
	})

	t.Run("check for failures", func(t *testing.T) {
		newmanResult := NewmanExecutionResult{
			Metadata: ExecutionJSONResult{
				Run: Run{
					Timings: RunTimings{
						Started:   1,
						Completed: 60,
					},
					Failures: []Failure{
						{
							Error: FailureError{
								Name:      "AssertionError",
								Index:     0,
								Test:      "Environment variables are set",
								Message:   "expected undefined to equal 'dupa'",
								Stack:     "AssertionError: expected undefined to equal 'dupa'\n   at Object.eval sandbox-script.js:1:1)",
								Checksum:  "85bbd591a93fc0aa946f5db3fe3033c3",
								ID:        "33d4aa87-7911-4b60-b545-4fc1e20d671d",
								Timestamp: 1628767471559,
							},
						},
					},
				},
			},
		}

		result := MapMetadataToResult(newmanResult)

		assert.Equal(t, "failed", string(*result.Status), "failure, expecting failed status")
	})

	t.Run("steps mappings", func(t *testing.T) {

		newmanResult := NewmanExecutionResult{
			Metadata: ExecutionJSONResult{
				Run: Run{
					Timings: RunTimings{
						Started:   1,
						Completed: 60,
					},
					Failures: []Failure{
						{
							Error: FailureError{
								Name:      "AssertionError",
								Index:     0,
								Test:      "Environment variables are set",
								Message:   "expected undefined to equal 'dupa'",
								Stack:     "AssertionError: expected undefined to equal 'dupa'\n   at Object.eval sandbox-script.js:1:1)",
								Checksum:  "85bbd591a93fc0aa946f5db3fe3033c3",
								ID:        "33d4aa87-7911-4b60-b545-4fc1e20d671d",
								Timestamp: 1628767471559,
							},
						},
					},

					Executions: []Execution{
						{
							Item: Item{
								Name: "Users details for use exu",
							},
							Assertions: []Assertion{
								{
									Assertion: "User details page renders correctly",
									Skipped:   false,
								},
								{
									Assertion: "User id should be greater than 0",
									Skipped:   false,
									Error: &RunError{
										Name:    "AssertionError",
										Index:   0,
										Test:    "User id should be greater than 0",
										Message: "expected undefined to be greater than 0",
										Stack:   "AssertionError: expected undefined to be greater than 0\n   at Object.eval sandbox-script.js:1:1)",
									},
								},
							},
						},
						{
							Item: Item{
								Name: "User friends list",
							},
							Assertions: []Assertion{
								{
									Assertion: "List should have user phone",
									Skipped:   false,
									Error: &RunError{
										Name:    "AssertionError",
										Index:   0,
										Test:    "Phone exists on list",
										Message: "can't find phone pattern on list",
										Stack:   "AssertionError: can't find phone pattern on list\n   at Object.eval sandbox-script.js:1:1)",
									},
								},
							},
						},
						{
							Item: Item{
								Name: "User friends list",
							},
							Assertions: []Assertion{
								{
									Assertion: "User should have at least one friend added",
									Skipped:   false,
								},
								{
									Assertion: "List should be visible",
									Skipped:   false,
								},
							},
						},
					},
				},
			},
		}

		result := MapMetadataToResult(newmanResult)

		assert.Equal(t, "failed", string(*result.Status), "expecting failed status")
		assert.Equal(t, "failed", result.Steps[0].Status)
		assert.Equal(t, "failed", result.Steps[1].Status)
		assert.Equal(t, "passed", result.Steps[2].Status)
	})

}
