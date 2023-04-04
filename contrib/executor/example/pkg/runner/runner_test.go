package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("successful result", func(t *testing.T) {
		runner := NewRunner()
		res, err := runner.Run(
			ctx,
			testkube.Execution{
				Content: &testkube.TestContent{
					Uri: "https://testkube.io",
				},
			})

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusPassed, res.Status)
	})

	t.Run("failed 404 result", func(t *testing.T) {
		runner := NewRunner()
		res, err := runner.Run(
			ctx,
			testkube.Execution{
				Content: &testkube.TestContent{
					Uri: "https://testkube.io/some-non-existing-uri-blablablabl",
				},
			})

		assert.NoError(t, err)
		assert.Equal(t, testkube.ExecutionStatusFailed, res.Status)

	})

	t.Run("network connection issues returns errors", func(t *testing.T) {
		runner := NewRunner()
		_, err := runner.Run(
			ctx,
			testkube.Execution{
				Content: &testkube.TestContent{
					Uri: "blabla://non-existing-uri",
				},
			})

		assert.Error(t, err)
	})

}
