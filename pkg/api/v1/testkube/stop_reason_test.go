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
		{name: "unschedulable", reason: StopReasonUnschedulable, want: "no node can run the pod"},
		{name: "config missing", reason: StopReasonConfigMissing, want: "a secret or a config map that a container needs is not available"},
		{name: "volume mount failed", reason: StopReasonVolumeMountFailed, want: "Kubernetes cannot mount a volume of the pod"},
		{name: "admission denied", reason: StopReasonAdmissionDenied, want: "the cluster did not accept the pod"},
		{name: "initialization timeout", reason: StopReasonInitTimeout, want: "the first step did not start before the initialization timeout of the workflow"},
		{name: "process killed", reason: StopReasonProcessKilled, want: "the test process was killed, possibly by an out-of-memory kill"},
		{name: "step timeout", reason: StopReasonStepTimeout, want: "the step did not finish within its timeout"},
		{name: "user cancel", reason: StopReasonUserCancel, want: "a person canceled the execution"},
		{name: "force cancel", reason: StopReasonForceCancel, want: "a person canceled the execution by force"},
		{name: "template missing", reason: StopReasonTemplateMissing, want: "a template that the workflow uses does not exist"},
		{name: "queue limit exceeded", reason: StopReasonQueueLimitExceeded, want: "the environment reached its queue limit"},
		{name: "git auth failed", reason: StopReasonGitAuthFailed, want: "the credential for the repository was refused"},
		{name: "git clone failed", reason: StopReasonGitCloneFailed, want: "the repository could not be cloned"},
		{name: "service not ready", reason: StopReasonServiceNotReady, want: "a service of the step did not become ready"},
		{name: "artifact upload failed", reason: StopReasonArtifactUploadFailed, want: "the artifacts could not be uploaded"},
		{name: "child workflow failed", reason: StopReasonChildWorkflowFailed, want: "a workflow that this step ran did not pass"},
		{name: "out of memory kill", reason: StopReasonOOMKilled, want: "the container exceeded its memory limit"},
		{name: "evicted", reason: StopReasonEvicted, want: "Kubernetes evicted the pod"},
		{name: "preempted", reason: StopReasonPreempted, want: "the scheduler preempted the pod to run a pod with a higher priority"},
		{name: "node shutdown", reason: StopReasonNodeShutdown, want: "the node that ran the pod shut down"},
		{name: "container error", reason: StopReasonContainerError, want: "a container of the pod could not run"},
		{name: "deadline exceeded", reason: StopReasonDeadlineExceeded, want: "the pod exceeded the deadline of the job"},
		{name: "job deleted", reason: StopReasonJobDeleted, want: "the job of the execution was deleted"},
		{name: "fail fast", reason: StopReasonFailFast, want: "another parallel worker failed"},
		{name: "trigger abort", reason: StopReasonTriggerAbort, want: "the trigger of the execution was deleted"},
		{name: "exit code", reason: StopReasonExitCode, want: "a step of the test failed"},
		{name: "unknown", reason: StopReasonUnknown, want: "the cause is not known"},
		{name: "token from a newer control plane has no words yet", reason: StopReason("later-added"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.reason.Sentence())
		})
	}
}
