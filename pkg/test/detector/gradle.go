package detector

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// GradleAdapter is an adapter for gradle tests
type GradleAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d GradleAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d GradleAdapter) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	// TODO: implement for multiple files gradle tests
	return "", false
}

// IsTestName detecs if filename has a conventional test name
func (d GradleAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d GradleAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d GradleAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d GradleAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d GradleAdapter) GetType() string {
	return "gradle/project"
}
