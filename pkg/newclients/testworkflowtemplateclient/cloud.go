package testworkflowtemplateclient

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

var _ TestWorkflowTemplateClient = &cloudTestWorkflowTemplateClient{}

type cloudTestWorkflowTemplateClient struct {
	client controlplaneclient.TestWorkflowTemplatesClient
}

func NewCloudTestWorkflowTemplateClient(client controlplaneclient.TestWorkflowTemplatesClient, disableOfficialTemplates bool) TestWorkflowTemplateClient {
	if disableOfficialTemplates {
		return &cloudTestWorkflowTemplateClient{client: client}
	}
	return NewTestWorkflowTemplateClientWithOfficials(&cloudTestWorkflowTemplateClient{client: client})
}

func (c *cloudTestWorkflowTemplateClient) Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflowTemplate, error) {
	return c.client.GetTestWorkflowTemplate(ctx, environmentId, name)
}

func (c *cloudTestWorkflowTemplateClient) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error) {
	return "", nil
}

func (c *cloudTestWorkflowTemplateClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflowTemplate, error) {
	list, err := c.client.ListTestWorkflowTemplates(ctx, environmentId, controlplaneclient.ListTestWorkflowTemplateOptions{
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Offset:     options.Offset,
		Limit:      options.Limit,
	}).All()
	if err != nil {
		return nil, err
	}
	return common.MapSlice(list, func(t *testkube.TestWorkflowTemplate) testkube.TestWorkflowTemplate {
		return *t
	}), nil
}

func (c *cloudTestWorkflowTemplateClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	return c.client.ListTestWorkflowTemplateLabels(ctx, environmentId)
}

func (c *cloudTestWorkflowTemplateClient) Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error {
	return c.client.UpdateTestWorkflowTemplate(ctx, environmentId, workflow)
}

func (c *cloudTestWorkflowTemplateClient) Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflowTemplate) error {
	return c.client.CreateTestWorkflowTemplate(ctx, environmentId, workflow)
}

func (c *cloudTestWorkflowTemplateClient) Delete(ctx context.Context, environmentId string, name string) error {
	return c.client.DeleteTestWorkflowTemplate(ctx, environmentId, name)
}

func (c *cloudTestWorkflowTemplateClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	return c.client.DeleteTestWorkflowTemplatesByLabels(ctx, environmentId, labels)
}

func (c *cloudTestWorkflowTemplateClient) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher {
	return channels.Transform(c.client.WatchTestWorkflowTemplateUpdates(ctx, environmentId, includeInitialData), func(t *controlplaneclient.TestWorkflowTemplateUpdate) (Update, bool) {
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
