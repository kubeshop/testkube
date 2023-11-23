package testkube

import (
	"errors"
	"fmt"

	"github.com/adhocore/gronx"
)

func (test *TestUpsertRequest) QuoteTestTextFields() {
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

func ValidateUpsertTestRequest(test TestUpsertRequest) error {
	if test.Name == "" {
		return errors.New("test name cannot be empty")
	}
	if test.Type_ == "" {
		return errors.New("test type cannot be empty")
	}
	if test.Content == nil {
		return errors.New("test content cannot be empty")
	}
	if test.Schedule != "" {
		gron := gronx.New()
		if !gron.IsValid(test.Schedule) {
			return errors.New("invalin cron expression in test schedule")
		}
	}
	return nil
}
