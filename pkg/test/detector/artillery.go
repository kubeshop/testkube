package detector

import (
	"path/filepath"
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Can be one file

// ArtilleryAdapter is detector adapter for Artillery like tests
type ArtilleryAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d ArtilleryAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
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
func (d ArtilleryAdapter) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".yml")
	return
}

// IsTestName detecs if filename has a conventional test name
func (d ArtilleryAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d ArtilleryAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d ArtilleryAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d ArtilleryAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d ArtilleryAdapter) GetType() string {
	return "artillery/test"
}
