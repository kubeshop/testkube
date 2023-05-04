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

		if testSuite.ExecutionRequest.CronJobTemplate != "" {
			testSuite.ExecutionRequest.CronJobTemplate = fmt.Sprintf("%q", testSuite.ExecutionRequest.CronJobTemplate)
		}
	}
}
