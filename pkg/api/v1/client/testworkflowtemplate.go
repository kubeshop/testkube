package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestWorkflowTemplateClient creates new TestWorkflowTemplate client
func NewTestWorkflowTemplateClient(
	testWorkflowTemplateTransport Transport[testkube.TestWorkflowTemplate],
) TestWorkflowTemplateClient {
	return TestWorkflowTemplateClient{
		testWorkflowTemplateTransport: testWorkflowTemplateTransport,
	}
}

// TestWorkflowTemplateClient is a client for tests
type TestWorkflowTemplateClient struct {
	testWorkflowTemplateTransport Transport[testkube.TestWorkflowTemplate]
}

// GetTestWorkflowTemplate returns single test by id
func (c TestWorkflowTemplateClient) GetTestWorkflowTemplate(id string) (testkube.TestWorkflowTemplate, error) {
	id = strings.ReplaceAll(id, "/", "--")
	uri := c.testWorkflowTemplateTransport.GetURI("/test-workflow-templates/%s", id)
	return c.testWorkflowTemplateTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestWorkflowTemplates list all tests
func (c TestWorkflowTemplateClient) ListTestWorkflowTemplates(selector string) (testkube.TestWorkflowTemplates, error) {
	params := map[string]string{"selector": selector}
	uri := c.testWorkflowTemplateTransport.GetURI("/test-workflow-templates")
	return c.testWorkflowTemplateTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// DeleteTestWorkflowTemplates deletes multiple test workflow templates by labels
func (c TestWorkflowTemplateClient) DeleteTestWorkflowTemplates(selector string) error {
	uri := c.testWorkflowTemplateTransport.GetURI("/test-workflow-templates")
	return c.testWorkflowTemplateTransport.Delete(uri, selector, true)
}

// CreateTestWorkflowTemplate creates new TestWorkflowTemplate Custom Resource
func (c TestWorkflowTemplateClient) CreateTestWorkflowTemplate(template testkube.TestWorkflowTemplate) (result testkube.TestWorkflowTemplate, err error) {
	template.Name = strings.ReplaceAll(template.Name, "/", "--")

	body, err := json.Marshal(template)
	if err != nil {
		return result, err
	}

	uri := c.testWorkflowTemplateTransport.GetURI("/test-workflow-templates")
	return c.testWorkflowTemplateTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTestWorkflowTemplate updates TestWorkflowTemplate Custom Resource
func (c TestWorkflowTemplateClient) UpdateTestWorkflowTemplate(template testkube.TestWorkflowTemplate) (result testkube.TestWorkflowTemplate, err error) {
	if template.Name == "" {
		return result, fmt.Errorf("test workflow template name '%s' is not valid", template.Name)
	}
	template.Name = strings.ReplaceAll(template.Name, "/", "--")

	body, err := json.Marshal(template)
	if err != nil {
		return result, err
	}

	uri := c.testWorkflowTemplateTransport.GetURI("/test-workflow-templates/%s", template.Name)
	return c.testWorkflowTemplateTransport.Execute(http.MethodPut, uri, body, nil)
}

// DeleteTestWorkflowTemplate deletes single test by name
func (c TestWorkflowTemplateClient) DeleteTestWorkflowTemplate(name string) error {
	if name == "" {
		return fmt.Errorf("test workflow template name '%s' is not valid", name)
	}
	name = strings.ReplaceAll(name, "/", "--")

	uri := c.testWorkflowTemplateTransport.GetURI("/test-workflow-templates/%s", name)
	return c.testWorkflowTemplateTransport.Delete(uri, "", true)
}
