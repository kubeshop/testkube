package runner

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {

	t.Run("runner should return error if Tracetest endpoint is not provided", func(t *testing.T) {
		// given
		runner, err := NewRunner()
		execution := testkube.NewQueuedExecution()
		execution.Content = testkube.NewStringTestContent("hello I'm test content")

		// when
		_, err = runner.Run(*execution)

		// then
		assert.Error(t, err)
		assert.Equal(t, "TRACETEST_ENDPOINT variable was not found", err.Error())
	})

}
