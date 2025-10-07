package webhookclient

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

var _ WebhookClient = &cloudWebhookClient{}

type cloudWebhookClient struct {
	client controlplaneclient.WebhooksClient
}

func NewCloudWebhookClient(client controlplaneclient.WebhooksClient) WebhookClient {
	return &cloudWebhookClient{client: client}
}

func (c *cloudWebhookClient) Get(ctx context.Context, environmentId string, name string) (*testkube.Webhook, error) {
	return c.client.GetWebhook(ctx, environmentId, name)
}

func (c *cloudWebhookClient) List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.Webhook, error) {
	list, err := c.client.ListWebhooks(ctx, environmentId, controlplaneclient.ListWebhookOptions{
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Offset:     options.Offset,
		Limit:      options.Limit,
	}).All()
	if err != nil {
		return nil, err
	}
	return common.MapSlice(list, func(t *testkube.Webhook) testkube.Webhook {
		return *t
	}), nil
}

func (c *cloudWebhookClient) ListLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	return c.client.ListWebhookLabels(ctx, environmentId)
}

func (c *cloudWebhookClient) Update(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	return c.client.UpdateWebhook(ctx, environmentId, webhook)
}

func (c *cloudWebhookClient) UpdateStatus(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	// For cloud storage (MongoDB), we can safely update the entire webhook document
	// since status fields are not protected like in Kubernetes custom resources
	return c.client.UpdateWebhook(ctx, environmentId, webhook)
}

func (c *cloudWebhookClient) Create(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	return c.client.CreateWebhook(ctx, environmentId, webhook)
}

func (c *cloudWebhookClient) Delete(ctx context.Context, environmentId string, name string) error {
	return c.client.DeleteWebhook(ctx, environmentId, name)
}

func (c *cloudWebhookClient) DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	return c.client.DeleteWebhooksByLabels(ctx, environmentId, labels)
}

func (c *cloudWebhookClient) GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error) {
	return "", nil
}

func (c *cloudWebhookClient) WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher {
	return channels.Transform(c.client.WatchWebhookUpdates(ctx, environmentId, includeInitialData), func(t *controlplaneclient.WebhookUpdate) (Update, bool) {
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
