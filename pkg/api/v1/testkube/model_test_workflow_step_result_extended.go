package testkube

func (r *TestWorkflowStepResult) Clone() *TestWorkflowStepResult {
	if r == nil {
		return nil
	}
	return &TestWorkflowStepResult{
		ErrorMessage: r.ErrorMessage,
		ErrorReason:  r.ErrorReason,
		Status:       r.Status,
		ExitCode:     r.ExitCode,
		Attempts:     r.Attempts,
		QueuedAt:     r.QueuedAt,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
	}
}
