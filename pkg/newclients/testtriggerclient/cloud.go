package testtriggerclient

import (
	"context"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

var _ TestTriggerClient = &cloudTestTriggerClient{}

type cloudTestTriggerClient struct {
	client controlplaneclient.TestTriggersClient
}

func NewCloudTestTriggerClient(client controlplaneclient.TestTriggersClient) TestTriggerClient {
	return &cloudTestTriggerClient{client: client}
}

func (c *cloudTestTriggerClient) Get(ctx context.Context, environmentId string, name string, namespace string) (*testkube.TestTrigger, error) {
	return c.client.GetTestTrigger(ctx, environmentId, name, namespace)
}

func (c *cloudTestTriggerClient) List(ctx context.Context, environmentId string, options ListOptions, namespace string) ([]testkube.TestTrigger, error) {
	list, err := c.client.ListTestTriggers(ctx, environmentId, controlplaneclient.ListTestTriggerOptions{
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Selector:   options.Selector,
		Offset:     options.Offset,
		Limit:      options.Limit,
	}, namespace).All()
	if err != nil {
		return nil, err
	}
	return common.MapSlice(list, func(t *testkube.TestTrigger) testkube.TestTrigger {
		return *t
	}), nil
}

func (c *cloudTestTriggerClient) Update(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error {
	return c.client.UpdateTestTrigger(ctx, environmentId, trigger)
}

func (c *cloudTestTriggerClient) Create(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error {
	return c.client.CreateTestTrigger(ctx, environmentId, trigger)
}

func (c *cloudTestTriggerClient) Delete(ctx context.Context, environmentId string, name string, namespace string) error {
	return c.client.DeleteTestTrigger(ctx, environmentId, name, namespace)
}

func (c *cloudTestTriggerClient) DeleteAll(ctx context.Context, environmentId string, namespace string) (uint32, error) {
	return c.client.DeleteAllTestTriggers(ctx, environmentId, namespace)
}

func (c *cloudTestTriggerClient) DeleteByLabels(ctx context.Context, environmentId string, selector string, namespace string) (uint32, error) {
	return c.client.DeleteTestTriggersByLabels(ctx, environmentId, selector, namespace)
}
