package testkube

// ToTestSuiteUpsertRequest converts to TestSuiteUpsertRequest model
func (t *TestSuiteUpsertRequestV2) ToTestSuiteUpsertRequest() *TestSuiteUpsertRequest {
	var before, steps, after []TestSuiteBatchStep
	for _, step := range t.Before {
		before = append(before, *step.ToTestSuiteBatchStep())
	}

	for _, step := range t.Steps {
		steps = append(steps, *step.ToTestSuiteBatchStep())
	}

	for _, step := range t.After {
		after = append(after, *step.ToTestSuiteBatchStep())
	}

	return &TestSuiteUpsertRequest{
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

// ToTestSuiteBatchStep converts to ToTestSuiteBatchStep model
func (s *TestSuiteStepV2) ToTestSuiteBatchStep() *TestSuiteBatchStep {
	return &TestSuiteBatchStep{
		StopOnFailure: s.StopTestOnFailure,
		Batch: []TestSuiteStep{
			{
				Execute: s.Execute,
				Delay:   s.Delay,
			},
		},
	}
}
