package detector

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Adapter defines methods for test detection
type Adapter interface {
	// Is detects based on upsert test options what kind of test it is
	Is(options apiClient.UpsertTestOptions) (string, bool)
	// IsWithPath detects based on path(extension) what kind of test it is
	IsWithPath(path string, options apiClient.UpsertTestOptions) (string, bool)
	// GetType returns test type
	GetType() string
}
