package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestWriteDeclineCause(t *testing.T) {
	finished := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		result      *testkube.TestWorkflowResult
		reason      testkube.StartReason
		message     string
		wantMessage string
		wantReason  string
	}{
		{
			name:        "writes the message of the runner",
			result:      &testkube.TestWorkflowResult{Initialization: &testkube.TestWorkflowStepResult{}},
			reason:      testkube.StartReasonImagePullFailed,
			message:     "pull access denied for private/image",
			wantMessage: "pull access denied for private/image",
			wantReason:  string(testkube.StartReasonImagePullFailed),
		},
		{
			name:        "tells the user to upgrade a runner that sends no cause",
			result:      &testkube.TestWorkflowResult{Initialization: &testkube.TestWorkflowStepResult{}},
			wantMessage: "The runner did not send the cause. Upgrade the agent to see the cause here.",
		},
		{
			name:        "creates the initialization step when the result has none",
			result:      &testkube.TestWorkflowResult{},
			reason:      testkube.StartReasonJobCreateFailed,
			message:     "jobs.batch is forbidden",
			wantMessage: "jobs.batch is forbidden",
			wantReason:  string(testkube.StartReasonJobCreateFailed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.result.FinishedAt = finished

			writeDeclineCause(tt.result, tt.reason, tt.message)

			require.NotNil(t, tt.result.Initialization)
			assert.Equal(t, tt.wantMessage, tt.result.Initialization.ErrorMessage)
			assert.Equal(t, tt.wantReason, tt.result.Initialization.ErrorReason)
			assert.Equal(t, testkube.ABORTED_TestWorkflowStepStatus, *tt.result.Initialization.Status)
			assert.Equal(t, finished, tt.result.Initialization.FinishedAt)
		})
	}
}
