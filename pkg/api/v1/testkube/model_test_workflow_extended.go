package testkube

type TestWorkflows []TestWorkflow

func (t TestWorkflows) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Created", "Labels"}
	for _, e := range t {
		output = append(output, []string{
			e.Name,
			e.Description,
			e.Created.String(),
			MapToString(e.Labels),
		})
	}

	return
}
