package testkube

// the result for a page of executions
func (result TestExecutionsResult) Table() (header []string, output [][]string) {
	header = []string{"Test", "Name", "ID", "Status"}

	for _, e := range result.Results {
		var status string
		if e.Status != nil {
			status = string(*e.Status)
		}
		output = append(output, []string{
			e.TestName,
			e.Name,
			e.Id,
			status,
		})
	}

	return
}
