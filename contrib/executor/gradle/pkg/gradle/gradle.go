package gradle

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Detector is an adapter for gradle tests
type Detector struct{}

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	// TODO: implement for multiple files gradle tests
	return "", false
}

// GetType returns test type
func (d Detector) GetType() string {
	return "gradle/project"
}
