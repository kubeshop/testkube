package cypress

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Has multiple files

// Detector is detector adapter for Cypress like tests
type Detector struct{}

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
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

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	//TODO: implement support for multiple files tests
	return "", false
}

// GetType returns test type
func (d Detector) GetType() string {
	return "cypress/project"
}
