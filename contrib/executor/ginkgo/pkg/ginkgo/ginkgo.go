package ginkgo

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Multiple files usually .go and go.mod and go.sum

// Detector is detector adapter for Ginkgo test
type Detector struct{}

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "github.com/onsi/ginkgo/") {
		return d.GetType(), true
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	// TODO: implement for multiple files test
	return "", false
}

// GetType returns test type
func (d Detector) GetType() string {
	return "ginkgo/test"
}
