package detector

import (
	"encoding/json"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

type PostmanCollectionAdapter struct {
}

func (d PostmanCollectionAdapter) Is(options apiClient.CreateScriptOptions) (name string, ok bool) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(options.Content), &data)
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
