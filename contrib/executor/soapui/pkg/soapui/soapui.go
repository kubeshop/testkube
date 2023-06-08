package soapui

import (
	"path/filepath"
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// single file .xml

// Detector is detector adapter for SoapUI test
type Detector struct{}

const (
	// Type is test type
	Type = "soapui/xml"
)

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "<con:soapui-project") {
		return d.GetType(), true
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".xml")
	return
}

// GetType returns test type
func (d Detector) GetType() string {
	return Type
}
