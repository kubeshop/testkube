package testkube

func (w *TestWorkflowStepParallel) ContainsExecuteAction() bool {
	if w.Execute != nil && (len(w.Execute.Tests) != 0 || len(w.Execute.Workflows) != 0) {
		return true
	}

	steps := append(w.Setup, append(w.Steps, w.After...)...)
	for _, step := range steps {
		if step.ContainsExecuteAction() {
			return true
		}
	}

	return false
}
