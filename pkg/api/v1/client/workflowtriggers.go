package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewWorkflowTriggerClient creates new WorkflowTrigger (v2) client.
func NewWorkflowTriggerClient(transport Transport[testkube.WorkflowTrigger]) WorkflowTriggerClient {
	return WorkflowTriggerClient{transport: transport}
}

// WorkflowTriggerClient talks to the REST endpoints for v2 WorkflowTriggers.
type WorkflowTriggerClient struct {
	transport Transport[testkube.WorkflowTrigger]
}

func (c WorkflowTriggerClient) GetWorkflowTrigger(name string) (testkube.WorkflowTrigger, error) {
	uri := c.transport.GetURI("/workflow-triggers/%s", name)
	return c.transport.Execute(http.MethodGet, uri, nil, nil)
}

func (c WorkflowTriggerClient) ListWorkflowTriggers(selector string) ([]testkube.WorkflowTrigger, error) {
	uri := c.transport.GetURI("/workflow-triggers")
	params := map[string]string{}
	if selector != "" {
		params["selector"] = selector
	}
	return c.transport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

func (c WorkflowTriggerClient) CreateWorkflowTrigger(trigger testkube.WorkflowTrigger) (testkube.WorkflowTrigger, error) {
	uri := c.transport.GetURI("/workflow-triggers")
	body, err := json.Marshal(trigger)
	if err != nil {
		return testkube.WorkflowTrigger{}, err
	}
	return c.transport.Execute(http.MethodPost, uri, body, nil)
}

func (c WorkflowTriggerClient) UpdateWorkflowTrigger(trigger testkube.WorkflowTrigger) (testkube.WorkflowTrigger, error) {
	uri := c.transport.GetURI("/workflow-triggers/%s", trigger.Name)
	body, err := json.Marshal(trigger)
	if err != nil {
		return testkube.WorkflowTrigger{}, err
	}
	return c.transport.Execute(http.MethodPatch, uri, body, nil)
}

func (c WorkflowTriggerClient) DeleteWorkflowTrigger(name string) error {
	uri := c.transport.GetURI("/workflow-triggers/%s", name)
	return c.transport.Delete(uri, "", true)
}

func (c WorkflowTriggerClient) DeleteWorkflowTriggers(selector string) error {
	uri := c.transport.GetURI("/workflow-triggers")
	return c.transport.Delete(uri, selector, true)
}
