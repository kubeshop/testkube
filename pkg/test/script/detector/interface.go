package detector

import (
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

type Adapter interface {
	Is(apiClient.CreateScriptOptions) (bool, string)
}
