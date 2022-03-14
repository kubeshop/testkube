package detector

import (
	"encoding/json"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

// PostmanCollectionAdapter is detector adapter for Postman collection saved as JSON content
type PostmanCollectionAdapter struct {
}

// Is detects based on upsert test options what kind of test it is
func (d PostmanCollectionAdapter) Is(options apiClient.UpsertTestOptions) (name string, ok bool) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(options.Content.Data), &data)
	if err != nil {
		return
	}

	if info, ok := data["info"]; ok {
		if id, ok := info.(map[string]interface{})["_postman_id"]; ok && id != "" {
			return "postman/collection", true
		}
	}

	return
}
