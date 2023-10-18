package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTemplateClient creates new Template client
func NewTemplateClient(templateTransport Transport[testkube.Template]) TemplateClient {
	return TemplateClient{
		templateTransport: templateTransport,
	}
}

// TemplateClient is a client for templates
type TemplateClient struct {
	templateTransport Transport[testkube.Template]
}

// GetTemplate gets template by name
func (c TemplateClient) GetTemplate(name string) (template testkube.Template, err error) {
	uri := c.templateTransport.GetURI("/templates/%s", name)
	return c.templateTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTemplates list all templates
func (c TemplateClient) ListTemplates(selector string) (templates testkube.Templates, err error) {
	uri := c.templateTransport.GetURI("/templates")
	params := map[string]string{
		"selector": selector,
	}

	return c.templateTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateTemplate creates new Template Custom Resource
func (c TemplateClient) CreateTemplate(options CreateTemplateOptions) (template testkube.Template, err error) {
	uri := c.templateTransport.GetURI("/templates")
	request := testkube.TemplateCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return template, err
	}

	return c.templateTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTemplate updates Template Custom Resource
func (c TemplateClient) UpdateTemplate(options UpdateTemplateOptions) (template testkube.Template, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.templateTransport.GetURI("/templates/%s", name)
	request := testkube.TemplateUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return template, err
	}

	return c.templateTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteTemplates deletes all templates
func (c TemplateClient) DeleteTemplates(selector string) (err error) {
	uri := c.templateTransport.GetURI("/templates")
	return c.templateTransport.Delete(uri, selector, true)
}

// DeleteTemplate deletes single template by name
func (c TemplateClient) DeleteTemplate(name string) (err error) {
	uri := c.templateTransport.GetURI("/templates/%s", name)
	return c.templateTransport.Delete(uri, "", true)
}
