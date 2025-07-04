package testkube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
)

func (r *TestWorkflowResult) IsFinished() bool {
	return !r.FinishedAt.IsZero() && r.Status.Finished()
}

func (r *TestWorkflowResult) IsStatus(s TestWorkflowStatus) bool {
	if r == nil || r.Status == nil {
		return s == QUEUED_TestWorkflowStatus
	}
	return *r.Status == s
}

func (r *TestWorkflowResult) LatestTimestamp() time.Time {
	ts := time.Time{}
	if r.FinishedAt.After(ts) {
		ts = r.FinishedAt
	} else if r.StartedAt.After(ts) {
		ts = r.StartedAt
	} else if r.QueuedAt.After(ts) {
		ts = r.QueuedAt
	}
	if r.Initialization.FinishedAt.After(ts) {
		ts = r.Initialization.FinishedAt
	} else if r.Initialization.StartedAt.After(ts) {
		ts = r.Initialization.StartedAt
	} else if r.Initialization.QueuedAt.After(ts) {
		ts = r.Initialization.QueuedAt
	}
	for k := range r.Steps {
		if r.Steps[k].FinishedAt.After(ts) {
			ts = r.Steps[k].FinishedAt
		} else if r.Steps[k].StartedAt.After(ts) {
			ts = r.Steps[k].StartedAt
		} else if r.Steps[k].QueuedAt.After(ts) {
			ts = r.Steps[k].QueuedAt
		}
	}
	return ts
}

func (r *TestWorkflowResult) approxCurrentTimestamp() time.Time {
	ts := latestTimestamp(r.QueuedAt, r.StartedAt, r.Initialization.QueuedAt, r.Initialization.StartedAt, r.Initialization.FinishedAt)
	for i := range r.Steps {
		ts = latestTimestamp(ts, r.Steps[i].QueuedAt, r.Steps[i].StartedAt, r.Steps[i].FinishedAt)
	}
	return ts
}

func (r *TestWorkflowResult) IsQueued() bool {
	return r.IsStatus(QUEUED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsRunning() bool {
	return r.IsStatus(RUNNING_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsFailed() bool {
	return r.IsStatus(FAILED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsAborted() bool {
	return r.IsStatus(ABORTED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsCanceled() bool {
	return r.IsStatus(CANCELED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsPassed() bool {
	return r.IsStatus(PASSED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsNotPassed() bool {
	return r.IsFinished() && !r.IsNotPassed()
}

func (r *TestWorkflowResult) IsPaused() bool {
	return r.IsStatus(PAUSED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsAnyError() bool {
	return r.IsFinished() && !r.IsStatus(PASSED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) HasPauseAt(ref string, t time.Time) bool {
	for _, p := range r.Pauses {
		if ref == p.Ref && !p.PausedAt.After(t) && (p.ResumedAt.IsZero() || !p.ResumedAt.Before(t)) {
			return true
		}
	}
	return false
}

func (r *TestWorkflowResult) HasUnfinishedPause(ref string) bool {
	for _, p := range r.Pauses {
		if ref == p.Ref && p.ResumedAt.IsZero() {
			return true
		}
	}
	return false
}

func (r *TestWorkflowResult) Current(sig []TestWorkflowSignature) string {
	if !r.IsRunning() || r.Initialization.FinishedAt.IsZero() {
		return ""
	}
	current := ""
	walkSteps(sig, func(signature TestWorkflowSignature) {
		if s, ok := r.Steps[signature.Ref]; ok && len(signature.Children) == 0 && !s.QueuedAt.IsZero() && s.FinishedAt.IsZero() && current == "" {
			current = signature.Ref
		}
	})
	return current
}

func (r *TestWorkflowResult) IsAnyStepAborted() bool {
	// When initialization was aborted or failed - it's immediately end
	if r.Initialization.Status.AnyError() {
		return true
	}

	// Analyze the rest of the steps
	for _, step := range r.Steps {
		// When any step was aborted - it's immediately end
		if step.Status.Aborted() {
			return true
		}
	}
	return false
}

func (r *TestWorkflowResult) IsAnyStepCanceled() bool {
	// When initialization was aborted or failed - it's immediately end
	if r.Initialization.Status.AnyError() {
		return true
	}

	// Analyze the rest of the steps
	for _, step := range r.Steps {
		// When any step was aborted - it's immediately end
		if step.Status.Canceled() {
			return true
		}
	}
	return false
}

func (r *TestWorkflowResult) IsAnyStepPaused() bool {
	// When initialization was aborted or failed - it's immediately end
	if r.Initialization.Status.AnyError() {
		return false
	}

	// Analyze the rest of the steps
	for _, step := range r.Steps {
		// When any step was aborted - it's immediately end
		if step.Status.Paused() {
			return true
		}
	}
	return false
}

func (r *TestWorkflowResult) IsKnownStep(ref string) bool {
	if ref == constants.InitStepName {
		return true
	}
	_, ok := r.Steps[ref]
	return ok
}

func (r *TestWorkflowResult) AreAllStepsFinished() bool {
	for _, step := range r.Steps {
		if !step.Finished() {
			return false
		}
	}
	return true
}

// TODO: Optimize
func (r *TestWorkflowResult) Equal(r2 *TestWorkflowResult) bool {
	if r == nil && r2 == nil {
		return true
	}
	if r == nil || r2 == nil {
		return false
	}
	v1, _ := json.Marshal(r)
	v2, _ := json.Marshal(r2)
	return bytes.Equal(v1, v2)
}

func (r *TestWorkflowResult) Fatal(err error, aborted bool, ts time.Time) {
	r.Initialization.ErrorMessage = err.Error()
	r.Status = common.Ptr(FAILED_TestWorkflowStatus)
	r.PredictedStatus = r.Status
	if aborted {
		r.Status = common.Ptr(ABORTED_TestWorkflowStatus)
	}
	if r.QueuedAt.IsZero() {
		r.QueuedAt = ts.UTC()
	}
	if r.StartedAt.IsZero() {
		r.StartedAt = ts.UTC()
	}
	if r.FinishedAt.IsZero() {
		r.FinishedAt = ts.UTC()
	}
	if r.Initialization.Status == nil || !(*r.Initialization.Status).Finished() {
		r.Initialization.Status = common.Ptr(FAILED_TestWorkflowStepStatus)
		if aborted {
			r.Initialization.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
		}
		r.Initialization.FinishedAt = r.FinishedAt
	}
	for i := range r.Steps {
		if r.Steps[i].Status == nil || (*r.Steps[i].Status == QUEUED_TestWorkflowStepStatus) {
			s := r.Steps[i]
			s.Status = common.Ptr(SKIPPED_TestWorkflowStepStatus)
			r.Steps[i] = s
		} else if *r.Steps[i].Status == RUNNING_TestWorkflowStepStatus || *r.Steps[i].Status == PAUSED_TestWorkflowStepStatus {
			s := r.Steps[i]
			s.Status = common.Ptr(FAILED_TestWorkflowStepStatus)
			if aborted {
				s.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
			}
			r.Steps[i] = s
		}
	}
	r.HealDuration(r.QueuedAt)
}

func (r *TestWorkflowResult) Clone() *TestWorkflowResult {
	if r == nil {
		return nil
	}
	steps := make(map[string]TestWorkflowStepResult, len(r.Steps))
	for k, v := range r.Steps {
		steps[k] = *v.Clone()
	}
	return &TestWorkflowResult{
		Status:          r.Status,
		PredictedStatus: r.PredictedStatus,
		QueuedAt:        r.QueuedAt,
		StartedAt:       r.StartedAt,
		FinishedAt:      r.FinishedAt,
		Duration:        r.Duration,
		TotalDuration:   r.TotalDuration,
		DurationMs:      r.DurationMs,
		PausedMs:        r.PausedMs,
		Pauses:          slices.Clone(r.Pauses),
		TotalDurationMs: r.DurationMs + r.PausedMs,
		Initialization:  r.Initialization.Clone(),
		Steps:           steps,
	}
}

func (r *TestWorkflowResult) HealDuration(scheduledAt time.Time) {
	if !r.FinishedAt.IsZero() {
		r.PausedMs = 0

		// Finalize pauses
		for i := range r.Pauses {
			step := r.Steps[r.Pauses[i].Ref]
			if !step.FinishedAt.IsZero() {
				if r.Pauses[i].ResumedAt.IsZero() {
					r.Pauses[i].ResumedAt = step.FinishedAt
				}
				if r.Pauses[i].PausedAt.Before(step.StartedAt) {
					r.Pauses[i].PausedAt = step.StartedAt
				}
				if r.Pauses[i].ResumedAt.Before(r.Pauses[i].PausedAt) {
					r.Pauses[i].PausedAt = r.Pauses[i].ResumedAt
				}
			}
		}

		// Get unique pause periods
		pauses := make([]TestWorkflowPause, 0)
	loop:
		for _, p := range r.Pauses {
			for i := range pauses {
				// They don't overlap
				if p.PausedAt.After(pauses[i].ResumedAt) || p.ResumedAt.Before(pauses[i].PausedAt) {
					continue
				}

				// The existing pause period may take some period before
				if pauses[i].PausedAt.After(p.PausedAt) {
					pauses[i].PausedAt = p.PausedAt
					p.PausedAt = pauses[i].ResumedAt
					if p.ResumedAt.Before(p.PausedAt) {
						p.ResumedAt = p.PausedAt
					}
				}

				// The existing pause period may take some period after
				if pauses[i].ResumedAt.Before(p.ResumedAt) {
					pauses[i].ResumedAt = p.ResumedAt
					p.ResumedAt = pauses[i].PausedAt
				}

				// The pause is already enclosed in existing list
				continue loop
			}

			pauses = append(pauses, p)
		}

		for _, p := range pauses {
			resumedAt := p.ResumedAt
			if resumedAt.IsZero() {
				resumedAt = r.FinishedAt
			}
			milli := int32(resumedAt.Sub(p.PausedAt).Milliseconds())
			if milli > 0 {
				r.PausedMs += milli
			}
		}

		queuedAt := r.QueuedAt
		if queuedAt.IsZero() {
			queuedAt = scheduledAt
		}
		totalDuration := r.FinishedAt.Sub(scheduledAt)
		duration := totalDuration - time.Duration(1e3*r.PausedMs)
		if totalDuration < 0 {
			totalDuration = time.Duration(0)
		}
		if duration < 0 {
			duration = time.Duration(0)
		}
		r.DurationMs = int32(duration.Milliseconds())
		r.Duration = duration.Round(time.Millisecond).String()
		r.TotalDurationMs = int32(totalDuration.Milliseconds())
		r.TotalDuration = totalDuration.Round(time.Millisecond).String()
	}
}

func (r *TestWorkflowResult) HealMissingPauseStatuses() {
	for ref := range r.Steps {
		if !r.Steps[ref].Status.Paused() && !r.Steps[ref].Status.Finished() && r.HasUnfinishedPause(ref) {
			step := r.Steps[ref]
			step.Status = common.Ptr(PAUSED_TestWorkflowStepStatus)
			r.Steps[ref] = step
		}
	}
}

func isStepOptional(sigSequence []TestWorkflowSignature, ref string) bool {
	for i := range sigSequence {
		if sigSequence[i].Ref == ref {
			return sigSequence[i].Optional
		}
	}
	return false
}

func (r *TestWorkflowResult) healPredictedStatus(sigSequence []TestWorkflowSignature) {
	// Mark as aborted, when any step is aborted
	switch {
	case r.Initialization.Status.AnyError(), r.IsAnyStepAborted():
		r.PredictedStatus = common.Ptr(ABORTED_TestWorkflowStatus)
		return
	case r.IsAnyStepCanceled():
		r.PredictedStatus = common.Ptr(CANCELED_TestWorkflowStatus)
		return
	}

	// Determine if there are some steps failed
	for ref := range r.Steps {
		if r.Steps[ref].Status.Aborted() || r.Steps[ref].Status.Canceled() || (r.Steps[ref].Status.AnyError() && !isStepOptional(sigSequence, ref)) {
			r.PredictedStatus = common.Ptr(FAILED_TestWorkflowStatus)
			return
		}
	}
	r.PredictedStatus = common.Ptr(PASSED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) healStatus() {
	if !r.FinishedAt.IsZero() && r.AreAllStepsFinished() {
		r.Status = r.PredictedStatus
	} else if r.IsAnyStepPaused() {
		r.Status = common.Ptr(PAUSED_TestWorkflowStatus)
	} else if !r.StartedAt.IsZero() {
		r.Status = common.Ptr(RUNNING_TestWorkflowStatus)
	}
}

func (r *TestWorkflowResult) HealStatus(sigSequence []TestWorkflowSignature) {
	r.healPredictedStatus(sigSequence)
	r.healStatus()
}

// TODO: Take in account scheduledAt to avoid having timestamp before schedule (because of seconds precision)
func (r *TestWorkflowResult) HealTimestamps(sigSequence []TestWorkflowSignature, scheduledTs, firstContainerStartTs, completionTs time.Time, ended bool) {
	// Adjust queue/start time to the schedule time
	r.QueuedAt = latestTimestamp(r.QueuedAt, scheduledTs)
	r.StartedAt = latestTimestamp(r.StartedAt, r.QueuedAt)

	// Detect the initialization queue time
	r.Initialization.QueuedAt = earliestTimestamp(r.StartedAt, r.Initialization.QueuedAt)

	// Ensure there is the start time for the initialization if it's started or done
	if !r.Initialization.NotStarted() {
		r.Initialization.StartedAt = latestTimestamp(r.Initialization.StartedAt, firstContainerStartTs, r.Initialization.QueuedAt)
	}

	// Ensure there is the end time for the initialization if it's done
	if r.Initialization.Finished() && r.Initialization.FinishedAt.IsZero() {
		// Fallback to have any timestamp in case something went wrong
		if r.Initialization.Aborted() {
			r.Initialization.FinishedAt = latestTimestamp(r.Initialization.StartedAt, completionTs)
		} else {
			r.Initialization.FinishedAt = r.Initialization.StartedAt
		}
	}

	// Set up everywhere queued at time to past finished time
	lastTs := r.Initialization.FinishedAt
	for i, s := range sigSequence {
		if len(s.Children) > 0 {
			continue
		}
		step := r.Steps[s.Ref]
		if !step.QueuedAt.Equal(lastTs) {
			step.QueuedAt = lastTs
		}
		if step.FinishedAt.IsZero() && (step.Status.Aborted() || (step.Status.Skipped() && step.ErrorMessage != "")) {
			step.FinishedAt = completionTs
		}
		if step.FinishedAt.IsZero() && ended {
			if len(sigSequence) > i+1 {
				if !r.Steps[sigSequence[i+1].Ref].QueuedAt.IsZero() {
					step.FinishedAt = r.Steps[sigSequence[i+1].Ref].QueuedAt
				} else if !r.Steps[sigSequence[i+1].Ref].StartedAt.IsZero() {
					step.FinishedAt = r.Steps[sigSequence[i+1].Ref].StartedAt
				} else {
					step.FinishedAt = completionTs
				}
			} else {
				step.FinishedAt = completionTs
			}
		}
		if !step.QueuedAt.IsZero() && !step.FinishedAt.IsZero() && step.StartedAt.IsZero() {
			step.StartedAt = step.QueuedAt
		}

		if !step.StartedAt.IsZero() && step.StartedAt.Before(step.QueuedAt) {
			step.StartedAt = step.QueuedAt
		}
		if !step.FinishedAt.IsZero() && step.FinishedAt.Before(step.StartedAt) {
			step.FinishedAt = step.StartedAt
		}

		r.Steps[s.Ref] = step
		lastTs = step.FinishedAt
	}

	// Set up for groups too.
	// Do it from end, so we handle nested groups
	for i := len(sigSequence) - 1; i >= 0; i-- {
		if len(sigSequence[i].Children) == 0 {
			continue
		}
		s := sigSequence[i]
		seq := s.Sequence()
		expectedQueuedAt := r.Steps[seq[1].Ref].QueuedAt
		expectedFinishedAt := r.Steps[seq[len(seq)-1].Ref].FinishedAt
		step := r.Steps[s.Ref]
		if !r.Steps[s.Ref].QueuedAt.Equal(expectedQueuedAt) {
			step.QueuedAt = expectedQueuedAt
		}
		if !r.Steps[s.Ref].FinishedAt.Equal(expectedFinishedAt) {
			step.FinishedAt = expectedFinishedAt
		}
		r.Steps[s.Ref] = step
	}

	if ended {
		r.FinishedAt = firstNonZero(r.approxCurrentTimestamp(), completionTs)
	}
}

func (r *TestWorkflowResult) HealAbortedOrCanceled(sigSequence []TestWorkflowSignature, errorStr, defaultErrorStr string, terminationCode string) {
	errorMessage := fmt.Sprintf("The execution has been %s. (%s)", terminationCode, errorStr)
	if errorStr == "" {
		errorMessage = fmt.Sprintf("The execution has been %s.", terminationCode)
	}

	// Create marker to know if there is any step marked as aborted or canceled already
	aborted := false
	canceled := false

	// Check the initialization step
	if !r.Initialization.Finished() || r.Initialization.Aborted() || r.Initialization.Canceled() {
		if terminationCode == string(CANCELED_TestWorkflowStatus) {
			canceled = true
			r.Initialization.Status = common.Ptr(CANCELED_TestWorkflowStepStatus)
		} else {
			aborted = true
			r.Initialization.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
		}
		r.Initialization.ErrorMessage = errorMessage
	}

	// Check all the executable steps in the sequence
	for i := range sigSequence {
		ref := sigSequence[i].Ref
		if ref == constants.InitStepName || !r.IsKnownStep(ref) || len(sigSequence[i].Children) > 0 {
			continue
		}
		step := r.Steps[ref]
		if step.Finished() && !step.Aborted() && !step.Canceled() && (!step.Skipped() || step.ErrorMessage == "") {
			if (step.Status.Aborted() || step.Status.Canceled()) && (step.ErrorMessage == "" || step.ErrorMessage == defaultErrorStr) {
				step.ErrorMessage = errorMessage
			}
			continue
		}
		if aborted || canceled {
			step.Status = common.Ptr(SKIPPED_TestWorkflowStepStatus)
			step.ErrorMessage = fmt.Sprintf("The execution was aborted before. %s", color.FgDarkGray.Render("("+errorStr+")"))
			if canceled {
				step.ErrorMessage = fmt.Sprintf("The execution was canceled before. %s", color.FgDarkGray.Render("("+errorStr+")"))
			}
		} else {
			if terminationCode == string(CANCELED_TestWorkflowStatus) {
				canceled = true
				step.Status = common.Ptr(CANCELED_TestWorkflowStepStatus)
			} else {
				aborted = true
				step.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
			}
			step.ErrorMessage = errorMessage
		}
		r.Steps[ref] = step
	}

	// Adjust all the group steps in the sequence.
	// Do it from end, so we can handle nested groups
	for i := len(sigSequence) - 1; i >= 0; i-- {
		ref := sigSequence[i].Ref
		if ref == constants.InitStepName || !r.IsKnownStep(ref) || len(sigSequence[i].Children) == 0 {
			continue
		}
		step := r.Steps[ref]
		if step.Finished() {
			continue
		}
		allSkipped := true
		anyAborted := false
		anyCanceled := false
		for _, childSig := range sigSequence[i].Children {
			// TODO: What about nested virtual groups? We don't have their statuses
			if !r.IsKnownStep(childSig.Ref) {
				continue
			}
			if r.Steps[childSig.Ref].Status.Aborted() {
				anyAborted = true
			}
			if r.Steps[childSig.Ref].Status.Canceled() {
				anyCanceled = true
			}
			if !r.Steps[childSig.Ref].Status.Skipped() {
				allSkipped = false
			}
		}
		if allSkipped {
			step.Status = common.Ptr(SKIPPED_TestWorkflowStepStatus)
		} else if anyAborted {
			step.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
		} else if anyCanceled {
			step.Status = common.Ptr(CANCELED_TestWorkflowStepStatus)
		}
		r.Steps[ref] = step
	}

	// The rest of steps is unrecognized, so just mark them as aborted with information about faulty state
	for ref, step := range r.Steps {
		if step.Finished() {
			continue
		}
		if r.IsCanceled() {
			step.Status = common.Ptr(CANCELED_TestWorkflowStepStatus)
			step.ErrorMessage = fmt.Sprintf("The execution was canceled, but we could not determine steps order: %s", errorStr)
		} else {
			step.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
			step.ErrorMessage = fmt.Sprintf("The execution was aborted, but we could not determine steps order: %s", errorStr)
		}
		r.Steps[ref] = step
	}
}

func walkSteps(sig []TestWorkflowSignature, fn func(signature TestWorkflowSignature)) {
	for _, s := range sig {
		walkSteps(s.Children, fn)
		fn(s)
	}
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
