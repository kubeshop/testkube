package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStopReason_Sentence(t *testing.T) {
	tests := []struct {
		name   string
		reason StopReason
		want   string
	}{
		{name: "abort all", reason: StopReasonAbortAll, want: "all executions of the workflow were stopped"},
		{name: "superseded", reason: StopReasonSuperseded, want: "a newer commit superseded this run"},
		{name: "queue timeout", reason: StopReasonQueueTimeout, want: "the execution exceeded the queue timeout of the workflow"},
		{name: "queued too long", reason: StopReasonQueuedTooLong, want: "the execution stayed queued for too long"},
		{name: "transition timeout", reason: StopReasonTransitionTimeout, want: "the execution stayed in a transitional state for too long"},
		{name: "stop not confirmed", reason: StopReasonStopNotConfirmed, want: "the runner did not confirm the stop in time"},
		{name: "execution timeout", reason: StopReasonExecutionTimeout, want: "the execution ran for too long"},
		{name: "execution stuck", reason: StopReasonExecutionStuck, want: "the execution is stuck in the running state"},
		{name: "worker resume failed", reason: StopReasonWorkerResumeFailed, want: "the parallel worker could not be resumed"},
		{name: "token from a newer control plane has no words yet", reason: StopReason("later-added"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.reason.Sentence())
		})
	}
}
