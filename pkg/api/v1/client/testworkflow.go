package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestWorkflowClient creates new TestWorkflow client
func NewTestWorkflowClient(
	testWorkflowTransport Transport[testkube.TestWorkflow],
	testWorkflowWithExecutionTransport Transport[testkube.TestWorkflowWithExecution],
	testWorkflowExecutionTransport Transport[testkube.TestWorkflowExecution],
	testWorkflowExecutionsResultTransport Transport[testkube.TestWorkflowExecutionsResult],
	artifactTransport Transport[testkube.Artifact],
) TestWorkflowClient {
	return TestWorkflowClient{
		testWorkflowTransport:                 testWorkflowTransport,
		testWorkflowWithExecutionTransport:    testWorkflowWithExecutionTransport,
		testWorkflowExecutionTransport:        testWorkflowExecutionTransport,
		testWorkflowExecutionsResultTransport: testWorkflowExecutionsResultTransport,
		artifactTransport:                     artifactTransport,
	}
}

// TestWorkflowClient is a client for test workflows
type TestWorkflowClient struct {
	testWorkflowTransport                 Transport[testkube.TestWorkflow]
	testWorkflowWithExecutionTransport    Transport[testkube.TestWorkflowWithExecution]
	testWorkflowExecutionTransport        Transport[testkube.TestWorkflowExecution]
	testWorkflowExecutionsResultTransport Transport[testkube.TestWorkflowExecutionsResult]
	artifactTransport                     Transport[testkube.Artifact]
}

// GetTestWorkflow returns single test workflow by id
func (c TestWorkflowClient) GetTestWorkflow(id string) (testkube.TestWorkflow, error) {
	uri := c.testWorkflowTransport.GetURI("/test-workflows/%s", id)
	return c.testWorkflowTransport.Execute(http.MethodGet, uri, nil, nil)
}

// GetTestWorkflowWithExecution returns single test workflow with execution by id
func (c TestWorkflowClient) GetTestWorkflowWithExecution(id string) (testkube.TestWorkflowWithExecution, error) {
	uri := c.testWorkflowWithExecutionTransport.GetURI("/test-workflow-with-executions/%s", id)
	return c.testWorkflowWithExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestWorkflows list all test workflows
func (c TestWorkflowClient) ListTestWorkflows(selector string) (testkube.TestWorkflows, error) {
	uri := c.testWorkflowTransport.GetURI("/test-workflows")
	params := map[string]string{"selector": selector}
	return c.testWorkflowTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// ListTestWorkflowWithExecutions list all test workflows with their latest executions
func (c TestWorkflowClient) ListTestWorkflowWithExecutions(selector string) (testkube.TestWorkflowWithExecutions, error) {
	uri := c.testWorkflowWithExecutionTransport.GetURI("/test-workflow-with-executions")
	params := map[string]string{"selector": selector}
	return c.testWorkflowWithExecutionTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
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

// GetTestWorkflowExecution returns single test workflow execution by id
func (c TestWorkflowClient) GetTestWorkflowExecution(id string) (testkube.TestWorkflowExecution, error) {
	uri := c.testWorkflowExecutionTransport.GetURI("/test-workflow-executions/%s", id)
	return c.testWorkflowExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestWorkflowExecutions list test workflow executions for selected workflow
func (c TestWorkflowClient) ListTestWorkflowExecutions(id string, limit int, selector string) (testkube.TestWorkflowExecutionsResult, error) {
	uri := c.testWorkflowExecutionsResultTransport.GetURI("/test-workflow-executions/")
	if id != "" {
		uri = c.testWorkflowExecutionsResultTransport.GetURI(fmt.Sprintf("/test-workflows/%s/executions", id))
	}
	params := map[string]string{
		"selector": selector,
		"pageSize": fmt.Sprintf("%d", limit),
	}
	return c.testWorkflowExecutionsResultTransport.Execute(http.MethodGet, uri, nil, params)
}

// AbortTestWorkflowExecution aborts selected execution
func (c TestWorkflowClient) AbortTestWorkflowExecution(workflow, id string) error {
	uri := c.testWorkflowTransport.GetURI("/test-workflows/%s/executions/%s/abort", workflow, id)
	return c.testWorkflowTransport.ExecuteMethod(http.MethodPost, uri, "", false)
}

// AbortTestWorkflowExecutions aborts all workflow executions
func (c TestWorkflowClient) AbortTestWorkflowExecutions(workflow string) error {
	uri := c.testWorkflowTransport.GetURI("/test-workflows/%s/abort", workflow)
	return c.testWorkflowTransport.ExecuteMethod(http.MethodPost, uri, "", false)
}

// GetTestWorkflowExecutionArtifacts returns execution artifacts
func (c TestWorkflowClient) GetTestWorkflowExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.artifactTransport.GetURI("/test-workflow-executions/%s/artifacts", executionID)
	return c.artifactTransport.ExecuteMultiple(http.MethodGet, uri, nil, nil)
}

// DownloadTestWorkflowArtifact downloads file
func (c TestWorkflowClient) DownloadTestWorkflowArtifact(executionID, fileName, destination string) (artifact string, err error) {
	uri := c.testWorkflowExecutionTransport.GetURI("/test-workflow-executions/%s/artifacts/%s", executionID, url.QueryEscape(fileName))
	return c.testWorkflowExecutionTransport.GetFile(uri, fileName, destination, nil)
}

// DownloadTestWorkflowArtifactArchive downloads archive
func (c TestWorkflowClient) DownloadTestWorkflowArtifactArchive(executionID, destination string, masks []string) (archive string, err error) {
	uri := c.testWorkflowExecutionTransport.GetURI("/test-workflow-executions/%s/artifact-archive", executionID)
	return c.testWorkflowExecutionTransport.GetFile(uri, fmt.Sprintf("%s.tar.gz", executionID), destination, map[string][]string{"mask": masks})
}
