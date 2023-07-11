package testkube

// ToTestSuiteUpdateRequest converts to TestSuiteUpdateRequest model
func (t *TestSuiteUpdateRequestV2) ToTestSuiteUpdateRequest() *TestSuiteUpdateRequest {
	var before, steps, after *[]TestSuiteBatchStep

	if t.Before != nil {
		before = &[]TestSuiteBatchStep{}
		for _, step := range *t.Before {
			*before = append(*before, *step.ToTestSuiteBatchStep())
		}
	}

	if t.Steps != nil {
		steps = &[]TestSuiteBatchStep{}
		for _, step := range *t.Steps {
			*steps = append(*steps, *step.ToTestSuiteBatchStep())
		}
	}

	if t.After != nil {
		after = &[]TestSuiteBatchStep{}
		for _, step := range *t.After {
			*after = append(*after, *step.ToTestSuiteBatchStep())
		}
	}

	return &TestSuiteUpdateRequest{
		Name:             t.Name,
		Namespace:        t.Namespace,
		Description:      t.Description,
		Before:           before,
		Steps:            steps,
		After:            after,
		Labels:           t.Labels,
		Schedule:         t.Schedule,
		Repeats:          t.Repeats,
		Created:          t.Created,
		ExecutionRequest: t.ExecutionRequest,
		Status:           t.Status,
	}
}
