package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriggerStatus(t *testing.T) {
	t.Parallel()

	status := newTriggerStatus()

	status.start()

	assert.True(t, status.ActiveTests)
	assert.NotNil(t, status.LastExecutionStarted)
	assert.Nil(t, status.LastExecutionFinished)

	status.finish()

	assert.False(t, status.ActiveTests)
	assert.NotNil(t, status.LastExecutionStarted)
	assert.NotNil(t, status.LastExecutionFinished)
	assert.True(t, status.LastExecutionFinished.After(*status.LastExecutionStarted))
}
