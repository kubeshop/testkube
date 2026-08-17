package controlplaneclient

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestStickyChildRunningContextType(t *testing.T) {
	t.Run("QualityLoop parent is sticky, child inherits QUALITYLOOP", func(t *testing.T) {
		got, ok := stickyChildRunningContextType(testkube.QUALITYLOOP_TestWorkflowRunningContextActorType)
		assert.True(t, ok)
		assert.Equal(t, cloud.RunningContextType_QUALITYLOOP, got)
	})

	t.Run("USER parent is not sticky, child falls through to EXECUTION default", func(t *testing.T) {
		_, ok := stickyChildRunningContextType(testkube.USER_TestWorkflowRunningContextActorType)
		assert.False(t, ok)
	})

	t.Run("CRON parent is not sticky", func(t *testing.T) {
		_, ok := stickyChildRunningContextType(testkube.CRON_TestWorkflowRunningContextActorType)
		assert.False(t, ok)
	})

	t.Run("TESTTRIGGER parent is not sticky", func(t *testing.T) {
		_, ok := stickyChildRunningContextType(testkube.TESTTRIGGER_TestWorkflowRunningContextActorType)
		assert.False(t, ok)
	})

	t.Run("empty parent actor type is not sticky", func(t *testing.T) {
		_, ok := stickyChildRunningContextType("")
		assert.False(t, ok)
	})
}
