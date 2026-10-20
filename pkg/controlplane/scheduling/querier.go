package scheduling

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// RunnerInfo identifies the runner a dispatch is addressed to.
//
// The standalone control plane knows its single runner in advance, so this is
// filled from constants rather than resolved from a policy.
type RunnerInfo struct {
	Id            string
	Name          string
	EnvironmentId string
}

// ExecutionTransition is the whole of what the control-instruction path needs:
// which execution, and what the runner should do to it.
//
// Hydrating a full execution to carry an id and a target state is what turned
// one poll into 1+6N queries. The previous shape passed a 40-field struct with a
// nil Result, which the handler then dereferenced.
type ExecutionTransition struct {
	Id              string
	Status          testkube.TestWorkflowStatus
	PredictedStatus testkube.TestWorkflowStatus
}

// ExecutionQuerier reads the work the runner has to be told about.
//
// These return slices rather than iterators on purpose. The postgres
// implementation materialised its whole result set before yielding the first
// element, so the iterator bought the caller nothing, and its
// yield(zeroValue, err) convention produced two bugs: one bad row aborted the
// entire iteration, and the error branch was duplicated with the second copy
// unreachable.
type ExecutionQuerier interface {
	// Transitions returns every execution awaiting a pause, resume or stop.
	Transitions(ctx context.Context) ([]ExecutionTransition, error)

	// ToStart returns at most limit executions to dispatch, oldest scheduled
	// first: everything ASSIGNED, plus STARTING rows last handed out before
	// redispatchBefore.
	//
	// The returned executions carry only the fields the runner is sent. Workflow
	// holds the name and nothing else - the runner fetches the real workflow with
	// GetExecutionWorkflow, and decoding the spec here is what made the poll
	// exceed the runner's call deadline.
	ToStart(ctx context.Context, limit int, redispatchBefore time.Time) ([]testkube.TestWorkflowExecution, error)

	// StaleStarting returns the ids of executions handed to a runner that never
	// reported back, so they can be failed explicitly instead of sitting in
	// STARTING forever.
	StaleStarting(ctx context.Context, staleBefore time.Time, limit int) ([]string, error)
}
