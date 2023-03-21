package detector

import (
	"path/filepath"
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// single file .xml

// SoapUIAdapter is detector adapter for SoapUI test
type SoapUIAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d SoapUIAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
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
func (d SoapUIAdapter) IsWithPath(path string, options apiClient.UpsertTestOptions) (name string, ok bool) {
	name, ok = d.Is(options)
	ext := filepath.Ext(path)
	ok = ok && (ext == ".xml")
	return
}

// GetType returns test type
func (d SoapUIAdapter) GetType() string {
	return "soapui/xml"
}

// IsTestName detecs if filename has a conventional test name
func (d SoapUIAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d SoapUIAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d SoapUIAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d SoapUIAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}
