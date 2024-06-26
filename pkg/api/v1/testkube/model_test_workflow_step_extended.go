package testkube

import "github.com/kubeshop/testkube/pkg/utils"

func (w *TestWorkflowStep) ConvertDots(fn func(string) string) *TestWorkflowStep {
	if w == nil {
		return w
	}
	for i := range w.Use {
		if w.Use[i].Config != nil {
			w.Use[i].Config = convertDotsInMap(w.Use[i].Config, fn)
		}
	}
	if w.Template != nil && w.Template.Config != nil {
		w.Template.Config = convertDotsInMap(w.Template.Config, fn)
	}
	for i := range w.Steps {
		w.Steps[i].ConvertDots(fn)
	}
	return w
}

func (w *TestWorkflowStep) EscapeDots() *TestWorkflowStep {
	return w.ConvertDots(utils.EscapeDots)
}

func (w *TestWorkflowStep) UnscapeDots() *TestWorkflowStep {
	return w.ConvertDots(utils.UnescapeDots)
}

func (w *TestWorkflowStep) ContainsExecuteAction() bool {
	if w.Execute != nil && (len(w.Execute.Tests) != 0 || len(w.Execute.Workflows) != 0) {
		return true
	}

	steps := append(w.Setup, w.Steps...)
	for _, step := range steps {
		if step.ContainsExecuteAction() {
			return true
		}
	}

	if w.Parallel.ContainsExecuteAction() {
		return true
	}

	return false
}
