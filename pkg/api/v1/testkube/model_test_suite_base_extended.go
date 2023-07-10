package testkube

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/data/set"
)

type TestSuites []TestSuite

func (tests TestSuites) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Steps", "Labels", "Schedule"}
	for _, e := range tests {
		output = append(output, []string{
			e.Name,
			e.Description,
			fmt.Sprintf("%d", len(e.Steps)),
			MapToString(e.Labels),
			e.Schedule,
		})
	}

	return
}

func (t TestSuite) GetObjectRef() *ObjectRef {
	return &ObjectRef{
		Name:      t.Name,
		Namespace: t.Namespace,
	}
}

// GetTestNames return test names for TestSuite
func (t TestSuite) GetTestNames() []string {
	var names []string
	var batches []TestSuiteBatchStep

	batches = append(batches, t.Before...)
	batches = append(batches, t.Steps...)
	batches = append(batches, t.After...)
	for _, batch := range batches {
		for _, step := range batch.Execute {
			if step.Test != "" {
				names = append(names, step.Test)
			}
		}
	}

	return set.Of(names...).ToArray()
}

func (t *TestSuite) QuoteTestSuiteTextFields() {
	if t.Description != "" {
		t.Description = fmt.Sprintf("%q", t.Description)
	}

	if t.Schedule != "" {
		t.Schedule = fmt.Sprintf("%q", t.Schedule)
	}

	if t.ExecutionRequest != nil {
		for key, value := range t.ExecutionRequest.Variables {
			if value.Value != "" {
				value.Value = fmt.Sprintf("%q", value.Value)
				t.ExecutionRequest.Variables[key] = value
			}
		}

		if t.ExecutionRequest.CronJobTemplate != "" {
			t.ExecutionRequest.CronJobTemplate = fmt.Sprintf("%q", t.ExecutionRequest.CronJobTemplate)
		}
	}
}
