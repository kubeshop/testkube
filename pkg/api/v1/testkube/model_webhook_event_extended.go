package testkube

func (e WebhookEvent) Log() []any {

	var executionId, executionName string
	if e.Execution != nil {
		executionId = e.Execution.Id
		executionName = e.Execution.Name
	}

	return []any{
		"uri", e.Uri,
		"type", e.Type_.String(),
		"executionId", executionId,
		"executionName", executionName,
	}
}
