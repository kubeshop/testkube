package testkube

import (
	"fmt"
	"slices"
	"time"

	"github.com/kubeshop/testkube/internal/common"
)

func (r *TestWorkflowResult) IsFinished() bool {
	return !r.IsStatus(QUEUED_TestWorkflowStatus) && !r.IsStatus(RUNNING_TestWorkflowStatus) && !r.IsStatus(PAUSED_TestWorkflowStatus)
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

func (r *TestWorkflowResult) IsPassed() bool {
	return r.IsStatus(PASSED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsPaused() bool {
	return r.IsStatus(PAUSED_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsAnyError() bool {
	return r.IsFinished() && !r.IsStatus(PASSED_TestWorkflowStatus)
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
	if r.Initialization.Status == nil || (*r.Initialization.Status == QUEUED_TestWorkflowStepStatus) || (*r.Initialization.Status == RUNNING_TestWorkflowStepStatus) {
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
	r.RecomputeDuration()
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

func getTestWorkflowStepStatus(result TestWorkflowStepResult) TestWorkflowStepStatus {
	if result.Status == nil {
		return QUEUED_TestWorkflowStepStatus
	}
	return *result.Status
}

func (r *TestWorkflowResult) UpdateStepResult(sig []TestWorkflowSignature, ref string, result TestWorkflowStepResult, scheduledAt time.Time) TestWorkflowStepResult {
	v := r.Steps[ref]
	v.Merge(result)
	r.Steps[ref] = v
	r.Recompute(sig, scheduledAt)
	return v
}

func (r *TestWorkflowResult) RecomputeDuration() {
	if !r.FinishedAt.IsZero() {
		r.PausedMs = 0

		// Get unique pause periods
		pauses := make([]TestWorkflowPause, 0)
	loop:
		for _, p := range r.Pauses {
			// Finalize the pause if it's not
			step := r.Steps[p.Ref]
			if !step.FinishedAt.IsZero() && p.ResumedAt.IsZero() {
				p.ResumedAt = step.FinishedAt
			}

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

		totalDuration := r.FinishedAt.Sub(r.QueuedAt)
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

func (r *TestWorkflowResult) PauseStart(sig []TestWorkflowSignature, scheduledAt time.Time, ref string, start time.Time) {
	if r.HasPauseAt(ref, start) {
		fmt.Println("     Already had a pause")
		return
	}
	fmt.Println("     Added a pause entry")
	r.Pauses = append(r.Pauses, TestWorkflowPause{Ref: ref, PausedAt: start})
	r.Recompute(sig, scheduledAt)
}

func (r *TestWorkflowResult) PauseEnd(sig []TestWorkflowSignature, scheduledAt time.Time, ref string, end time.Time) {
	for i, p := range r.Pauses {
		if p.Ref != ref {
			continue
		}
		fmt.Println("     Found pause for ref possibly to resume", ref)
		if !p.PausedAt.After(end) && !p.ResumedAt.Before(end) {
			// It's already covered by another period
			fmt.Println("     It's already covered", p.PausedAt, p.ResumedAt)
			return
		}
		if !p.PausedAt.After(end) && (p.ResumedAt.IsZero() || p.ResumedAt.Equal(end)) {
			// It found a period to fulfill
			fmt.Println("     Filling the pause from", p.PausedAt)
			r.Pauses[i].ResumedAt = end
			r.Recompute(sig, scheduledAt)
			return
		}
	}
}

func (r *TestWorkflowResult) Recompute(sig []TestWorkflowSignature, scheduledAt time.Time) {
	// Recompute steps
	for _, ch := range sig {
		r.RecomputeStep(ch)
	}

	// Build status on the internal failure
	if getTestWorkflowStepStatus(*r.Initialization) == ABORTED_TestWorkflowStepStatus {
		r.Status = common.Ptr(ABORTED_TestWorkflowStatus)
		r.PredictedStatus = r.Status
		return
	} else if getTestWorkflowStepStatus(*r.Initialization) == FAILED_TestWorkflowStepStatus {
		r.Status = common.Ptr(FAILED_TestWorkflowStatus)
		r.PredictedStatus = r.Status
		return
	}

	// Calibrate the execution time initially
	r.QueuedAt = adjustMinimumTime(r.QueuedAt, scheduledAt)
	r.StartedAt = adjustMinimumTime(r.StartedAt, r.QueuedAt)
	r.FinishedAt = adjustMinimumTime(r.FinishedAt, r.StartedAt)
	initialDate := r.StartedAt
	if initialDate.IsZero() {
		initialDate = r.QueuedAt
	}

	// Calibrate the initialization step
	if r.Initialization != nil {
		r.Initialization.QueuedAt = adjustMinimumTime(r.Initialization.QueuedAt, initialDate)
		r.Initialization.StartedAt = adjustMinimumTime(r.Initialization.StartedAt, r.Initialization.QueuedAt)
		r.Initialization.FinishedAt = adjustMinimumTime(r.Initialization.FinishedAt, r.Initialization.StartedAt)
		initialDate = getLastDate(*r.Initialization, initialDate)
	}

	// Prepare sequential list of container steps
	type step struct {
		ref    string
		result TestWorkflowStepResult
	}
	seq := make([]step, 0)
	walkSteps(sig, func(s TestWorkflowSignature) {
		if len(s.Children) > 0 {
			return
		}
		seq = append(seq, step{ref: s.Ref, result: r.Steps[s.Ref]})
	})

	// Calibrate the clock for container steps
	for i := 0; i < len(seq); i++ {
		if i != 0 {
			initialDate = getLastDate(seq[i-1].result, initialDate)
		}
		seq[i].result.QueuedAt = initialDate
		seq[i].result.StartedAt = adjustMinimumTime(seq[i].result.StartedAt, seq[i].result.QueuedAt)
		seq[i].result.FinishedAt = adjustMinimumTime(seq[i].result.FinishedAt, seq[i].result.StartedAt)
	}
	for _, s := range seq {
		r.Steps[s.ref] = s.result
	}

	// Calibrate the clock for group steps
	walkSteps(sig, func(s TestWorkflowSignature) {
		if len(s.Children) == 0 {
			return
		}
		first := getFirstContainer(s.Children)
		last := getLastContainer(s.Children)
		if first == nil || last == nil {
			return
		}
		res := r.Steps[s.Ref]
		res.QueuedAt = r.Steps[first.Ref].QueuedAt
		res.StartedAt = r.Steps[first.Ref].StartedAt
		res.FinishedAt = r.Steps[last.Ref].FinishedAt
		r.Steps[s.Ref] = res
	})

	// Calibrate execution clock
	if r.Initialization != nil {
		if !r.Initialization.QueuedAt.IsZero() && r.Initialization.QueuedAt.Before(r.QueuedAt) {
			r.QueuedAt = r.Initialization.QueuedAt
		}
		if !r.Initialization.StartedAt.IsZero() && r.Initialization.StartedAt.Before(r.StartedAt) {
			r.StartedAt = r.Initialization.StartedAt
		}
	}
	last := r.Steps[sig[len(sig)-1].Ref]
	if !last.FinishedAt.IsZero() {
		r.FinishedAt = adjustMinimumTime(r.FinishedAt, last.FinishedAt)
	}

	// Check pause status
	isPaused := false
	walkSteps(sig, func(s TestWorkflowSignature) {
		if r.Steps[s.Ref].Status != nil && *r.Steps[s.Ref].Status == PAUSED_TestWorkflowStepStatus {
			isPaused = true
		}
	})

	// Recompute the TestWorkflow status
	totalSig := TestWorkflowSignature{Children: sig}
	result, _ := predictTestWorkflowStepStatus(TestWorkflowStepResult{}, totalSig, r)
	status := common.Ptr(FAILED_TestWorkflowStatus)
	switch result {
	case ABORTED_TestWorkflowStepStatus:
		status = common.Ptr(ABORTED_TestWorkflowStatus)
	case PASSED_TestWorkflowStepStatus, SKIPPED_TestWorkflowStepStatus:
		status = common.Ptr(PASSED_TestWorkflowStatus)
	}
	r.PredictedStatus = status
	if !r.FinishedAt.IsZero() || *status == ABORTED_TestWorkflowStatus {
		r.Status = r.PredictedStatus
	} else if isPaused {
		r.Status = common.Ptr(PAUSED_TestWorkflowStatus)
	}

	if !isPaused && r.Status != nil && *r.Status == PAUSED_TestWorkflowStatus {
		r.Status = common.Ptr(RUNNING_TestWorkflowStatus)
	}

	if r.FinishedAt.IsZero() && r.Status != nil && *r.Status == ABORTED_TestWorkflowStatus {
		r.FinishedAt = r.LatestTimestamp()
	}

	// Compute the duration
	r.RecomputeDuration()
}

func (r *TestWorkflowResult) RecomputeStep(sig TestWorkflowSignature) {
	// Compute nested steps
	for _, ch := range sig.Children {
		r.RecomputeStep(ch)
	}

	// Simplify accessing value
	v := r.Steps[sig.Ref]
	defer func() {
		r.Steps[sig.Ref] = v
	}()

	// Compute time
	v = recomputeTestWorkflowStepResult(v, sig, r)

	// Mark as paused during pause period
	if r.HasUnfinishedPause(sig.Ref) && !r.IsFinished() && v.FinishedAt.IsZero() {
		v.Status = common.Ptr(PAUSED_TestWorkflowStepStatus)
	} else if v.Status != nil && *v.Status == PAUSED_TestWorkflowStepStatus {
		v.Status = common.Ptr(RUNNING_TestWorkflowStepStatus)
	}
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

func walkSteps(sig []TestWorkflowSignature, fn func(signature TestWorkflowSignature)) {
	for _, s := range sig {
		walkSteps(s.Children, fn)
		fn(s)
	}
}

func getFirstContainer(sig []TestWorkflowSignature) *TestWorkflowSignature {
	for i := 0; i < len(sig); i++ {
		s := sig[i]
		if len(s.Children) == 0 {
			return &s
		}
		c := getFirstContainer(s.Children)
		if c != nil {
			return c
		}
	}
	return nil
}

func getLastContainer(sig []TestWorkflowSignature) *TestWorkflowSignature {
	for i := len(sig) - 1; i >= 0; i-- {
		s := sig[i]
		if len(s.Children) == 0 {
			return &s
		}
		c := getLastContainer(s.Children)
		if c != nil {
			return c
		}
	}
	return nil
}

func getLastDate(r TestWorkflowStepResult, def time.Time) time.Time {
	if !r.FinishedAt.IsZero() {
		return r.FinishedAt
	}
	if !r.StartedAt.IsZero() {
		return r.StartedAt
	}
	if !r.QueuedAt.IsZero() {
		return r.QueuedAt
	}
	return def
}

func adjustMinimumTime(dst, min time.Time) time.Time {
	if dst.IsZero() {
		return dst
	}
	if min.After(dst) {
		return min
	}
	return dst
}

func predictTestWorkflowStepStatus(v TestWorkflowStepResult, sig TestWorkflowSignature, r *TestWorkflowResult) (TestWorkflowStepStatus, bool) {
	children := sig.Children
	if len(children) == 0 {
		if !getTestWorkflowStepStatus(v).Finished() {
			return PASSED_TestWorkflowStepStatus, false
		}
		return *v.Status, true
	}

	// Compute the status
	skipped := true
	aborted := false
	failed := false
	finished := true
	for _, ch := range children {
		status := getTestWorkflowStepStatus(r.Steps[ch.Ref])
		if status != SKIPPED_TestWorkflowStepStatus {
			skipped = false
		}
		if status == ABORTED_TestWorkflowStepStatus {
			aborted = true
		}
		if !ch.Optional && (status == FAILED_TestWorkflowStepStatus || status == TIMEOUT_TestWorkflowStepStatus) {
			failed = true
		}
		if !status.Finished() {
			finished = false
		}
	}

	if getTestWorkflowStepStatus(v) == FAILED_TestWorkflowStepStatus {
		return FAILED_TestWorkflowStepStatus, finished
	} else if aborted {
		return ABORTED_TestWorkflowStepStatus, finished
	} else if (failed && !sig.Negative) || (!failed && sig.Negative) {
		return FAILED_TestWorkflowStepStatus, finished
	} else if skipped {
		return SKIPPED_TestWorkflowStepStatus, finished
	} else {
		return PASSED_TestWorkflowStepStatus, finished
	}
}

func recomputeTestWorkflowStepResult(v TestWorkflowStepResult, sig TestWorkflowSignature, r *TestWorkflowResult) TestWorkflowStepResult {
	// Ensure there is a queue time if the step is already started
	if v.QueuedAt.IsZero() {
		if !v.StartedAt.IsZero() {
			v.QueuedAt = v.StartedAt
		} else if !v.FinishedAt.IsZero() {
			v.QueuedAt = v.FinishedAt
		}
	}

	// Ensure there is a start time if the step is already finished
	if v.StartedAt.IsZero() && !v.FinishedAt.IsZero() {
		v.StartedAt = v.QueuedAt
	}

	// Compute children
	children := sig.Children
	if len(children) == 0 {
		return v
	}

	// Compute nested steps
	for _, ch := range children {
		r.RecomputeStep(ch)
	}

	// Compute time
	v.QueuedAt = r.Steps[children[0].Ref].QueuedAt
	v.StartedAt = r.Steps[children[0].Ref].StartedAt
	v.FinishedAt = r.Steps[children[len(children)-1].Ref].StartedAt

	// It has been already marked as failed internally from some step below
	if getTestWorkflowStepStatus(v) == FAILED_TestWorkflowStepStatus {
		return v
	}

	// It is finished already
	if !v.FinishedAt.IsZero() {
		predicted, finished := predictTestWorkflowStepStatus(v, sig, r)
		if finished && (v.Status == nil || !(*v.Status).Finished()) {
			v.Status = common.Ptr(predicted)
		}
		return v
	}

	if !v.StartedAt.IsZero() {
		v.Status = common.Ptr(RUNNING_TestWorkflowStepStatus)
	}

	return v
}
