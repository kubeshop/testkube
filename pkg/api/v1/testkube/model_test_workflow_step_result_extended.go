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

func (r *TestWorkflowStepResult) Finished() bool {
	return r.Status.Finished()
}

func (r *TestWorkflowStepResult) Aborted() bool {
	return r.Status.Aborted()
}

func (r *TestWorkflowStepResult) Canceled() bool {
	return r.Status.Canceled()
}

func (r *TestWorkflowStepResult) Skipped() bool {
	return r.Status.Skipped()
}

func (r *TestWorkflowStepResult) NotStarted() bool {
	return r.Status.NotStarted()
}
