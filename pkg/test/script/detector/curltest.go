package detector

import (
	"encoding/json"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

type CurlTestAdapter struct {
}

func (d CurlTestAdapter) Is(options apiClient.CreateScriptOptions) (ok bool, name string) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(options.Content), &data)
	if err != nil {
		return
	}

	if info, ok := data["command"]; ok {
		if commands, ok := info.([]interface{}); ok {
			if app, ok := commands[0].(string); ok && app == "curl" {
				return true, "curl/test"
			}
		}
	}

	return
}
