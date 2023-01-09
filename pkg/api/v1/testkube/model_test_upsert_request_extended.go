package testkube

import (
	"fmt"
)

func (test *TestUpsertRequest) QuoteTestTextFields() {
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
			&test.ExecutionRequest.PreRunScript,
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
	}
}
