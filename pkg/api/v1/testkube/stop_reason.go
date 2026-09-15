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
	default:
		return ""
	}
}
