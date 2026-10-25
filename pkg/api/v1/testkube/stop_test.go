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

func TestStopCauses(t *testing.T) {
	written := &Cause{Reason: string(StopReasonVolumeMountFailed), Message: "secret \"x\" not found"}
	runnerStop := Stop{Actor: StopActorRunner, Reason: StopReasonInitTimeout}
	tests := []struct {
		name        string
		causes      *StopCauses
		wantWritten *Cause
		wantEnding  string
		wantLead    string
	}{
		{
			name: "no causes give nothing",
		},
		{
			name:        "the cause that the runner wrote into the step",
			causes:      &StopCauses{Written: map[string]*Cause{"": written}},
			wantWritten: written,
		},
		{
			name:       "the error that Kubernetes reported ends the execution",
			causes:     &StopCauses{Stop: runnerStop, Ending: "Job timed out after 60 seconds"},
			wantEnding: "Job timed out after 60 seconds",
			wantLead:   "Job timed out after 60 seconds",
		},
		{
			name:       "without an error in its own words, the free text of the stop ends the execution",
			causes:     &StopCauses{Stop: Stop{Actor: StopActorControlPlane, Reason: StopReasonExecutionTimeout, Detail: "after 1h"}},
			wantEnding: "after 1h",
			wantLead:   "after 1h",
		},
		{
			name:       "the error that Kubernetes reported wins over the free text of the stop",
			causes:     &StopCauses{Stop: Stop{Actor: StopActorControlPlane, Reason: StopReasonExecutionTimeout, Detail: "after 1h"}, Ending: "Job timed out after 3600 seconds"},
			wantEnding: "Job timed out after 3600 seconds",
			wantLead:   "Job timed out after 3600 seconds",
		},
		{
			name:     "without an ending, the reason of the stop leads the cause",
			causes:   &StopCauses{Stop: runnerStop},
			wantLead: "the first step did not start before the initialization timeout of the workflow",
		},
		{
			name:   "a stop that a person decided adds no lead, because the heading names the cancel",
			causes: &StopCauses{Stop: Stop{Actor: StopActorUser, Reason: StopReasonUserCancel}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantWritten, tt.causes.written(""))
			assert.Equal(t, tt.wantEnding, tt.causes.ending())
			assert.Equal(t, tt.wantLead, tt.causes.lead())
		})
	}
}
