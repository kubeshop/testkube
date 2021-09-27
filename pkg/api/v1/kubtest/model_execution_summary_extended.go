package kubtest

type ExecutionsSummary []ExecutionSummary

func (executions ExecutionsSummary) Table() (header []string, output [][]string) {
	header = []string{"Script", "Type", "Name", "ID", "Status"}

	for _, e := range executions {
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
