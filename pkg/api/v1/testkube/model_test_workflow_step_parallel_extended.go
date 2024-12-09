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

func (w *TestWorkflowStepParallel) GetTemplateRefs() []TestWorkflowTemplateRef {
	var templateRefs []TestWorkflowTemplateRef

	if w.Template != nil {
		templateRefs = append(templateRefs, *w.Template)
	}

	steps := append(w.Setup, append(w.Steps, w.After...)...)
	for _, step := range steps {
		templateRefs = append(templateRefs, step.GetTemplateRefs()...)
	}

	return templateRefs
}

func (w *TestWorkflowStepParallel) HasService(name string) bool {
	steps := append(w.Setup, append(w.Steps, w.After...)...)
	for _, step := range steps {
		if step.HasService(name) {
			return true
		}
	}

	if _, ok := w.Services[name]; ok {
		return true
	}

	return false
}
