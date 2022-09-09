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
		e := Event{Type_: EventStartTest, Execution: execution}

		// when
		valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.True(t, valid)
	})

	t.Run("should not pass events with not matching selector", func(t *testing.T) {
		// given
		execution := NewQueuedExecution()
		execution.Labels = map[string]string{"test": "2"}
		e := Event{Type_: EventStartTest, Execution: execution}

		// when
		valid := e.Valid("test=1", AllEventTypes)

		// then
		assert.False(t, valid)
	})

	t.Run("should pass events without selector", func(t *testing.T) {
		// given
		execution := NewQueuedExecution()
		e := Event{Type_: EventStartTest, Execution: execution}

		// when
		valid := e.Valid("", AllEventTypes)

		// then
		assert.True(t, valid)
	})
}
