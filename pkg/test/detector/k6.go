package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// K6Adapter is detector adapter for Postman collection saved as JSON content
type K6Adapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d K6Adapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if strings.Contains(options.Content.Data, "from 'k6") {
		return d.GetType(), true
	}

	return
}

// IsTestName detecs if filename has a conventional test name
func (d K6Adapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d K6Adapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d K6Adapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d K6Adapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d K6Adapter) GetType() string {
	return "k6/script"
}
