package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewWebhookTemplateClient creates new WebhookTemplate client
func NewWebhookTemplateClient(webhookTemplateTransport Transport[testkube.WebhookTemplate]) WebhookTemplateClient {
	return WebhookTemplateClient{
		webhookTemplateTransport: webhookTemplateTransport,
	}
}

// WebhookTemplateClient is a client for webhook templates
type WebhookTemplateClient struct {
	webhookTemplateTransport Transport[testkube.WebhookTemplate]
}

// GetWebhookTemplate gets webhook template by name
func (c WebhookTemplateClient) GetWebhookTemplate(name string) (webhookTemplate testkube.WebhookTemplate, err error) {
	uri := c.webhookTemplateTransport.GetURI("/webhook-templates/%s", name)
	return c.webhookTemplateTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListWebhookTemplates list all webhook templates
func (c WebhookTemplateClient) ListWebhookTemplates(selector string) (webhookTemplates testkube.WebhookTemplates, err error) {
	uri := c.webhookTemplateTransport.GetURI("/webhook-templates")
	params := map[string]string{
		"selector": selector,
	}

	return c.webhookTemplateTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateWebhookTemplate creates new WebhookTemplate Custom Resource
func (c WebhookTemplateClient) CreateWebhookTemplate(options CreateWebhookTemplateOptions) (webhookTemplate testkube.WebhookTemplate, err error) {
	uri := c.webhookTemplateTransport.GetURI("/webhook-templates")
	request := testkube.WebhookTemplateCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return webhookTemplate, err
	}

	return c.webhookTemplateTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateWebhookTemplate updates WebhookTemplate Custom Resource
func (c WebhookTemplateClient) UpdateWebhookTemplate(options UpdateWebhookTemplateOptions) (webhookTemplate testkube.WebhookTemplate, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.webhookTemplateTransport.GetURI("/webhook-templates/%s", name)
	request := testkube.WebhookTemplateUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return webhookTemplate, err
	}

	return c.webhookTemplateTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteWebhookTemplates deletes all webhook templates
func (c WebhookTemplateClient) DeleteWebhookTemplates(selector string) (err error) {
	uri := c.webhookTemplateTransport.GetURI("/webhook-templates")
	return c.webhookTemplateTransport.Delete(uri, selector, true)
}

// DeleteWebhookTemplate deletes single webhook template by name
func (c WebhookTemplateClient) DeleteWebhookTemplate(name string) (err error) {
	uri := c.webhookTemplateTransport.GetURI("/webhook-templates/%s", name)
	return c.webhookTemplateTransport.Delete(uri, "", true)
}
