package runner

import (
	"testing"

	"github.com/kubeshop/testkube/contrib/executor/jmeter/pkg/parser"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestMapStatus(t *testing.T) {

	t.Run("should map valid status", func(t *testing.T) {
		out := MapStatus(parser.Result{Success: false})
		assert.Equal(t, out, string(testkube.FAILED_ExecutionStatus))
	})

	t.Run("should map invalid status", func(t *testing.T) {
		out := MapStatus(parser.Result{Success: true})
		assert.Equal(t, out, string(testkube.PASSED_ExecutionStatus))
	})

}

func TestMapResultsToExecutionResults(t *testing.T) {

	t.Run("results are mapped to execution results", func(t *testing.T) {

		out := []byte("log output")
		results := parser.Results{
			HasError:         true,
			LastErrorMessage: "some error",
			Results: []parser.Result{
				{
					Success: false,
					Error:   "some error",
				},
			},
		}

		result := MapResultsToExecutionResults(out, results)

		assert.Equal(t, "log output", result.Output)
		assert.Equal(t, "some error", result.ErrorMessage)
	})

}
