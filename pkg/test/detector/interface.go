package detector

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Adapter defines methods for test detection
type Adapter interface {
	// Is detects based on upsert test options what kind of test it is
	Is(options apiClient.UpsertTestOptions) (string, bool)
	// IsTestName detecs if filename has a conventional test name
	IsTestName(filename string) (string, bool)
	// IsEnvName detecs if filename has a conventional env name
	IsEnvName(filename string) (string, string, bool)
	// IsSecretEnvName detecs if filename has a conventional secret env name
	IsSecretEnvName(filename string) (string, string, bool)
	// GetSecretVariables retuns secret variables
	GetSecretVariables(data string) (map[string]testkube.Variable, error)
	// GetType returns test type
	GetType() string
}
