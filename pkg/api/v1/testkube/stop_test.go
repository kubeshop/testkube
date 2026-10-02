package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStop_Sentence(t *testing.T) {
	tests := []struct {
		name string
		stop Stop
		want string
	}{
		{
			name: "an actor and a reason",
			stop: Stop{Actor: StopActorRunner, Reason: StopReasonExecutionStuck},
			want: "by the runner: the execution is stuck in the running state",
		},
		{
			name: "an actor, a reason, and a detail",
			stop: Stop{Actor: StopActorRunner, Reason: StopReasonWorkerResumeFailed, Detail: "pod not found"},
			want: "by the runner: the parallel worker could not be resumed: pod not found",
		},
		{
			name: "an actor alone",
			stop: Stop{Actor: StopActorUser},
			want: "by the user",
		},
		{
			name: "a reason alone, as an older worker writes it",
			stop: Stop{Reason: "Job has been aborted by the system"},
			want: "Job has been aborted by the system",
		},
		{
			name: "a code that has no words keeps its raw text",
			stop: Stop{Actor: StopActorControlPlane, Reason: "later-added"},
			want: "by the control plane: later-added",
		},
		{
			name: "a detail alone says too little",
			stop: Stop{Detail: "pod not found"},
			want: "",
		},
		{
			name: "a stop that names nothing",
			stop: Stop{Code: string(ABORTED_TestWorkflowStatus)},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.stop.Sentence())
		})
	}
}
