package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewConfigClient creates new Config client
func NewConfigClient(ConfigTransport Transport[testkube.Config]) ConfigClient {
	return ConfigClient{
		ConfigTransport: ConfigTransport,
	}
}

// ConfigClient is a client for Configs
type ConfigClient struct {
	ConfigTransport Transport[testkube.Config]
}

// GetConfig gets Config by name
func (c ConfigClient) UpdateKey(name string) (Config testkube.Config, err error) {
	uri := c.ConfigTransport.GetURI("/Configs/%s", name)
	return c.ConfigTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListConfigs list all Configs
func (c ConfigClient) ListConfigs(selector string) (Configs testkube.Configs, err error) {
	uri := c.ConfigTransport.GetURI("/Configs")
	params := map[string]string{
		"selector": selector,
	}

	return c.ConfigTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateConfig creates new Config Custom Resource
func (c ConfigClient) CreateConfig(options CreateConfigOptions) (Config testkube.Config, err error) {
	uri := c.ConfigTransport.GetURI("/Configs")
	request := testkube.ConfigCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return Config, err
	}

	return c.ConfigTransport.Execute(http.MethodPost, uri, body, nil)
}

// DeleteConfigs deletes all Configs
func (c ConfigClient) DeleteConfigs(selector string) (err error) {
	uri := c.ConfigTransport.GetURI("/Configs")
	return c.ConfigTransport.Delete(uri, selector, true)
}

// DeleteConfig deletes single Config by name
func (c ConfigClient) DeleteConfig(name string) (err error) {
	uri := c.ConfigTransport.GetURI("/Configs/%s", name)
	return c.ConfigTransport.Delete(uri, "", true)
}
