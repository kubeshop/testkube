package testkube

import (
	"fmt"
)

func (testSuite *TestSuiteUpsertRequest) QuoteTestSuiteTextFields() {
	if testSuite.Description != "" {
		testSuite.Description = fmt.Sprintf("%q", testSuite.Description)
	}

	if testSuite.Schedule != "" {
		testSuite.Schedule = fmt.Sprintf("%q", testSuite.Schedule)
	}

	if testSuite.ExecutionRequest != nil {
		for key, value := range testSuite.ExecutionRequest.Variables {
			if value.Value != "" {
				value.Value = fmt.Sprintf("%q", value.Value)
				testSuite.ExecutionRequest.Variables[key] = value
			}
		}

		var fields = []*string{
			&testSuite.ExecutionRequest.JobTemplate,
			&testSuite.ExecutionRequest.CronJobTemplate,
			&testSuite.ExecutionRequest.ScraperTemplate,
			&testSuite.ExecutionRequest.PvcTemplate,
		}

		for _, field := range fields {
			if *field != "" {
				*field = fmt.Sprintf("%q", *field)
			}
		}
	}
}
