package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

func TestNewMetricsRecorderConfig_SkipFlag(t *testing.T) {
	t.Run("Skip is false when DisableResourceMetrics is false and skip arg is false", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = false

		cfg := newMetricsRecorderConfig("step1", false, testworkflowconfig.ContainerResourceConfig{})

		assert.False(t, cfg.Skip)
	})

	t.Run("Skip is true when DisableResourceMetrics is true regardless of skip arg", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = true

		cfg := newMetricsRecorderConfig("step1", false, testworkflowconfig.ContainerResourceConfig{})

		assert.True(t, cfg.Skip)
	})

	t.Run("Skip is true when skip arg is true regardless of DisableResourceMetrics", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = false

		cfg := newMetricsRecorderConfig("step1", true, testworkflowconfig.ContainerResourceConfig{})

		assert.True(t, cfg.Skip)
	})

	t.Run("Skip is true when both DisableResourceMetrics and skip arg are true", func(t *testing.T) {
		data.ClearState()
		s := data.GetState()
		s.InternalConfig.Worker.DisableResourceMetrics = true

		cfg := newMetricsRecorderConfig("step1", true, testworkflowconfig.ContainerResourceConfig{})

		assert.True(t, cfg.Skip)
	})
}
