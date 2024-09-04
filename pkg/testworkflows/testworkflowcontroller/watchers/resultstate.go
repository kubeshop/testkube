package watchers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	DefaultErrorMessage = "Job has been aborted"
)

type resultState struct {
	result testkube.TestWorkflowResult
	state  ExecutionState
	mu     sync.RWMutex

	// Temporary data to avoid finishing before
	ended bool

	// Cached data
	actions     actiontypes.ActionGroups
	endRefs     [][]string
	sigSequence []testkube.TestWorkflowSignature
}

type ResultState interface {
	Result() testkube.TestWorkflowResult
	Append(ts time.Time, hint instructions.Instruction)
	Align(state ExecutionState)
	End()
}

// TODO: Optimize memory
// TODO: Provide initial actions/signature
func NewResultState(initial testkube.TestWorkflowResult) ResultState {
	// Apply data that are required yet may be missing
	if initial.Initialization == nil {
		initial.Initialization = &testkube.TestWorkflowStepResult{
			Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
		}
	}
	if initial.Steps == nil {
		initial.Steps = make(map[string]testkube.TestWorkflowStepResult)
	}
	if initial.Status == nil {
		initial.Status = common.Ptr(testkube.QUEUED_TestWorkflowStatus)
	}

	// Mark initial as non-finished, as the state is not yet marked as ended
	if initial.Status.Finished() {
		initial.Status = common.Ptr(testkube.RUNNING_TestWorkflowStatus)
	}
	if !initial.FinishedAt.IsZero() {
		initial.FinishedAt = time.Time{}
	}

	// Build the state object
	state := &resultState{result: initial}
	return state
}

func (r *resultState) Result() testkube.TestWorkflowResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return *r.result.Clone()
}

func (r *resultState) isKnownStep(ref string) bool {
	if ref == InitStepRef {
		return true
	}
	_, ok := r.result.Steps[ref]
	return ok
}

func (r *resultState) useInitialStateData() {
	// TODO: Apply Running state, queued at, started at for initialization/workflow
}

func (r *resultState) reconcileStateData() {
	// TODO: Apply predicted status
}

func (r *resultState) fillGaps(force bool) {
	if r.state == nil {
		return
	}

	// Mark the initialization step as running
	if r.state.PodCreated() && r.result.Initialization.NotStarted() {
		r.result.Initialization.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
	}

	// Don't compute anything more if there is no Pod information
	if r.state.Pod() == nil {
		return
	}

	// Find the furthest step with any status
	processedStepsCount := 0
	if force {
		processedStepsCount = len(r.sigSequence)
	} else {
		for i := range r.sigSequence {
			step := r.result.Steps[r.sigSequence[i].Ref]
			if !step.NotStarted() {
				processedStepsCount = i
				if step.Finished() {
					processedStepsCount++
				}
			}
		}
	}

	// Analyze the references
	containerIndexes := make(map[string]int)
	refIndexes := make(map[string]int)
	for containerIndex := range r.endRefs {
		for refIndex := range r.endRefs[containerIndex] {
			containerIndexes[r.endRefs[containerIndex][refIndex]] = containerIndex
			refIndexes[r.endRefs[containerIndex][refIndex]] = refIndex
		}
	}

	// Gather the container results
	containerResults := make([]ContainerResult, len(r.actions))
	for i := range r.actions {
		containerResults[i] = r.state.Pod().ContainerResult(fmt.Sprintf("%d", i+1), r.state.ExecutionError())
	}

	// Apply statuses from the container results since the last step
	for i := 0; i < processedStepsCount; i++ {
		ref := r.sigSequence[i].Ref
		container := containerResults[containerIndexes[ref]]
		if len(container.Statuses) <= refIndexes[ref] {
			continue
		}

		// TODO: estimate startedAt/finishedAt too?

		if ref == InitStepRef {
			r.result.Initialization.Status = common.Ptr(container.Statuses[refIndexes[ref]].Status)
			r.result.Initialization.ExitCode = float64(container.Statuses[refIndexes[ref]].ExitCode)
		} else {
			step := r.result.Steps[ref]
			step.Status = common.Ptr(container.Statuses[refIndexes[ref]].Status)
			step.ExitCode = float64(container.Statuses[refIndexes[ref]].ExitCode)
			r.result.Steps[ref] = step
		}
	}
}

func (r *resultState) reconcile() {
	// Build the completion timestamp
	var completionTs time.Time
	if r.state != nil {
		completionTs = r.state.CompletionTimestamp()
	}

	// Build the timestamp for initial container
	var containerStartTs time.Time
	if r.state != nil {
		containerStartTs = r.state.ContainerStartTimestamp("1")
	}

	//// TODO: Try to estimate signature sequence (?)
	//if len(r.sigSequence) == 0 {
	//}

	r.fillGaps(false)
	r.result.HealDuration()
	r.result.HealTimestamps(r.sigSequence, containerStartTs, completionTs, r.ended)
	r.result.HealDuration()
	r.result.HealMissingPauseStatuses()
	r.result.HealStatus()
}

// Append applies the precise hint information about the action that took place
func (r *resultState) Append(ts time.Time, hint instructions.Instruction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.reconcile()

	// Ensure we have UTC timestamp
	ts = ts.UTC()

	// Load the current step information
	init := hint.Ref == InitStepRef
	step, ok := r.result.Steps[hint.Ref]
	if init {
		step = *r.result.Initialization
		ok = true
	}

	// Ignore the virtual steps
	if !ok {
		return
	}

	// Apply the hint
	switch hint.Name {
	case constants.InstructionStart:
		step.StartedAt = ts
		step.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
	case constants.InstructionEnd:
		status := testkube.TestWorkflowStepStatus(hint.Value.(string))
		if status == "" {
			status = testkube.PASSED_TestWorkflowStepStatus
		}
		step.Status = common.Ptr(status)
		step.FinishedAt = ts
	case constants.InstructionExecution:
		serialized, _ := json.Marshal(hint.Value)
		var executionResult constants.ExecutionResult
		_ = json.Unmarshal(serialized, &executionResult)
		step.ExitCode = float64(executionResult.ExitCode)
		if executionResult.Details != "" {
			step.ErrorMessage = executionResult.Details
		}
	case constants.InstructionPause:
		pauseTsStr := hint.Value.(string)
		pauseTs, err := time.Parse(time.RFC3339Nano, pauseTsStr)
		if err != nil {
			pauseTs = ts
		}
		if !r.result.HasPauseAt(hint.Ref, pauseTs) {
			step.Status = common.Ptr(testkube.PAUSED_TestWorkflowStepStatus)
			r.result.Pauses = append(r.result.Pauses, testkube.TestWorkflowPause{Ref: hint.Ref, PausedAt: pauseTs})
		}
	case constants.InstructionResume:
		resumeTsStr := hint.Value.(string)
		resumeTs, err := time.Parse(time.RFC3339Nano, resumeTsStr)
		if err != nil {
			resumeTs = ts
		}
		if r.result.HasPauseAt(hint.Ref, resumeTs) {
			step.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
			for pi, p := range r.result.Pauses {
				if p.Ref != hint.Ref {
					continue
				}
				// Check if it's not already covered by that period
				if !p.PausedAt.After(resumeTs) && !p.ResumedAt.Before(resumeTs) {
					break
				}
				// Check if the period could not be fulfilled with that timestamp
				if !p.PausedAt.After(resumeTs) && (p.ResumedAt.IsZero() || p.ResumedAt.Equal(resumeTs)) {
					r.result.Pauses[pi].ResumedAt = resumeTs
					break
				}
			}
		}
	}

	// Save the step
	if init {
		r.result.Initialization = common.Ptr(step)
	} else {
		r.result.Steps[hint.Ref] = step
	}
}

func (r *resultState) useSignature(sig []stage.Signature) {
	r.sigSequence = stage.MapSignatureListToInternal(stage.MapSignatureToSequence(sig))
}

func (r *resultState) useActionGroups(actions actiontypes.ActionGroups) {
	r.actions = actions
	_, r.endRefs = ExtractRefsFromActionGroup(actions)
}

func (r *resultState) Align(state ExecutionState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.reconcile()

	r.state = state

	// Cache the data used for reconciliation
	if len(r.sigSequence) == 0 {
		sig, _ := r.state.Signature()
		r.useSignature(sig)
	}
	if len(r.actions) == 0 {
		actions, _ := r.state.ActionGroups()
		r.useActionGroups(actions)
	}

	// Initialization phase
	if !state.EstimatedJobCreationTimestamp().IsZero() {
		r.result.QueuedAt = state.EstimatedJobCreationTimestamp().UTC()
	}
	if !state.EstimatedPodCreationTimestamp().IsZero() {
		r.result.StartedAt = state.EstimatedPodCreationTimestamp().UTC()
	}

	// Create missing step results that are recognized with the signature
	for i := range r.sigSequence {
		if _, ok := r.result.Steps[r.sigSequence[i].Ref]; !ok {
			r.result.Steps[r.sigSequence[i].Ref] = testkube.TestWorkflowStepResult{
				Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
			}
		}
	}
}

// End tries to finalize the result based on the available data.
// It will try to fill all the gaps.
func (r *resultState) End() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Mark as finished
	r.ended = true

	// Ensure that the steps without the information are fulfilled and marked as aborted
	r.fillGaps(true)

	errorMessage := DefaultErrorMessage
	if r.state != nil && r.state.ExecutionError() != "" {
		errorMessage = r.state.ExecutionError()
	}
	r.result.HealAborted(r.sigSequence, errorMessage, DefaultErrorMessage)

	// Finalize the status
	r.reconcile()
}
