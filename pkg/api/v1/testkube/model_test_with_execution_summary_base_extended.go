package testkube

type TestWithExecutionSummaries []TestWithExecutionSummary

func (t TestWithExecutionSummaries) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Type", "Created", "Labels", "Schedule", "Status", "Execution id"}
	for _, e := range t {
		if e.Test == nil {
			continue
		}

		status := ""
		executionID := ""
		if e.LatestExecution != nil {
			executionID = e.LatestExecution.Id
			if e.LatestExecution.Status != nil {
				status = string(*e.LatestExecution.Status)
			}
		}
		output = append(output, []string{
			e.Test.Name,
			e.Test.Description,
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

func (t TestWithExecutionSummary) GetObjectRef(namespace string) *ObjectRef {
	name := ""
	if t.Test != nil {
		name = t.Test.Name
	}

	return &ObjectRef{
		Name:      name,
		Namespace: namespace,
	}
}
