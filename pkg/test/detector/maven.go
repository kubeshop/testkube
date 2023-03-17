package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// has multiple files and a pom.xml file

// MavenAdapter is detector adapter for Maven test
type MavenAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d MavenAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if options.Content.Data == "" {
		return
	}

	if strings.Contains(options.Content.Data, "<project") {
		return d.GetType(), true
	}

	return
}

// IsTestName detecs if filename has a conventional test name
func (d MavenAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d MavenAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d MavenAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d MavenAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d MavenAdapter) GetType() string {
	return "maven/project"
}
