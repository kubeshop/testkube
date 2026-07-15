package testworkflows

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
)

func TestTestWorkflowNotFoundError(t *testing.T) {
	t.Run("returns CLIError with hint for not found errors", func(t *testing.T) {
		notFound := fmt.Errorf("problem: no test workflow found my-wf: %w", &client.HTTPStatusError{StatusCode: http.StatusNotFound})

		cliErr := testWorkflowNotFoundError("my-wf", "testkube", notFound)

		assert.NotNil(t, cliErr)
		assert.Equal(t, common.TKErrResourceNotFound, cliErr.Code)
		assert.Contains(t, cliErr.Title, "my-wf")
		assert.Contains(t, cliErr.MoreInfo, "kubectl get testworkflows")
		assert.Contains(t, cliErr.MoreInfo, "testkube")
		assert.Contains(t, cliErr.MoreInfo, "Control Plane")
	})

	t.Run("returns nil for non-404 errors", func(t *testing.T) {
		serverErr := fmt.Errorf("problem: boom: %w", &client.HTTPStatusError{StatusCode: http.StatusInternalServerError})

		assert.Nil(t, testWorkflowNotFoundError("my-wf", "testkube", serverErr))
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		assert.Nil(t, testWorkflowNotFoundError("my-wf", "testkube", nil))
	})
}
