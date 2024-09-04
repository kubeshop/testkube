package testkube

import "strings"

type TestWorkflowTemplates []TestWorkflowTemplate

func (t TestWorkflowTemplates) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Created", "Labels"}
	for _, e := range t {
		output = append(output, []string{
			strings.ReplaceAll(e.Name, "--", "/"),
			e.Description,
			e.Created.String(),
			MapToString(e.Labels),
		})
	}

	return
}

func (w TestWorkflowTemplate) GetName() string {
	return w.Name
}

func (w TestWorkflowTemplate) GetNamespace() string {
	return w.Namespace
}

func (w TestWorkflowTemplate) GetLabels() map[string]string {
	return w.Labels
}

func (w TestWorkflowTemplate) GetAnnotations() map[string]string {
	return w.Annotations
}
