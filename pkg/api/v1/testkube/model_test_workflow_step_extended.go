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
