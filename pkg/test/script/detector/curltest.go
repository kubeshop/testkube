package detector

import (
	"encoding/json"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

type CurlTestAdapter struct {
}

func (d CurlTestAdapter) Is(options apiClient.UpsertScriptOptions) (name string, ok bool) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(options.Content.Data), &data)
	if err != nil {
		return
	}

	if info, ok := data["command"]; ok {
		if commands, ok := info.([]interface{}); ok {
			if app, ok := commands[0].(string); ok && app == "curl" {
				return "curl/test", true
			}
		}
	}

	return
}
