package kubepug

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Detector is a test adapter for kubepug
type Detector struct{}

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	// TODO: implement kubepug detector

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	// TODO: implement kubepug detector
	return "", false
}

// GetType returns test type
func (d Detector) GetType() string {
	return "kubepug/yaml"
}
