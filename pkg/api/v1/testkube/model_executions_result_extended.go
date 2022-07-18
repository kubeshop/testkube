package testkube

import "strconv"

func (result ExecutionsResult) Table() (header []string, output [][]string) {
	header = []string{"ID", "Name", "Type", "Number", "Status", "Labels"}

	for _, e := range result.Results {
		var status string
		if e.Status != nil {
			status = string(*e.Status)
		}
		output = append(output, []string{
			e.Id,
			e.TestName,
			e.TestType,
			strconv.Itoa(e.Number),
			status,
			MapToString(e.Labels),
		})
	}

	return
}
