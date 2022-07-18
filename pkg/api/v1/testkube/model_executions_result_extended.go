package testkube

func (result ExecutionsResult) Table() (header []string, output [][]string) {
	header = []string{"ID", "Name", "Test Name", "Type", "Status", "Labels"}

	for _, e := range result.Results {
		var status string
		if e.Status != nil {
			status = string(*e.Status)
		}
		output = append(output, []string{
			e.Id,
			e.Name,
			e.TestName,
			e.TestType,
			status,
			MapToString(e.Labels),
		})
	}

	return
}
