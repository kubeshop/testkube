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

// ListTemplates list all templates

// CreateTemplate creates new Template Custom Resource

// UpdateTemplate updates Template Custom Resource

// DeleteTemplates deletes all templates

// DeleteTemplate deletes single template by name
