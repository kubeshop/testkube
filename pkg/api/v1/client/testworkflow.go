package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestWorkflowClient creates new TestWorkflow client
func NewTestWorkflowClient(
	testWorkflowTransport Transport[testkube.TestWorkflow],
	testWorkflowExecutionTransport Transport[testkube.TestWorkflowExecution],
) TestWorkflowClient {
	return TestWorkflowClient{
		testWorkflowTransport:          testWorkflowTransport,
		testWorkflowExecutionTransport: testWorkflowExecutionTransport,
	}
}

// TestWorkflowClient is a client for tests
type TestWorkflowClient struct {
	testWorkflowTransport          Transport[testkube.TestWorkflow]
	testWorkflowExecutionTransport Transport[testkube.TestWorkflowExecution]
}

// GetTestWorkflow returns single test by id
func (c TestWorkflowClient) GetTestWorkflow(id string) (testkube.TestWorkflow, error) {
	uri := c.testWorkflowTransport.GetURI("/test-workflows/%s", id)
	return c.testWorkflowTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestWorkflows list all tests
func (c TestWorkflowClient) ListTestWorkflows(selector string) (testkube.TestWorkflows, error) {
	uri := c.testWorkflowTransport.GetURI("/test-workflows")
	params := map[string]string{"selector": selector}
	return c.testWorkflowTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// DeleteTestWorkflows deletes multiple test workflows by labels
func (c TestWorkflowClient) DeleteTestWorkflows(selector string) error {
	uri := c.testWorkflowTransport.GetURI("/test-workflows")
	return c.testWorkflowTransport.Delete(uri, selector, true)
}

// CreateTestWorkflow creates new TestWorkflow Custom Resource
func (c TestWorkflowClient) CreateTestWorkflow(workflow testkube.TestWorkflow) (result testkube.TestWorkflow, err error) {
	uri := c.testWorkflowTransport.GetURI("/test-workflows")

	body, err := json.Marshal(workflow)
	if err != nil {
		return result, err
	}

	return c.testWorkflowTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTestWorkflow updates TestWorkflow Custom Resource
func (c TestWorkflowClient) UpdateTestWorkflow(workflow testkube.TestWorkflow) (result testkube.TestWorkflow, err error) {
	if workflow.Name == "" {
		return result, fmt.Errorf("test workflow name '%s' is not valid", workflow.Name)
	}

	uri := c.testWorkflowTransport.GetURI("/test-workflows/%s", workflow.Name)

	body, err := json.Marshal(workflow)
	if err != nil {
		return result, err
	}

	return c.testWorkflowTransport.Execute(http.MethodPut, uri, body, nil)
}

// DeleteTestWorkflow deletes single test by name
func (c TestWorkflowClient) DeleteTestWorkflow(name string) error {
	if name == "" {
		return fmt.Errorf("test workflow name '%s' is not valid", name)
	}

	uri := c.testWorkflowTransport.GetURI("/test-workflows/%s", name)
	return c.testWorkflowTransport.Delete(uri, "", true)
}

// ExecuteTestWorkflow starts new TestWorkflow execution
func (c TestWorkflowClient) ExecuteTestWorkflow(name string, request testkube.TestWorkflowExecutionRequest) (result testkube.TestWorkflowExecution, err error) {
	if name == "" {
		return result, fmt.Errorf("test workflow name '%s' is not valid", name)
	}

	uri := c.testWorkflowExecutionTransport.GetURI("/test-workflows/%s/executions", name)

	body, err := json.Marshal(request)
	if err != nil {
		return result, err
	}

	return c.testWorkflowExecutionTransport.Execute(http.MethodPost, uri, body, nil)
}

// GetTestWorkflowExecutionNotifications returns events stream from job pods, based on job pods logs
func (c TestWorkflowClient) GetTestWorkflowExecutionNotifications(id string) (notifications chan testkube.TestWorkflowExecutionNotification, err error) {
	notifications = make(chan testkube.TestWorkflowExecutionNotification)
	uri := c.testWorkflowTransport.GetURI("/test-workflow-executions/%s/notifications", id)
	err = c.testWorkflowTransport.GetTestWorkflowExecutionNotifications(uri, notifications)
	return notifications, err
}
