package testkube

import (
	"bytes"
	"encoding/json"
	"slices"
	"time"

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

func (r *TestWorkflowResult) RecomputeDuration() {
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

func walkSteps(sig []TestWorkflowSignature, fn func(signature TestWorkflowSignature)) {
	for _, s := range sig {
		walkSteps(s.Children, fn)
		fn(s)
	}
}
