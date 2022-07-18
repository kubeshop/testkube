package testkube

type TestWithExecutions []TestWithExecution

func (t TestWithExecutions) Table() (header []string, output [][]string) {
	header = []string{"Name", "Type", "Created", "Labels", "Schedule", "Status", "Execution id"}
	for _, e := range t {
		if e.Test == nil {
			continue
		}

		status := ""
		executionID := ""
		if e.LatestExecution != nil {
			executionID = e.LatestExecution.Id
			if e.LatestExecution.ExecutionResult != nil && e.LatestExecution.ExecutionResult.Status != nil {
				status = string(*e.LatestExecution.ExecutionResult.Status)
			}
		}
		output = append(output, []string{
			e.Test.Name,
			e.Test.Type_,
			e.Test.Created.String(),
			MapToString(e.Test.Labels),
			e.Test.Schedule,
			status,
			executionID,
		})
	}

	return
}

func (t TestWithExecution) GetObjectRef() *ObjectRef {
	name := ""
	if t.Test != nil {
		name = t.Test.Name
	}

	return &ObjectRef{
		Name:      name,
		Namespace: "testkube",
	}
}
