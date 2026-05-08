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
