package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewWebhookClient creates new Webhook client
func NewWebhookClient(webhookTransport Transport[testkube.Webhook]) WebhookClient {
	return WebhookClient{
		webhookTransport: webhookTransport,
	}
}

// WebhookClient is a client for webhooks
type WebhookClient struct {
	webhookTransport Transport[testkube.Webhook]
}

// GetWebhook gets webhook by name
func (c WebhookClient) GetWebhook(name string) (webhook testkube.Webhook, err error) {
	uri := c.webhookTransport.GetURI("/webhooks/%s", name)
	return c.webhookTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListWebhooks list all webhooks
func (c WebhookClient) ListWebhooks(selector string) (webhooks testkube.Webhooks, err error) {
	uri := c.webhookTransport.GetURI("/webhooks")
	params := map[string]string{
		"selector": selector,
	}

	return c.webhookTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateWebhook creates new Webhook Custom Resource
func (c WebhookClient) CreateWebhook(options CreateWebhookOptions) (webhook testkube.Webhook, err error) {
	uri := c.webhookTransport.GetURI("/webhooks")
	request := testkube.WebhookCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return webhook, err
	}

	return c.webhookTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateWebhook updates Webhook Custom Resource
func (c WebhookClient) UpdateWebhook(options UpdateWebhookOptions) (webhook testkube.Webhook, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.webhookTransport.GetURI("/webhooks/%s", name)
	request := testkube.WebhookUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return webhook, err
	}

	return c.webhookTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteWebhooks deletes all webhooks
func (c WebhookClient) DeleteWebhooks(selector string) (err error) {
	uri := c.webhookTransport.GetURI("/webhooks")
	return c.webhookTransport.Delete(uri, selector, true)
}

// DeleteWebhook deletes single webhook by name
func (c WebhookClient) DeleteWebhook(name string) (err error) {
	uri := c.webhookTransport.GetURI("/webhooks/%s", name)
	return c.webhookTransport.Delete(uri, "", true)
}
