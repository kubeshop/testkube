package watchers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	constants2 "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type resultState struct {
	result testkube.TestWorkflowResult
	state  ExecutionState
	mu     sync.RWMutex
}

type ResultState interface {
	Result() testkube.TestWorkflowResult
	Append(ts time.Time, hint instructions.Instruction)
	Align(state ExecutionState)
	End()
}

// TODO: Optimize memory
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

	// Build the state object
	state := &resultState{result: initial}
	state.applyPredictedStatus()
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

func (r *resultState) reconcileDuration() {
	r.result.RecomputeDuration()
}

func (r *resultState) areAllStepsFinished() bool {
	for _, step := range r.result.Steps {
		if !step.Finished() {
			return false
		}
	}
	return true
}

func (r *resultState) isAnyStepAborted() bool {
	// When initialization was aborted or failed - it's immediately end
	if r.result.Initialization.Status.AnyError() {
		return true
	}

	// Analyze the rest of the steps
	for _, step := range r.result.Steps {
		// When any step was aborted - it's immediately end
		if step.Status != nil && *step.Status == testkube.ABORTED_TestWorkflowStepStatus {
			return true
		}
	}
	return false
}

func (r *resultState) isAnyStepPaused() bool {
	if *r.result.Initialization.Status == testkube.PAUSED_TestWorkflowStepStatus {
		return true
	}
	for _, step := range r.result.Steps {
		if step.Status != nil && *step.Status == testkube.PAUSED_TestWorkflowStepStatus {
			return true
		}
	}
	return false
}

func (r *resultState) applyStatus() {
	if !r.result.FinishedAt.IsZero() {
		r.result.Status = r.result.PredictedStatus
	} else if !r.result.StartedAt.IsZero() {
		if r.isAnyStepPaused() {
			r.result.Status = common.Ptr(testkube.PAUSED_TestWorkflowStatus)
		} else {
			r.result.Status = common.Ptr(testkube.RUNNING_TestWorkflowStatus)
		}
	}
}

func (r *resultState) applyPredictedStatus() {
	// Mark as aborted, when any step is aborted
	if r.isAnyStepAborted() || r.result.Initialization.Status.AnyError() {
		r.result.PredictedStatus = common.Ptr(testkube.ABORTED_TestWorkflowStatus)
		return
	}

	// Determine if there are some steps failed
	for ref := range r.result.Steps {
		if r.result.Steps[ref].Status != nil && r.result.Steps[ref].Status.AnyError() {
			r.result.PredictedStatus = common.Ptr(testkube.FAILED_TestWorkflowStatus)
			return
		}
	}
	r.result.PredictedStatus = common.Ptr(testkube.PASSED_TestWorkflowStatus)
}

func (r *resultState) markAborted() {
	// Load the error message
	defaultMessage := "Job was aborted" // TODO: Move as constant
	bareErrorMessage := defaultMessage
	if r.state != nil && r.state.ExecutionError() != "" {
		bareErrorMessage = r.state.ExecutionError()
	}
	errorMessage := fmt.Sprintf("The execution has been aborted. (%s)", bareErrorMessage)

	// Fetch the sequence
	sig, _ := r.state.Signature()
	sigSequence := stage.MapSignatureToSequence(sig)

	// Create marker to know if there is any step marked as aborted already
	aborted := false

	// Check the initialization step
	if !r.result.Initialization.Finished() || r.result.Initialization.Aborted() {
		aborted = true
		r.result.Initialization.Status = common.Ptr(testkube.ABORTED_TestWorkflowStepStatus)
		r.result.Initialization.ErrorMessage = errorMessage
	}

	// Check all the executable steps in the sequence
	for i := range sigSequence {
		ref := sigSequence[i].Ref()
		if ref == InitStepRef || !r.isKnownStep(ref) || len(sigSequence[i].Children()) > 0 {
			continue
		}
		step := r.result.Steps[ref]
		if step.Finished() && !step.Aborted() && (!step.Skipped() || step.ErrorMessage == "") {
			if *step.Status == testkube.ABORTED_TestWorkflowStepStatus && (step.ErrorMessage == "" || step.ErrorMessage == defaultMessage) {
				step.ErrorMessage = errorMessage
			}
			continue
		}
		if aborted {
			step.Status = common.Ptr(testkube.SKIPPED_TestWorkflowStepStatus)
			step.ErrorMessage = fmt.Sprintf("The execution was aborted before. %s", color.FgDarkGray.Render("("+bareErrorMessage+")"))
		} else {
			aborted = true
			step.Status = common.Ptr(testkube.ABORTED_TestWorkflowStepStatus)
			step.ErrorMessage = errorMessage
		}
		r.result.Steps[ref] = step
	}

	// Adjust all the group steps in the sequence.
	// Do it from end, so we can handle nested groups
	for i := len(sigSequence) - 1; i >= 0; i-- {
		ref := sigSequence[i].Ref()
		if ref == InitStepRef || !r.isKnownStep(ref) || len(sigSequence[i].Children()) == 0 {
			continue
		}
		step := r.result.Steps[ref]
		if step.Finished() {
			continue
		}
		allSkipped := true
		anyAborted := false
		for _, childSig := range sigSequence[i].Children() {
			// TODO: What about nested virtual groups? We don't have their statuses
			if !r.isKnownStep(childSig.Ref()) {
				continue
			}
			if r.result.Steps[childSig.Ref()].Status.Aborted() {
				anyAborted = true
			}
			if !r.result.Steps[childSig.Ref()].Status.Skipped() {
				allSkipped = false
			}
		}
		if allSkipped {
			step.Status = common.Ptr(testkube.SKIPPED_TestWorkflowStepStatus)
		} else if anyAborted {
			step.Status = common.Ptr(testkube.ABORTED_TestWorkflowStepStatus)
		}
	}

	// The rest of steps is unrecognized, so just mark them as aborted with information about faulty state
	for ref, step := range r.result.Steps {
		if step.Finished() {
			continue
		}
		step.Status = common.Ptr(testkube.ABORTED_TestWorkflowStepStatus)
		step.ErrorMessage = fmt.Sprintf("The execution was aborted, but we could not determine steps order: %s", bareErrorMessage)
		r.result.Steps[ref] = step
	}
}

func (r *resultState) lastTimestamp() time.Time {
	// omit r.result.FinishedAt to avoid this approximation
	ts := latestTimestamp(r.result.QueuedAt, r.result.StartedAt, r.result.Initialization.QueuedAt, r.result.Initialization.StartedAt, r.result.Initialization.FinishedAt)
	for i := range r.result.Steps {
		ts = latestTimestamp(ts, r.result.Steps[i].QueuedAt, r.result.Steps[i].StartedAt, r.result.Steps[i].FinishedAt)
	}
	return ts
}

func (r *resultState) adjustTimestamps() {
	// Detect the initialization queue time
	r.result.Initialization.QueuedAt = earliestTimestamp(r.result.StartedAt, r.result.Initialization.QueuedAt)

	// Ensure there is the start time for the initialization if it's started or done
	if !r.result.Initialization.NotStarted() {
		var containerStartTs time.Time
		if r.state != nil {
			containerStartTs = r.state.ContainerStartTimestamp("1")
		}
		r.result.Initialization.StartedAt = latestTimestamp(r.result.Initialization.StartedAt, containerStartTs, r.result.Initialization.QueuedAt)
	}

	// Build the completion timestamp
	var completionTs time.Time
	if r.state != nil {
		completionTs = r.state.CompletionTimestamp()
	}

	// Ensure there is the end time for the initialization if it's done
	if r.result.Initialization.Finished() && r.result.Initialization.FinishedAt.IsZero() {
		// Fallback to have any timestamp in case something went wrong
		if r.result.Initialization.Aborted() {
			r.result.Initialization.FinishedAt = latestTimestamp(r.result.Initialization.StartedAt, completionTs)
		} else {
			r.result.Initialization.FinishedAt = r.result.Initialization.StartedAt
		}
	}

	// Fetch the sequence
	var sig []stage.Signature
	if r.state != nil {
		sig, _ = r.state.Signature()
	}
	sigSequence := stage.MapSignatureToSequence(sig)
	sigSequenceExecutionOnly := common.FilterSlice(sigSequence, func(sig stage.Signature) bool {
		return len(sig.Children()) == 0
	})
	sigSequenceExecutionGroupOnly := common.FilterSlice(sigSequence, func(sig stage.Signature) bool {
		return len(sig.Children()) > 0
	})

	if len(sigSequence) == 0 {
		// TODO: Try to estimate signature sequence
	}

	// Set up everywhere queued at time to past finished time
	lastTs := r.result.Initialization.FinishedAt
	for _, s := range sigSequenceExecutionOnly {
		step := r.result.Steps[s.Ref()]
		if !step.QueuedAt.Equal(lastTs) {
			step.QueuedAt = lastTs
		}
		if step.FinishedAt.IsZero() && step.Status != nil && *step.Status == testkube.ABORTED_TestWorkflowStepStatus {
			step.FinishedAt = completionTs
		}
		if !step.QueuedAt.IsZero() && !step.FinishedAt.IsZero() && step.StartedAt.IsZero() {
			step.StartedAt = step.QueuedAt
		}

		r.result.Steps[s.Ref()] = step
		lastTs = step.FinishedAt
	}

	// Set up for groups too.
	// Do it from end, so we handle nested groups
	for i := len(sigSequenceExecutionGroupOnly) - 1; i >= 0; i-- {
		s := sigSequenceExecutionGroupOnly[i]
		seq := s.Sequence()
		expectedQueuedAt := r.result.Steps[seq[1].Ref()].QueuedAt
		expectedFinishedAt := r.result.Steps[seq[len(seq)-1].Ref()].FinishedAt
		step := r.result.Steps[s.Ref()]
		if !r.result.Steps[s.Ref()].QueuedAt.Equal(expectedQueuedAt) {
			step.QueuedAt = expectedQueuedAt
		}
		if !r.result.Steps[s.Ref()].FinishedAt.Equal(expectedFinishedAt) {
			step.FinishedAt = expectedFinishedAt
		}
		r.result.Steps[s.Ref()] = step
	}

	if r.areAllStepsFinished() {
		r.result.FinishedAt = firstNonZero(r.lastTimestamp(), completionTs)
	}
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

	// Determine the execution instructions
	actions, _ := r.state.ActionGroups()
	_, endRefs := ExtractRefsFromActionGroup(actions)
	signature, _ := r.state.Signature()
	signatureSeq := stage.MapSignatureToSequence(signature)

	// Find the furthest step with any status
	processedStepsCount := 0
	if force {
		processedStepsCount = len(signatureSeq)
	} else {
		for i := range signatureSeq {
			step := r.result.Steps[signatureSeq[i].Ref()]
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
	for containerIndex := range endRefs {
		for refIndex := range endRefs[containerIndex] {
			containerIndexes[endRefs[containerIndex][refIndex]] = containerIndex
			refIndexes[endRefs[containerIndex][refIndex]] = refIndex
		}
	}

	// Gather the container results
	containerResults := make([]ContainerResult, len(actions))
	for i := range actions {
		containerResults[i] = r.state.Pod().ContainerResult(fmt.Sprintf("%d", i+1), r.state.ExecutionError())
	}

	// Apply statuses from the container results since the last step
	for i := 0; i < processedStepsCount; i++ {
		ref := signatureSeq[i].Ref()
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
	r.fillGaps(false)
	r.adjustTimestamps()
	r.fillGaps(false) // do it 2nd time, in case timestamps have added result.finishedAt date
	r.reconcileDuration()
	r.applyPredictedStatus()
	r.applyStatus()
}

// Append applies the precise hint information about the action that took place
func (r *resultState) Append(ts time.Time, hint instructions.Instruction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.reconcile()

	// Ensure we have UTC timestamp
	ts = ts.UTC()

	// Save the information about the final status if it's possible
	if hint.Ref == constants2.RootOperationName && hint.Name == constants.InstructionEnd {
		status := testkube.TestWorkflowStatus(hint.Value.(string))
		if status == "" {
			status = testkube.PASSED_TestWorkflowStatus
		}
		r.result.Status = common.Ptr(status)
		r.result.FinishedAt = ts
	}

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

func (r *resultState) Align(state ExecutionState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.reconcile()

	r.state = state

	// Initialization phase
	if !state.EstimatedJobCreationTimestamp().IsZero() {
		r.result.QueuedAt = state.EstimatedJobCreationTimestamp().UTC()
	}
	if !state.EstimatedPodCreationTimestamp().IsZero() {
		r.result.StartedAt = state.EstimatedPodCreationTimestamp().UTC()
	}

	// Determine the execution instructions
	signature, _ := state.Signature()
	signatureSeq := stage.MapSignatureToSequence(signature)

	// Create missing step results that are recognized with the signature
	for i := range signatureSeq {
		if _, ok := r.result.Steps[signatureSeq[i].Ref()]; !ok {
			r.result.Steps[signatureSeq[i].Ref()] = testkube.TestWorkflowStepResult{
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

	// Ensure that the steps without the information are marked as aborted
	r.fillGaps(true)
	r.markAborted()

	// Finalize the status
	r.reconcile()
}

func firstNonZero(timestamps ...time.Time) time.Time {
	for _, t := range timestamps {
		if !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func earliestTimestamp(ts ...time.Time) (earliest time.Time) {
	for _, t := range ts {
		if !t.IsZero() && (earliest.IsZero() || t.Before(earliest)) {
			earliest = t
		}
	}
	return
}

func latestTimestamp(ts ...time.Time) (latest time.Time) {
	for _, t := range ts {
		if !t.IsZero() && (latest.IsZero() || t.After(latest)) {
			latest = t
		}
	}
	return
}
