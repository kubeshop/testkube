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

	// A person stopped the execution.
	StopReasonUserCancel  StopReason = "user-cancel"
	StopReasonForceCancel StopReason = "force-cancel"

	// The control plane refused the execution at scheduling time.
	StopReasonTemplateMissing    StopReason = "template-missing"
	StopReasonQueueLimitExceeded StopReason = "queue-limit-exceeded"

	// A toolkit step reported the cause.
	StopReasonGitAuthFailed        StopReason = "git-auth-failed"
	StopReasonGitCloneFailed       StopReason = "git-clone-failed"
	StopReasonServiceNotReady      StopReason = "service-not-ready"
	StopReasonArtifactUploadFailed StopReason = "artifact-upload-failed"
	StopReasonChildWorkflowFailed  StopReason = "child-workflow-failed"

	// Kubernetes stopped the pod.
	StopReasonOOMKilled        StopReason = "oom-killed"
	StopReasonEvicted          StopReason = "evicted"
	StopReasonPreempted        StopReason = "preempted"
	StopReasonNodeShutdown     StopReason = "node-shutdown"
	StopReasonContainerError   StopReason = "container-error"
	StopReasonDeadlineExceeded StopReason = "deadline-exceeded"
	StopReasonJobDeleted       StopReason = "job-deleted"

	// Another component of the execution decided the stop.
	StopReasonFailFast     StopReason = "fail-fast"
	StopReasonTriggerAbort StopReason = "trigger-abort"

	// The test itself failed, and no other signal explains the result.
	StopReasonExitCode StopReason = "exit-code"
	StopReasonUnknown  StopReason = "unknown"
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
	case StopReasonUserCancel:
		return "a person canceled the execution"
	case StopReasonForceCancel:
		return "a person canceled the execution by force"
	case StopReasonTemplateMissing:
		return "a template that the workflow uses does not exist"
	case StopReasonQueueLimitExceeded:
		return "the environment reached its queue limit"
	case StopReasonGitAuthFailed:
		return "the credential for the repository was refused"
	case StopReasonGitCloneFailed:
		return "the repository could not be cloned"
	case StopReasonServiceNotReady:
		return "a service of the step did not become ready"
	case StopReasonArtifactUploadFailed:
		return "the artifacts could not be uploaded"
	case StopReasonChildWorkflowFailed:
		return "a workflow that this step ran did not pass"
	case StopReasonOOMKilled:
		return "the container exceeded its memory limit"
	case StopReasonEvicted:
		return "Kubernetes evicted the pod"
	case StopReasonPreempted:
		return "the scheduler preempted the pod to run a pod with a higher priority"
	case StopReasonNodeShutdown:
		return "the node that ran the pod shut down"
	case StopReasonContainerError:
		return "a container of the pod could not run"
	case StopReasonDeadlineExceeded:
		return "the pod exceeded the deadline of the job"
	case StopReasonJobDeleted:
		return "the job of the execution was deleted"
	case StopReasonFailFast:
		return "another parallel worker failed"
	case StopReasonTriggerAbort:
		return "the trigger of the execution was deleted"
	case StopReasonExitCode:
		return "a step of the test failed"
	case StopReasonUnknown:
		return "the cause is not known"
	default:
		return ""
	}
}
