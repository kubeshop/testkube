package detector

import (
	"encoding/json"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// CurlTestAdapter is detector adapter for CURL like tests
type CurlTestAdapter struct {
}

// Is detects based on upsert test options what kind of test it is
func (d CurlTestAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(options.Content.Data), &data)
	if err != nil {
		return
	}

	if info, ok := data["command"]; ok {
		if commands, ok := info.([]interface{}); ok {
			if app, ok := commands[0].(string); ok && app == "curl" {
				return d.GetType(), true
			}
		}
	}

	return
}

// IsTestName detecs if filename has a conventional test name
func (d CurlTestAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d CurlTestAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d CurlTestAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d CurlTestAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d CurlTestAdapter) GetType() string {
	return "curl/test"
}
