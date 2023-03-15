package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Can be one file with .jmx extension

// JMeterAdapter is adapter for JMeter tests
type JMeterAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d JMeterAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "<jmeterTestPlan") {
		return d.GetType(), true
	}

	return
}

// IsTestName detecs if filename has a conventional test name
func (d JMeterAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d JMeterAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d JMeterAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d JMeterAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d JMeterAdapter) GetType() string {
	return "jmeter/test"
}
