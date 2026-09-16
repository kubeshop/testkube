package testkube

// StopReason is the code for the cause of an abort or a cancel. The control plane
// stores it, sends it with the stop transition, and the runner writes it into the
// job annotations. The values are stable, because filters and telemetry use them.
type StopReason string

const (
	StopReasonAbortAll           StopReason = "abort-all"
	StopReasonSuperseded         StopReason = "superseded"
	StopReasonQueueTimeout       StopReason = "queue-timeout"
	StopReasonQueuedTooLong      StopReason = "queued-too-long"
	StopReasonTransitionTimeout  StopReason = "transition-timeout"
	StopReasonStopNotConfirmed   StopReason = "stop-not-confirmed"
	StopReasonExecutionTimeout   StopReason = "execution-timeout"
	StopReasonExecutionStuck     StopReason = "execution-stuck"
	StopReasonWorkerResumeFailed StopReason = "worker-resume-failed"
	StopReasonUnschedulable      StopReason = "unschedulable"
	StopReasonConfigMissing      StopReason = "config-missing"
	StopReasonVolumeMountFailed  StopReason = "volume-mount-failed"
	StopReasonAdmissionDenied    StopReason = "admission-denied"
	StopReasonInitTimeout        StopReason = "initialization-timeout"
	StopReasonProcessKilled      StopReason = "process-killed"
	StopReasonStepTimeout        StopReason = "step-timeout"
)

// Sentence returns the words for the reason in a message for people. It returns an
// empty string for a value it does not know, so a reader can keep the raw code.
func (r StopReason) Sentence() string {
	switch r {
	case StopReasonAbortAll:
		return "all executions of the workflow were stopped"
	case StopReasonSuperseded:
		return "a newer commit superseded this run"
	case StopReasonQueueTimeout:
		return "the execution exceeded the queue timeout of the workflow"
	case StopReasonQueuedTooLong:
		return "the execution stayed queued for too long"
	case StopReasonTransitionTimeout:
		return "the execution stayed in a transitional state for too long"
	case StopReasonStopNotConfirmed:
		return "the runner did not confirm the stop in time"
	case StopReasonExecutionTimeout:
		return "the execution ran for too long"
	case StopReasonExecutionStuck:
		return "the execution is stuck in the running state"
	case StopReasonWorkerResumeFailed:
		return "the parallel worker could not be resumed"
	case StopReasonUnschedulable:
		return "no node can run the pod"
	case StopReasonConfigMissing:
		return "a secret or a config map that a container needs is not available"
	case StopReasonVolumeMountFailed:
		return "Kubernetes cannot mount a volume of the pod"
	case StopReasonAdmissionDenied:
		return "the cluster did not accept the pod"
	case StopReasonInitTimeout:
		return "the first step did not start before the initialization timeout of the workflow"
	case StopReasonProcessKilled:
		return "the test process was killed, possibly by an out-of-memory kill"
	case StopReasonStepTimeout:
		return "the step did not finish within its timeout"
	default:
		return ""
	}
}
