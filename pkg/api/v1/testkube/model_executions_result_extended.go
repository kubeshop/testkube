package testkube

func (result ExecutionsResult) Table() (header []string, output [][]string) {
	header = []string{"Name", "Type", "Name", "ID", "Status", "Labels"}

	for _, e := range result.Results {
		var status string
		if e.Status != nil {
			status = string(*e.Status)
		}
		output = append(output, []string{
			e.TestName,
			e.TestType,
			e.Name,
			e.Id,
			status,
			LabelsToString(e.Labels),
		})
	}

	return
}
