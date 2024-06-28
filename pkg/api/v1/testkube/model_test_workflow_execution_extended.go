package testkube

import "github.com/kubeshop/testkube/pkg/utils"

type TestWorkflowExecutions []TestWorkflowExecution

func (executions TestWorkflowExecutions) Table() (header []string, output [][]string) {
	header = []string{"Id", "Name", "Test Workflow Name", "Status", "Labels"}

	for _, e := range executions {
		status := "unknown"
		if e.Result != nil && e.Result.Status != nil {
			status = string(*e.Result.Status)
		}

		output = append(output, []string{
			e.Id,
			e.Name,
			e.Workflow.Name,
			status,
			MapToString(e.Workflow.Labels),
		})
	}

	return
}

func (e *TestWorkflowExecution) ConvertDots(fn func(string) string) *TestWorkflowExecution {
	e.Workflow.ConvertDots(fn)
	e.ResolvedWorkflow.ConvertDots(fn)
	return e
}

func (e *TestWorkflowExecution) EscapeDots() *TestWorkflowExecution {
	return e.ConvertDots(utils.EscapeDots)
}

func (e *TestWorkflowExecution) UnscapeDots() *TestWorkflowExecution {
	return e.ConvertDots(utils.UnescapeDots)
}

func (e *TestWorkflowExecution) GetNamespace(defaultNamespace string) string {
	if e.Namespace == "" {
		return defaultNamespace
	}
	return e.Namespace
}

func (e *TestWorkflowExecution) ContainsExecuteAction() bool {
	if e == nil {
		return false
	}

	if e.ResolvedWorkflow == nil || e.ResolvedWorkflow.Spec == nil {
		return false
	}

	steps := append(e.ResolvedWorkflow.Spec.Setup, append(e.ResolvedWorkflow.Spec.Steps, e.ResolvedWorkflow.Spec.After...)...)
	for _, step := range steps {
		if step.ContainsExecuteAction() {
			return true
		}
	}

	return false
}

func (e *TestWorkflowExecution) GetTemplateRefs() []TestWorkflowTemplateRef {
	if e == nil {
		return nil
	}

	if e.ResolvedWorkflow == nil || e.ResolvedWorkflow.Spec == nil {
		return nil
	}

	var templateRefs []TestWorkflowTemplateRef
	steps := append(e.ResolvedWorkflow.Spec.Setup, append(e.ResolvedWorkflow.Spec.Steps, e.ResolvedWorkflow.Spec.After...)...)
	for _, step := range steps {
		templateRefs = append(templateRefs, step.GetTemplateRefs()...)
	}

	return templateRefs
}
