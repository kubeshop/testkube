package controlplaneclient

import (
	"context"
	"encoding/json"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

type ListWebhookOptions struct {
	Labels     map[string]string
	TextSearch string
	Selector   string
	Offset     uint32
	Limit      uint32
}

type WebhooksClient interface {
	ListWebhooks(ctx context.Context, environmentId string, options ListWebhookOptions, namespace string) ([]testkube.Webhook, error)
}

var _ WebhooksClient = (*client)(nil)

func (c *client) ListWebhooks(ctx context.Context, environmentId string, options ListWebhookOptions, namespace string) ([]testkube.Webhook, error) {
	req := &cloud.ListWebhooksV2Request{
		Offset:     options.Offset,
		Limit:      options.Limit,
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
		Selector:   options.Selector,
		Namespace:  namespace,
	}

	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListWebhooksV2, req)
	if err != nil {
		return nil, err
	}

	webhooks := make([]testkube.Webhook, 0)
	for _, item := range res.Items {
		var webhook testkube.Webhook
		if err := json.Unmarshal(item.Webhook, &webhook); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}
