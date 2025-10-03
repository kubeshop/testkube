package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

type ListWebhookOptions struct {
	Labels     map[string]string
	TextSearch string
	Offset     uint32
	Limit      uint32
}

type WebhookUpdate struct {
	Type      cloud.UpdateType
	Timestamp time.Time
	Resource  *testkube.Webhook
}

type WebhooksReader channels.Watcher[*testkube.Webhook]
type WebhookWatcher channels.Watcher[*WebhookUpdate]

type WebhooksClient interface {
	GetWebhook(ctx context.Context, environmentId, name string) (*testkube.Webhook, error)
	ListWebhooks(ctx context.Context, environmentId string, options ListWebhookOptions) WebhooksReader
	ListWebhookLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	UpdateWebhook(ctx context.Context, environmentId string, webhook testkube.Webhook) error
	CreateWebhook(ctx context.Context, environmentId string, webhook testkube.Webhook) error
	DeleteWebhook(ctx context.Context, environmentId, name string) error
	DeleteWebhooksByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error)
	WatchWebhookUpdates(ctx context.Context, environmentId string, includeInitialData bool) WebhookWatcher
}

func (c *client) GetWebhook(ctx context.Context, environmentId, name string) (*testkube.Webhook, error) {
	req := &cloud.GetWebhookRequest{Name: name}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.GetWebhook, req)
	if err != nil {
		return nil, err
	}
	var webhook testkube.Webhook
	if err = json.Unmarshal(res.Webhook, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (c *client) ListWebhooks(ctx context.Context, environmentId string, options ListWebhookOptions) WebhooksReader {
	req := &cloud.ListWebhooksRequest{
		Offset:     options.Offset,
		Limit:      100,
		Labels:     options.Labels,
		TextSearch: options.TextSearch,
	}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListWebhooks, req)
	if err != nil {
		return channels.NewError[*testkube.Webhook](err)
	}
	result := channels.NewWatcher[*testkube.Webhook]()
	go func() {
		var item *cloud.WebhookListItem
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			var webhook testkube.Webhook
			err = json.Unmarshal(item.Webhook, &webhook)
			result.Send(&webhook)
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		result.Close(err)
	}()
	return result
}

func (c *client) ListWebhookLabels(ctx context.Context, environmentId string) (map[string][]string, error) {
	req := &cloud.ListWebhookLabelsRequest{}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.ListWebhookLabels, req)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string, len(res.Labels))
	for _, label := range res.Labels {
		result[label.Name] = label.Value
	}
	return result, nil
}

func (c *client) UpdateWebhook(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	webhookBytes, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	req := &cloud.UpdateWebhookRequest{Webhook: webhookBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.UpdateWebhook, req)
	return err
}

func (c *client) CreateWebhook(ctx context.Context, environmentId string, webhook testkube.Webhook) error {
	webhookBytes, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	req := &cloud.CreateWebhookRequest{Webhook: webhookBytes}
	_, err = call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.CreateWebhook, req)
	return err
}

func (c *client) DeleteWebhook(ctx context.Context, environmentId, name string) error {
	req := &cloud.DeleteWebhookRequest{Name: name}
	_, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteWebhook, req)
	return err
}

func (c *client) DeleteWebhooksByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error) {
	req := &cloud.DeleteWebhooksByLabelsRequest{Labels: labels}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.DeleteWebhooksByLabels, req)
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

func (c *client) WatchWebhookUpdates(ctx context.Context, environmentId string, includeInitialData bool) WebhookWatcher {
	req := &cloud.WatchWebhookUpdatesRequest{IncludeInitialData: includeInitialData}
	res, err := call(ctx, c.metadata().SetEnvironmentID(environmentId).GRPC(), c.client.WatchWebhookUpdates, req)
	if err != nil {
		return channels.NewError[*WebhookUpdate](err)
	}
	watcher := channels.NewWatcher[*WebhookUpdate]()
	go func() {
		var item *cloud.WebhookUpdate
		for err == nil {
			item, err = res.Recv()
			if err != nil {
				break
			}
			if item.Ping {
				continue
			}
			var resource testkube.Webhook
			err = json.Unmarshal(item.Resource, &resource)
			watcher.Send(&WebhookUpdate{
				Type:      item.Type,
				Timestamp: item.Timestamp.AsTime(),
				Resource:  &resource,
			})
		}
		if errors.Is(err, io.EOF) {
			err = nil
		}
		watcher.Close(err)
	}()
	return watcher
}
