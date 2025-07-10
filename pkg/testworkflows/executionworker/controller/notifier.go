package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	color2 "github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	log2 "github.com/kubeshop/testkube/pkg/log"
	watchers2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	DefaultErrorMessage = "Job has been aborted"
)

// Not thread-safe, should be used synchronously
type notifier struct {
	// Data
	result      testkube.TestWorkflowResult
	state       watchers2.ExecutionState
	scheduledAt time.Time

	// Temporary data to avoid finishing too early
	lastTs time.Time
	ended  bool

	// Cached data for better performance
	actions     actiontypes.ActionGroups
	endRefs     [][]string
	sigSequence []testkube.TestWorkflowSignature

	// Sending state
	ctx context.Context
	ch  chan ChannelMessage[Notification]
}

func (n *notifier) send(value Notification) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()
	select {
	case <-n.ctx.Done():
	case n.ch <- ChannelMessage[Notification]{Value: value}:
	}
}

func (n *notifier) error(err error) {
	// Ignore when the channel is already closed
	defer func() {
		recover()
	}()
	select {
	case <-n.ctx.Done():
	case n.ch <- ChannelMessage[Notification]{Error: err}:
	}
}

// TODO: Find a way to avoid sending if it is identical
func (n *notifier) sendResult() {
	result := n.result.Clone()
	n.send(Notification{Timestamp: result.LatestTimestamp(), Result: result})
}

func (n *notifier) Raw(ref string, ts time.Time, message string, temporary bool) {
	if ts.After(n.lastTs) {
		n.lastTs = ts
	}
	if message != "" {
		if ref == constants.InitStepName {
			ref = ""
		}
		n.send(Notification{
			Timestamp: ts.UTC(),
			Log:       message,
			Ref:       ref,
			Temporary: temporary,
		})
	}
}

func (n *notifier) Log(ref string, ts time.Time, message string) {
	if message != "" {
		n.Raw(ref, ts, fmt.Sprintf("%s %s", ts.Format(constants.PreciseTimeFormat), message), false)
	}
}

func (n *notifier) Error(err error) {
	n.error(err)
}

func (n *notifier) Event(ref string, ts time.Time, level, reason, message, execution string) {
	log2.DefaultLogger.Debugw("notify event while watching pod", "execution", execution, "reason", reason, "level", level, "message", message, "timestamp", ts)
	color := color2.FgGray.Render
	if level != "Normal" {
		color = color2.FgYellow.Render
	}
	log := color(fmt.Sprintf("(%s) %s", reason, message))
	n.Raw(ref, ts, fmt.Sprintf("%s %s\n", ts.Format(constants.PreciseTimeFormat), log), level == "Normal")
}

func (n *notifier) Output(ref string, ts time.Time, output *instructions.Instruction) {
	if ref == constants.InitStepName {
		ref = ""
	} else if ref != "" {
		if _, ok := n.result.Steps[ref]; !ok {
			return
		}
	}
	n.send(Notification{Timestamp: ts.UTC(), Ref: ref, Output: output})
}

func (n *notifier) useSignature(sig []stage.Signature) {
	n.sigSequence = stage.MapSignatureListToInternal(stage.MapSignatureToSequence(sig))
}

func (n *notifier) useActionGroups(actions actiontypes.ActionGroups) {
	n.actions = actions
	_, n.endRefs = ExtractRefsFromActionGroup(actions)
}

func (n *notifier) Align(state watchers2.ExecutionState) {
	log2.DefaultLogger.Debugw("notify alignment while watching pod", "execution", getExecutionId(state))

	defer n.sendResult()
	defer n.reconcile()

	n.state = state

	// Cache the data used for reconciliation
	if len(n.sigSequence) == 0 {
		sig, _ := n.state.Signature()
		n.useSignature(sig)
	}
	if len(n.actions) == 0 {
		actions, _ := n.state.ActionGroups()
		n.useActionGroups(actions)
	}

	// Initialization phase
	if !state.EstimatedJobCreationTimestamp().IsZero() {
		n.result.QueuedAt = state.EstimatedJobCreationTimestamp().UTC()
	}
	if !state.EstimatedPodCreationTimestamp().IsZero() {
		n.result.StartedAt = state.EstimatedPodCreationTimestamp().UTC()
	}

	// Create missing step results that are recognized with the signature
	for i := range n.sigSequence {
		if _, ok := n.result.Steps[n.sigSequence[i].Ref]; !ok {
			n.result.Steps[n.sigSequence[i].Ref] = testkube.TestWorkflowStepResult{
				Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
			}
		}
	}
}

// Instruction applies the precise hint information about the action that took place
func (n *notifier) Instruction(ts time.Time, hint instructions.Instruction, executionId string) {
	log2.DefaultLogger.Debugw("notify instruction while watching pod", "execution", executionId, "hint", hint.Name)
	defer n.sendResult()
	defer n.reconcile()

	// Ensure we have UTC timestamp
	ts = ts.UTC()

	// Load the current step information
	init := hint.Ref == constants.InitStepName
	step, ok := n.result.Steps[hint.Ref]
	if init {
		step = *n.result.Initialization
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
			status = testkube.ABORTED_TestWorkflowStepStatus
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
		if !n.result.HasPauseAt(hint.Ref, pauseTs) {
			step.Status = common.Ptr(testkube.PAUSED_TestWorkflowStepStatus)
			n.result.Pauses = append(n.result.Pauses, testkube.TestWorkflowPause{Ref: hint.Ref, PausedAt: pauseTs})
		}
	case constants.InstructionResume:
		resumeTsStr := hint.Value.(string)
		resumeTs, err := time.Parse(time.RFC3339Nano, resumeTsStr)
		if err != nil {
			resumeTs = ts
		}
		if n.result.HasPauseAt(hint.Ref, resumeTs) {
			step.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
			for pi, p := range n.result.Pauses {
				if p.Ref != hint.Ref {
					continue
				}
				// Check if it's not already covered by that period
				if !p.PausedAt.After(resumeTs) && !p.ResumedAt.Before(resumeTs) {
					break
				}
				// Check if the period could not be fulfilled with that timestamp
				if !p.PausedAt.After(resumeTs) && (p.ResumedAt.IsZero() || p.ResumedAt.Equal(resumeTs)) {
					n.result.Pauses[pi].ResumedAt = resumeTs
					break
				}
			}
		}
	}

	// Save the step
	if init {
		n.result.Initialization = common.Ptr(step)
	} else {
		n.result.Steps[hint.Ref] = step
	}
}

// End tries to finalize the result based on the available data.
// It will try to fill all the gaps.
func (n *notifier) End() {
	defer n.sendResult()

	// Mark as finished
	n.ended = true

	// Ensure that the steps without the information are fulfilled and marked as aborted
	n.fillGaps(true)

	terminationCode := watchers2.GetTerminationCode(n.state.Job().Original())
	errorMessage := DefaultErrorMessage
	if n.state != nil && n.state.ExecutionError() != "" {
		errorMessage = n.state.ExecutionError()
	}
	n.result.HealAbortedOrCanceled(n.sigSequence, errorMessage, DefaultErrorMessage, terminationCode)

	// Finalize the status
	n.reconcile()
}

func (n *notifier) fillGaps(force bool) {
	if n.state == nil {
		return
	}

	// Mark the initialization step as running
	if n.state.PodCreated() && n.result.Initialization.NotStarted() {
		n.result.Initialization.Status = common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)
	}

	// Don't compute anything more if there is no Pod information
	if n.state.Pod() == nil {
		return
	}

	// Find the furthest step with any status
	processedStepsCount := 0
	if force {
		processedStepsCount = len(n.sigSequence)
	} else {
		for i := range n.sigSequence {
			step := n.result.Steps[n.sigSequence[i].Ref]
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
	for containerIndex := range n.endRefs {
		for refIndex := range n.endRefs[containerIndex] {
			containerIndexes[n.endRefs[containerIndex][refIndex]] = containerIndex
			refIndexes[n.endRefs[containerIndex][refIndex]] = refIndex
		}
	}

	// Gather the container results
	containerResults := make([]watchers2.ContainerResult, len(n.actions))
	for i := range n.actions {
		containerResults[i] = n.state.Pod().ContainerResult(fmt.Sprintf("%d", i+1), n.state.ExecutionError())
	}

	// Apply statuses from the container results since the last step
	for i := 0; i < processedStepsCount; i++ {
		ref := n.sigSequence[i].Ref
		container := containerResults[containerIndexes[ref]]
		if len(container.Statuses) <= refIndexes[ref] {
			continue
		}

		// TODO: estimate startedAt/finishedAt too?

		if ref == constants.InitStepName {
			n.result.Initialization.Status = common.Ptr(container.Statuses[refIndexes[ref]].Status)
			n.result.Initialization.ExitCode = float64(container.Statuses[refIndexes[ref]].ExitCode)
		} else {
			step := n.result.Steps[ref]
			step.Status = common.Ptr(container.Statuses[refIndexes[ref]].Status)
			step.ExitCode = float64(container.Statuses[refIndexes[ref]].ExitCode)
			n.result.Steps[ref] = step
		}
	}
}

func (n *notifier) reconcile() {
	// Build the completion timestamp
	var completionTs time.Time
	if n.state != nil {
		completionTs = n.state.CompletionTimestamp()
	}
	if !completionTs.IsZero() && n.lastTs.After(completionTs) {
		completionTs = n.lastTs
	}

	// Build the timestamp for initial container
	var containerStartTs time.Time
	if n.state != nil {
		containerStartTs = n.state.ContainerStartTimestamp("1")
	}

	//// TODO: Try to estimate signature sequence (?)
	//if len(n.sigSequence) == 0 {
	//}

	n.fillGaps(false)
	n.result.HealTimestamps(n.sigSequence, n.scheduledAt, containerStartTs, completionTs, n.ended)
	n.result.HealDuration(n.scheduledAt)
	n.result.HealMissingPauseStatuses()
	n.result.HealStatus(n.sigSequence)
}

// TODO: Optimize memory
// TODO: Provide initial actions/signature
func newNotifier(ctx context.Context, initialResult testkube.TestWorkflowResult, scheduledAt time.Time) *notifier {
	// Apply data that are required yet may be missing
	if initialResult.Initialization == nil {
		initialResult.Initialization = &testkube.TestWorkflowStepResult{
			Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
		}
	}
	if initialResult.Steps == nil {
		initialResult.Steps = make(map[string]testkube.TestWorkflowStepResult)
	}
	if initialResult.Status == nil {
		initialResult.Status = common.Ptr(testkube.QUEUED_TestWorkflowStatus)
	}

	// Mark initial as non-finished, as the state is not yet marked as ended
	if initialResult.Status.Finished() {
		initialResult.Status = common.Ptr(testkube.RUNNING_TestWorkflowStatus)
	}
	if !initialResult.FinishedAt.IsZero() {
		initialResult.FinishedAt = time.Time{}
	}

	return &notifier{
		result:      initialResult,
		scheduledAt: scheduledAt,

		ch:  make(chan ChannelMessage[Notification]),
		ctx: ctx,
	}
}

func getExecutionId(state watchers2.ExecutionState) string {
	job := state.Job()
	if job == nil {
		return ""
	}
	return job.Name()
}
