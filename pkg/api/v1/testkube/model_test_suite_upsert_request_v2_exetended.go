package testkube

import "time"

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
	var test string
	if s.Execute != nil {
		test = s.Execute.Name
	}

	var delay string
	if s.Delay != nil && s.Delay.Duration != 0 {
		delay = time.Duration(s.Delay.Duration * int32(time.Millisecond)).String()
	}

	return &TestSuiteBatchStep{
		StopOnFailure: s.StopTestOnFailure,
		Execute: []TestSuiteStep{
			{
				Test:  test,
				Delay: delay,
			},
		},
	}
}
