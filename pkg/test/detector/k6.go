package detector

import (
	"strings"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// K6Adapter is detector adapter for Postman collection saved as JSON content
type K6Adapter struct{}

// Is detects based on upsert test options what kind of test it is
func (d K6Adapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	if options.Content == nil {
		return
	}

	if strings.Contains(options.Content.Data, "from 'k6") {
		return "k6/script", true
	}

	return
}
