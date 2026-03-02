package webhookclient

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/webhooks"
)

// CloudWebhookClient lists webhooks from the Control Plane on demand and exposes the same interface as the K8s client.
type CloudWebhookClient struct {
	client    controlplaneclient.WebhooksClient
	envID     string
	namespace string
	log       *zap.SugaredLogger
}

// NewCloudWebhookClient builds a Webhook client backed by the Control Plane.
func NewCloudWebhookClient(
	client controlplaneclient.WebhooksClient,
	envID string,
	namespace string,
	logger *zap.SugaredLogger,
) *CloudWebhookClient {
	if logger == nil {
		logger = log.DefaultLogger
	}

	c := &CloudWebhookClient{
		client:    client,
		envID:     envID,
		namespace: namespace,
		log:       logger,
	}

	return c
}

// List returns the cached webhooks filtered by the provided label selector.
func (c *CloudWebhookClient) List(selector string) (*executorv1.WebhookList, error) {
	reqs, err := labels.ParseToRequirements(selector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %w", err)
	}

	webhooksList, err := c.client.ListWebhooks(context.Background(), c.envID, controlplaneclient.ListWebhookOptions{}, c.namespace)
	if err != nil {
		return nil, err
	}

	items := make([]executorv1.Webhook, 0, len(webhooksList))
	for i := range webhooksList {
		items = append(items, webhooks.MapAPIToCRD(webhooksList[i]))
	}

	sel := labels.NewSelector().Add(reqs...)
	if selector == "" {
		return &executorv1.WebhookList{Items: items}, nil
	}

	filtered := make([]executorv1.Webhook, 0, len(items))
	for i := range items {
		if sel.Matches(labels.Set(items[i].Labels)) {
			filtered = append(filtered, items[i])
		}
	}

	return &executorv1.WebhookList{Items: filtered}, nil
}
