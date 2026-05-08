package testkube

func (t TestWorkflowSpec) GetRequiredParameters() []string {
	keys := make([]string, 0)

	for key, value := range t.Config {
		if value.Default_ == nil {
			keys = append(keys, key)
		}
	}

	return keys
}
