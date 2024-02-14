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

func (r *TestWorkflowResult) Fatal(err error) {
	r.Initialization.ErrorMessage = err.Error()
	r.Status = common.Ptr(FAILED_TestWorkflowStatus)
	r.PredictedStatus = r.Status
	if r.Initialization.Status == nil || (*r.Initialization.Status == QUEUED_TestWorkflowStepStatus) || (*r.Initialization.Status == RUNNING_TestWorkflowStepStatus) {
		r.Initialization.Status = common.Ptr(FAILED_TestWorkflowStepStatus)
	}
	for i := range r.Steps {
		if r.Steps[i].Status == nil || (*r.Steps[i].Status == QUEUED_TestWorkflowStepStatus) || (*r.Steps[i].Status == RUNNING_TestWorkflowStepStatus) {
			s := r.Steps[i]
			s.Status = common.Ptr(SKIPPED_TestWorkflowStepStatus)
			r.Steps[i] = s
		}
	}
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

func (r *TestWorkflowResult) UpdateStepResult(sig []TestWorkflowSignature, ref string, result TestWorkflowStepResult) TestWorkflowStepResult {
	v := r.Steps[ref]
	v.Merge(result)
	r.Steps[ref] = v
	r.Recompute(sig)
	return v
}

func (r *TestWorkflowResult) Recompute(sig []TestWorkflowSignature) {
	// Recompute steps
	for _, ch := range sig {
		r.RecomputeStep(ch)
	}

	// Compute the duration
	if !r.FinishedAt.IsZero() {
		r.Duration = r.FinishedAt.Sub(r.QueuedAt).Round(time.Millisecond).String()
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
