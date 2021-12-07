package detector

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

type Adapter interface {
	Is(options apiClient.UpsertScriptOptions) (string, bool)
}
