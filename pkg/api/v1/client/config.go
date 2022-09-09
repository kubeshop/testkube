package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewConfigClient creates new Cnfig client
func NewConfigClient(configTransport Transport[testkube.Config]) ConfigClient {
	return ConfigClient{
		configTransport: configTransport,
	}
}

// ConfigClient is a client for config
type ConfigClient struct {
	configTransport Transport[testkube.Config]
}

func (c ConfigClient) UpdateConfig(config testkube.Config) (outputConfig testkube.Config, err error) {
	uri := c.configTransport.GetURI("/config")

	body, err := json.Marshal(config)
	if err != nil {
		return outputConfig, err
	}

	return c.configTransport.Execute(http.MethodPatch, uri, body, nil)
}

func (c ConfigClient) GetConfig() (config testkube.Config, err error) {
	uri := c.configTransport.GetURI("/config")
	return c.configTransport.Execute(http.MethodGet, uri, nil, nil)
}
