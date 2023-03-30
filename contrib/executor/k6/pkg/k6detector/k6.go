package k6detector

import (
	"path/filepath"
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Detector is detector adapter for K6 test
type Detector struct{}

const (
	// Type is test type
	Type = "k6/script"
)

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if strings.Contains(options.Content.Data, "from 'k6") {
		return d.GetType(), true
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".js")
	return
}

// GetType returns test type
func (d Detector) GetType() string {
	return Type
}
