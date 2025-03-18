package testworkflowclient

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

var _ TestWorkflowClient = &cloudTestWorkflowClient{}

type cloudTestWorkflowClient struct {
	client controlplaneclient.TestWorkflowsClient
}

func NewCloudTestWorkflowClient(client controlplaneclient.TestWorkflowsClient) TestWorkflowClient {
	return &cloudTestWorkflowClient{client: client}
}

func (c *cloudTestWorkflowClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error) {
	return c.client.GetTestWorkflow(ctx, environmentId, name)
}

func (c *cloudTestWorkflowClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflow, error) {
	list, err := c.client.ListTestWorkflows(ctx, environmentId, controlplaneclient.ListTestWorkflowOptions{
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Offset:     options.Offset,
		Limit:      options.Limit,
	}).All()
	if err != nil {
		return nil, err
	}
	return common.MapSlice(list, func(t *testkube.TestWorkflow) testkube.TestWorkflow {
		return *t
	}), nil
}

func (c *cloudTestWorkflowClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	return c.client.ListTestWorkflowLabels(ctx, environmentId)
}

func (c *cloudTestWorkflowClient) Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	return c.client.UpdateTestWorkflow(ctx, environmentId, workflow)
}

func (c *cloudTestWorkflowClient) Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error {
	return c.client.CreateTestWorkflow(ctx, environmentId, workflow)
}

func (c *cloudTestWorkflowClient) Delete(ctx context.Context, environmentId string, name string) error {
	return c.client.DeleteTestWorkflow(ctx, environmentId, name)
}

func (c *cloudTestWorkflowClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	return c.client.DeleteTestWorkflowsByLabels(ctx, environmentId, labels)
}

func (c *cloudTestWorkflowClient) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error) {
	return "", nil
}

func (c *cloudTestWorkflowClient) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher {
	return channels.Transform(c.client.WatchTestWorkflowUpdates(ctx, environmentId, includeInitialData), func(t *controlplaneclient.TestWorkflowUpdate) (Update, bool) {
		switch t.Type {
		case cloud.UpdateType_UPDATE:
			return Update{Type: EventTypeUpdate, Timestamp: t.Timestamp, Resource: t.Resource}, true
		case cloud.UpdateType_DELETE:
			return Update{Type: EventTypeDelete, Timestamp: t.Timestamp, Resource: t.Resource}, true
		case cloud.UpdateType_CREATE:
			return Update{Type: EventTypeCreate, Timestamp: t.Timestamp, Resource: t.Resource}, true
		default:
			return Update{}, false
		}
	})
}
