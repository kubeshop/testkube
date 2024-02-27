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
