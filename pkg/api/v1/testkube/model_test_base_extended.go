package testkube

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
)

const (
	//TestLabelTestType is a test label for a test type
	TestLabelTestType = "test-type"
	// TestLabelExecutor is a test label for an executor
	TestLabelExecutor = "executor"
	// TestLabelTestName is a test label for a test name
	TestLabelTestName = "test-name"
)

type Tests []Test

func (t Tests) Table() (header []string, output [][]string) {
	header = []string{"Name", "Description", "Type", "Created", "Labels", "Schedule"}
	for _, e := range t {
		output = append(output, []string{
			e.Name,
			e.Description,
			e.Type_,
			e.Created.String(),
			MapToString(e.Labels),
			e.Schedule,
		})
	}

	return
}

func (t Test) GetObjectRef(namespace string) *ObjectRef {
	return &ObjectRef{
		Name:      t.Name,
		Namespace: namespace,
	}
}

func PrepareExecutorArgs(binaryArgs []string) ([]string, error) {
	executorArgs := make([]string, 0)
	for _, arg := range binaryArgs {
		r := csv.NewReader(strings.NewReader(arg))
		r.Comma = ' '
		r.LazyQuotes = true
		r.TrimLeadingSpace = true

		records, err := r.ReadAll()
		if err != nil {
			return nil, err
		}

		if len(records) != 1 {
			return nil, errors.New("single string expected")
		}

		executorArgs = append(executorArgs, records[0]...)
	}

	return executorArgs, nil
}

func (test *Test) QuoteTestTextFields() {
	if test.Description != "" {
		test.Description = fmt.Sprintf("%q", test.Description)
	}

	if test.Content != nil && test.Content.Data != "" {
		test.Content.Data = fmt.Sprintf("%q", test.Content.Data)
	}

	if test.Schedule != "" {
		test.Schedule = fmt.Sprintf("%q", test.Schedule)
	}

	if test.ExecutionRequest != nil {
		var fields = []*string{
			&test.ExecutionRequest.VariablesFile,
			&test.ExecutionRequest.JobTemplate,
			&test.ExecutionRequest.CronJobTemplate,
			&test.ExecutionRequest.PreRunScript,
			&test.ExecutionRequest.PostRunScript,
			&test.ExecutionRequest.PvcTemplate,
			&test.ExecutionRequest.ScraperTemplate,
		}

		for _, field := range fields {
			if *field != "" {
				*field = fmt.Sprintf("%q", *field)
			}
		}

		for key, value := range test.ExecutionRequest.Envs {
			if value != "" {
				test.ExecutionRequest.Envs[key] = fmt.Sprintf("%q", value)
			}
		}

		for key, value := range test.ExecutionRequest.SecretEnvs {
			if value != "" {
				test.ExecutionRequest.SecretEnvs[key] = fmt.Sprintf("%q", value)
			}
		}

		for key, value := range test.ExecutionRequest.Variables {
			if value.Value != "" {
				value.Value = fmt.Sprintf("%q", value.Value)
				test.ExecutionRequest.Variables[key] = value
			}
		}

		for i := range test.ExecutionRequest.Args {
			if test.ExecutionRequest.Args[i] != "" {
				test.ExecutionRequest.Args[i] = fmt.Sprintf("%q", test.ExecutionRequest.Args[i])
			}
		}

		for i := range test.ExecutionRequest.Command {
			if test.ExecutionRequest.Command[i] != "" {
				test.ExecutionRequest.Command[i] = fmt.Sprintf("%q", test.ExecutionRequest.Command[i])
			}
		}

		if test.ExecutionRequest.SlavePodRequest != nil && test.ExecutionRequest.SlavePodRequest.PodTemplate != "" {
			test.ExecutionRequest.SlavePodRequest.PodTemplate = fmt.Sprintf("%q", test.ExecutionRequest.SlavePodRequest.PodTemplate)
		}
	}
}
