package testkube

import (
	"time"

	"github.com/kubeshop/testkube/internal/common"
)

func (r *TestWorkflowResult) IsFinished() bool {
	return !r.IsStatus(QUEUED_TestWorkflowStatus) && !r.IsStatus(RUNNING_TestWorkflowStatus)
}

func (r *TestWorkflowResult) IsStatus(s TestWorkflowStatus) bool {
	if r == nil || r.Status == nil {
		return s == QUEUED_TestWorkflowStatus
	}
	return *r.Status == s
}

func (r *TestWorkflowResult) IsQueued() bool {
	return r.IsStatus(QUEUED_TestWorkflowStatus)
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
		} else if *r.Steps[i].Status == RUNNING_TestWorkflowStepStatus {
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
		DurationMs:      r.DurationMs,
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
		r.Duration = r.FinishedAt.Sub(r.QueuedAt).Round(time.Millisecond).String()
		r.DurationMs = int32(r.FinishedAt.Sub(r.QueuedAt).Milliseconds())
	}
}

func (r *TestWorkflowResult) Recompute(sig []TestWorkflowSignature, scheduledAt time.Time) {
	// Recompute steps
	for _, ch := range sig {
		r.RecomputeStep(ch)
	}

	// Compute the duration
	r.RecomputeDuration()

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
		if r.Initialization.QueuedAt.Before(r.QueuedAt) {
			r.QueuedAt = r.Initialization.QueuedAt
		}
		if r.Initialization.StartedAt.Before(r.StartedAt) {
			r.StartedAt = r.Initialization.StartedAt
		}
	}
	last := r.Steps[sig[len(sig)-1].Ref]
	r.FinishedAt = adjustMinimumTime(r.FinishedAt, last.FinishedAt)

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
	}
}

func (r *TestWorkflowResult) RecomputeStep(sig TestWorkflowSignature) {
	children := sig.Children
	if len(children) == 0 {
		return
	}

	// Compute nested steps
	for _, ch := range children {
		r.RecomputeStep(ch)
	}

	// Simplify accessing value
	v := r.Steps[sig.Ref]
	defer func() {
		r.Steps[sig.Ref] = v
	}()

	// Compute time
	v = recomputeTestWorkflowStepResult(v, sig, r)
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
		if getTestWorkflowStepStatus(v) == QUEUED_TestWorkflowStepStatus || getTestWorkflowStepStatus(v) == RUNNING_TestWorkflowStepStatus {
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
		if status == QUEUED_TestWorkflowStepStatus || status == RUNNING_TestWorkflowStepStatus {
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
		if finished {
			v.Status = common.Ptr(predicted)
		}
		return v
	}

	if !v.StartedAt.IsZero() {
		v.Status = common.Ptr(RUNNING_TestWorkflowStepStatus)
	}

	return v
}
