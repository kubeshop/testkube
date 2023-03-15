package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Multiple files usually .go and go.mod and go.sum

// GinkgoAdapter is detector adapter for Ginkgo test
type GinkgoAdapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d GinkgoAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
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

// IsTestName detecs if filename has a conventional test name
func (d GinkgoAdapter) IsTestName(filename string) (string, bool) {
	return "", false
}

// IsEnvName detecs if filename has a conventional env name
func (d GinkgoAdapter) IsEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// IsSecretEnvName detecs if filename has a conventional secret env name
func (d GinkgoAdapter) IsSecretEnvName(filename string) (string, string, bool) {
	return "", "", false
}

// GetSecretVariables retuns secret variables
func (d GinkgoAdapter) GetSecretVariables(data string) (map[string]testkube.Variable, error) {
	return nil, nil
}

// GetType returns test type
func (d GinkgoAdapter) GetType() string {
	return "ginkgo/test"
}
