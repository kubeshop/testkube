package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//multiple files and a package.json file

// PlaywrightAdapter is detector adapter for Playwright test
type PlaywrightAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d PlaywrightAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "@playwright/test") {
		return d.GetType(), true
	}

	return
}

// GetType returns test type
func (d PlaywrightAdapter) GetType() string {
	return "playwright/script"
}

// IsTestName detecs if filename has a conventional test name
func (d PlaywrightAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d PlaywrightAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d PlaywrightAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}
