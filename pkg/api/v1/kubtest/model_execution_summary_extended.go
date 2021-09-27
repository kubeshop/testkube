package kubtest

type ExecutionsSummary []ExecutionSummary

func (executions ExecutionsSummary) Table() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Name", "ID", "Status"}

	for _, e := range executions {
		var status string
		if e.Status != nil {
			status = string(*e.Status)
		}
		output = append(output, []string{
			e.ScriptName,
			e.ScriptType,
			e.Name,
			e.Id,
			string(*e.Status),
		})
	}

	return
}
