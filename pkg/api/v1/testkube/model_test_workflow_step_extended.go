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

	if w.Parallel != nil && w.Parallel.ContainsExecuteAction() {
		return true
	}

	return false
}

// AgentSteps returns the agent operation of the step, then the ones of its nested steps
// and of its parallel step.
func (w *TestWorkflowStep) AgentSteps() []TestWorkflowStepAgent {
	var agents []TestWorkflowStepAgent
	if w.Agent != nil {
		agents = append(agents, *w.Agent)
	}
	for _, steps := range [][]TestWorkflowStep{w.Setup, w.Steps} {
		for i := range steps {
			agents = append(agents, steps[i].AgentSteps()...)
		}
	}
	if w.Parallel != nil {
		agents = append(agents, w.Parallel.AgentSteps()...)
	}
	return agents
}

func (w *TestWorkflowStep) GetTemplateRefs() []TestWorkflowTemplateRef {
	var templateRefs []TestWorkflowTemplateRef

	if w.Template != nil {
		templateRefs = append(templateRefs, *w.Template)
	}

	steps := append(w.Setup, w.Steps...)
	for _, step := range steps {
		templateRefs = append(templateRefs, step.GetTemplateRefs()...)
	}

	if w.Parallel != nil {
		templateRefs = append(templateRefs, w.Parallel.GetTemplateRefs()...)
	}

	return templateRefs
}

func (w *TestWorkflowStep) HasService(name string) bool {
	if w.Parallel != nil && w.Parallel.HasService(name) {
		return true
	}

	steps := append(w.Setup, w.Steps...)
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
