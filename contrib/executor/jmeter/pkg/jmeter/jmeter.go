package jmeter

import (
	"path/filepath"
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Can be one file with .jmx extension

// Detector is adapter for JMeter tests
type Detector struct{}

const (
	// Type is test type
	Type = "jmeter/test"
)

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "<jmeterTestPlan") {
		return d.GetType(), true
	}

	return
}

// IsWithPath detect test based on path and content
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".jmx")
	return
}

// GetType returns test type
func (d Detector) GetType() string {
	return Type
}
