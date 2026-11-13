package testkube

// ExecutionRetryInput is what ShouldRetryExecution reads. The caller passes the stored
// execution as it is: the function never reads a running one.
type ExecutionRetryInput struct {
	// Count and On come from spec.execution.retry. A zero Count means no policy.
	Count int32
	On    []string
	// Status and StatusDetails come from the finished result.
	Status        TestWorkflowStatus
	StatusDetails *TestWorkflowStatusDetails
	// Attempt is the execution's lineage attempt, 1 for an original run.
	Attempt int32
}

// Retry causes, as spec.execution.retry.on spells them.
const (
	ExecutionRetryOnInfrastructure = "infrastructure"
	ExecutionRetryOnTimeout        = "timeout"
	ExecutionRetryOnOOM            = "oom"
)

// executionRetryReasons maps a retry cause to the stop reasons it covers. Only the reasons
// listed here are ever retried: step failures, init failures and cancels are not in it.
// unschedulable is here although its type is init-failure, because it is the cluster lacking
// room, which a retry can fix; every other init-failure is deterministic.
var executionRetryReasons = map[string][]StopReason{
	ExecutionRetryOnInfrastructure: {StopReasonEvicted, StopReasonPreempted, StopReasonNodeShutdown, StopReasonUnschedulable},
	ExecutionRetryOnTimeout:        {StopReasonExecutionTimeout},
	ExecutionRetryOnOOM:            {StopReasonOOMKilled},
}

// ShouldRetryExecution decides whether a finished execution is retried as a new attempt of
// the same chain. It is true only when the policy exists, the attempt is within count (the
// first run is attempt 1, so count 2 allows attempts 1 and 2 to retry), the execution
// aborted or failed, a person did not stop it, and its reason is one that the policy's
// causes cover. An empty On means infrastructure. A missing StatusDetails never retries,
// because a stop with no reason cannot be told from a test failure. It is the only place
// that decides, in this repository and in testkube-cloud-api.
func ShouldRetryExecution(in ExecutionRetryInput) bool {
	if in.Count < 1 || in.Attempt > in.Count || in.StatusDetails == nil {
		return false
	}
	if in.Status != ABORTED_TestWorkflowStatus && in.Status != FAILED_TestWorkflowStatus {
		return false
	}
	if StatusDetailsType(in.StatusDetails.Type_) == StatusDetailsTypeUserCancel {
		return false
	}
	on := in.On
	if len(on) == 0 {
		on = []string{ExecutionRetryOnInfrastructure}
	}
	for _, cause := range on {
		for _, reason := range executionRetryReasons[cause] {
			if in.StatusDetails.Reason == string(reason) {
				return true
			}
		}
	}
	return false
}
