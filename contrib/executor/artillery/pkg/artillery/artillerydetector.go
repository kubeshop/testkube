package artillery

import (
	"path/filepath"
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Can be one file

// Detector is detector adapter for Artillery like tests
type Detector struct{}

const (
	// Type is test type
	Type = "artillery/test"
)

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "config:") && strings.Contains(options.Content.Data, "scenarios:") {
		return d.GetType(), true
	}

	return
}

// IsWithPath detect test based on path and content
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && ((ext == ".yml") || (ext == ".yaml"))
	return
}

// GetType returns test type
func (d Detector) GetType() string {
	return Type
}
