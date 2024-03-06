package testkube

func (r *TestWorkflowStepResult) Clone() *TestWorkflowStepResult {
	if r == nil {
		return nil
	}
	return &TestWorkflowStepResult{
		ErrorMessage: r.ErrorMessage,
		Status:       r.Status,
		ExitCode:     r.ExitCode,
		QueuedAt:     r.QueuedAt,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
	}
}

func (r *TestWorkflowStepResult) Merge(next TestWorkflowStepResult) {
	if next.ErrorMessage != "" {
		r.ErrorMessage = next.ErrorMessage
	}
	if next.Status != nil {
		r.Status = next.Status
	}
	if next.ExitCode != 0 && (r.ExitCode == 0 || r.ExitCode == -1) {
		r.ExitCode = next.ExitCode
	}
	if !next.QueuedAt.IsZero() {
		r.QueuedAt = next.QueuedAt
	}
	if !next.StartedAt.IsZero() {
		r.StartedAt = next.StartedAt
	}
	if !next.FinishedAt.IsZero() {
		r.FinishedAt = next.FinishedAt
	}
}
