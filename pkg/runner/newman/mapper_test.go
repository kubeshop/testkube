package newman

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapNewmanMetadataToResult(t *testing.T) {

	t.Run("map basic data", func(t *testing.T) {

		newmanResult := NewmanExecutionResult{
			RawOutput: "some text result",
			Metadata: ExecutionJSONResult{
				Run: Run{
					Timings: RunTimings{
						Started:   1,
						Completed: 60,
					},
					// No failures
					Failures: []Failure{},
				},
			},
		}

		result := MapMetadataToResult(newmanResult)

		assert.Equal(t, "success", result.Status, "no failures, expecting success status")
		assert.Equal(t, time.Unix(1, 0), result.StartTime)
		assert.Equal(t, time.Unix(60, 0), result.EndTime)
	})

	t.Run("check for failures", func(t *testing.T) {
		newmanResult := NewmanExecutionResult{
			Metadata: ExecutionJSONResult{
				Run: Run{
					Timings: RunTimings{
						Started:   1,
						Completed: 60,
					},
					// No failures
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

		assert.Equal(t, "failed", result.Status, "failure, expecting failed status")
		assert.Equal(t, time.Unix(1, 0), result.StartTime)
		assert.Equal(t, time.Unix(60, 0), result.EndTime)
	})

}
