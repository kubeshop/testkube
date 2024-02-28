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

		var fields = []*string{
			&t.ExecutionRequest.JobTemplate,
			&t.ExecutionRequest.CronJobTemplate,
			&t.ExecutionRequest.PvcTemplate,
			&t.ExecutionRequest.ScraperTemplate,
		}

		for _, field := range fields {
			if *field != "" {
				*field = fmt.Sprintf("%q", *field)
			}
		}
	}
	for i := range t.Before {
		for j := range t.Before[i].Execute {
			if t.Before[i].Execute[j].ExecutionRequest != nil {
				t.Before[i].Execute[j].ExecutionRequest.QuoteTestSuiteStepExecutionRequestTextFields()
			}
		}
	}
	for i := range t.After {
		for j := range t.After[i].Execute {
			if t.After[i].Execute[j].ExecutionRequest != nil {
				t.After[i].Execute[j].ExecutionRequest.QuoteTestSuiteStepExecutionRequestTextFields()
			}
		}
	}
	for i := range t.Steps {
		for j := range t.Steps[i].Execute {
			if t.Steps[i].Execute[j].ExecutionRequest != nil {
				t.Steps[i].Execute[j].ExecutionRequest.QuoteTestSuiteStepExecutionRequestTextFields()
			}
		}
	}
}

func (request *TestSuiteStepExecutionRequest) QuoteTestSuiteStepExecutionRequestTextFields() {
	for i := range request.Args {
		if request.Args[i] != "" {
			request.Args[i] = fmt.Sprintf("%q", request.Args[i])
		}
	}

	for i := range request.Command {
		if request.Command[i] != "" {
			request.Command[i] = fmt.Sprintf("%q", request.Command[i])
		}
	}

	var fields = []*string{
		&request.JobTemplate,
		&request.CronJobTemplate,
		&request.ScraperTemplate,
		&request.PvcTemplate,
	}

	for _, field := range fields {
		if *field != "" {
			*field = fmt.Sprintf("%q", *field)
		}
	}
}
