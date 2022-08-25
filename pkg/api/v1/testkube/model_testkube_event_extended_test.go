package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmitter_IsValidEvent(t *testing.T) {
	t.Run("should pass only events with given selector", func(t *testing.T) {
		// given
		execution := NewQueuedExecution()
		execution.Labels = map[string]string{"test": "1"}
		e := TestkubeEvent{Type_: TestkubeEventStartTest, Execution: execution}

		// when
		valid := e.Valid("test=1")

		// then
		assert.True(t, valid)
	})
}
