package curl

import (
	"encoding/json"
	"path/filepath"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// Detector is detector adapter for CURL like tests
type Detector struct {
}

const (
	// Type is test type
	Type = "curl/test"
)

// Is detects based on upsert test options what kind of test it is
func (d Detector) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(options.Content.Data), &data)
	if err != nil {
		return
	}

	if info, ok := data["command"]; ok {
		if commands, ok := info.([]interface{}); ok {
			if app, ok := commands[0].(string); ok && app == "curl" {
				return d.GetType(), true
			}
		}
	}

	return
}

// IsWithPath detects based on upsert test options what kind of test it is
func (d Detector) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".json")
	return
}

// GetType returns test type
func (d Detector) GetType() string {
	return Type
}
