package playwright

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

//multiple files and a package.json file

// Detector is detector adapter for Playwright test
type Detector struct{}

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
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

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	//TODO: implement support for multiple files tests
	return "", false
}

// GetType returns test type
func (d Detector) GetType() string {
	return "playwright/script"
}
