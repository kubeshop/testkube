package detector

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

type Adapter interface {
	// Is detects based on upsert test options what kind of test it is
	Is(options apiClient.UpsertTestOptions) (string, bool)
}
