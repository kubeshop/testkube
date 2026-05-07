package workflowtriggerclient

import (
	"context"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

var _ WorkflowTriggerClient = &cloudWorkflowTriggerClient{}

type cloudWorkflowTriggerClient struct {
	client controlplaneclient.WorkflowTriggersClient
}

// NewCloudWorkflowTriggerClient wraps a control-plane gRPC client so the agent
// can fetch/manage WorkflowTriggers from cloud in connected mode. Mirrors the
// TestTrigger cloud client for parity.
func NewCloudWorkflowTriggerClient(client controlplaneclient.WorkflowTriggersClient) WorkflowTriggerClient {
	return &cloudWorkflowTriggerClient{client: client}
}

func (c *cloudWorkflowTriggerClient) Get(ctx context.Context, environmentId, name, namespace string) (*testkube.WorkflowTrigger, error) {
	return c.client.GetWorkflowTrigger(ctx, environmentId, name, namespace)
}

func (c *cloudWorkflowTriggerClient) List(ctx context.Context, environmentId string, options ListOptions, namespace string) ([]testkube.WorkflowTrigger, error) {
	list, err := c.client.ListWorkflowTriggers(ctx, environmentId, controlplaneclient.ListWorkflowTriggerOptions{
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Selector:   options.Selector,
		Offset:     options.Offset,
		Limit:      options.Limit,
	}, namespace).All()
	if err != nil {
		return nil, err
	}
	return common.MapSlice(list, func(t *testkube.WorkflowTrigger) testkube.WorkflowTrigger {
		return *t
	}), nil
}

func (c *cloudWorkflowTriggerClient) Update(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error {
	return c.client.UpdateWorkflowTrigger(ctx, environmentId, trigger)
}

func (c *cloudWorkflowTriggerClient) Create(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error {
	return c.client.CreateWorkflowTrigger(ctx, environmentId, trigger)
}

func (c *cloudWorkflowTriggerClient) Delete(ctx context.Context, environmentId, name, namespace string) error {
	return c.client.DeleteWorkflowTrigger(ctx, environmentId, name, namespace)
}

func (c *cloudWorkflowTriggerClient) DeleteAll(ctx context.Context, environmentId, namespace string) (uint32, error) {
	return c.client.DeleteAllWorkflowTriggers(ctx, environmentId, namespace)
}

func (c *cloudWorkflowTriggerClient) DeleteByLabels(ctx context.Context, environmentId, selector, namespace string) (uint32, error) {
	return c.client.DeleteWorkflowTriggersByLabels(ctx, environmentId, selector, namespace)
}
