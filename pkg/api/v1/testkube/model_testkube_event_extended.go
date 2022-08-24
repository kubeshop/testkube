package testkube

func (e TestkubeEvent) Log() []any {

	var executionId, executionName, eventType string
	if e.Execution != nil {
		executionId = e.Execution.Id
		executionName = e.Execution.Name
	}

	if e.Type_ != nil {
		eventType = e.Type_.String()
	}

	return []any{
		"uri", e.Uri,
		"type", eventType,
		"executionId", executionId,
		"executionName", executionName,
	}
}
