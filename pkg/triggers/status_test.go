package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

func TestTriggerStatusForTestWorkflows(t *testing.T) {

	status := newTriggerStatus(&v1.TestTrigger{})

	status.testWorkflowExecutionIDs = []string{"test-workflow-execution-1"}
	status.start()

	assert.True(t, status.hasActiveTests())
	assert.NotNil(t, status.lastExecutionStarted)
	assert.Nil(t, status.lastExecutionFinished)

	status.done()
	status.removeTestWorkflowExecutionID("test-workflow-execution-1")

	assert.False(t, status.hasActiveTests())
	assert.NotNil(t, status.lastExecutionStarted)
	assert.NotNil(t, status.lastExecutionFinished)
	assert.True(t, status.lastExecutionFinished.After(*status.lastExecutionStarted))
}
