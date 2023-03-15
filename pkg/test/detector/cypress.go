package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Has multiple files

// CypressAdapter is detector adapter for Cypress like tests
type CypressAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d CypressAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "require('cypress')") {
		return d.GetType(), true
	}

	return
}

// IsTestName detecs if filename has a conventional test name
func (d CypressAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d CypressAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d CypressAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d CypressAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d CypressAdapter) GetType() string {
	return "cypress/project"
}
