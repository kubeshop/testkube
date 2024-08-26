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

func (t TestWorkflowWithExecution) GetName() string {
	return t.Workflow.Name
}

func (t TestWorkflowWithExecution) GetNamespace() string {
	return t.Workflow.Namespace
}

func (t TestWorkflowWithExecution) GetLabels() map[string]string {
	return t.Workflow.Labels
}

func (t TestWorkflowWithExecution) GetAnnotations() map[string]string {
	return t.Workflow.Annotations
}
