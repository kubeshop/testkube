package testkube

type TestWorkflowWithExecutions []TestWorkflowWithExecution

func (t TestWorkflowWithExecutions) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Created", "Labels", "Status", "Execution ID"}
	for _, e := range t {
		status := ""
		executionID := ""
		if e.LatestExecution != nil {
			executionID = e.LatestExecution.Id
			if e.LatestExecution.Result != nil && e.LatestExecution.Result.Status != nil {
				status = string(*e.LatestExecution.Result.Status)
			}
		}

		output = append(output, []string{
			e.Workflow.Name,
			e.Workflow.Description,
			e.Workflow.Created.String(),
			MapToString(e.Workflow.Labels),
			status,
			executionID,
		})
	}

	return
}
