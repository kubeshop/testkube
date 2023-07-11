package webhook

import (
	"fmt"

	executorsv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/mapper/webhooks"
)

var _ common.ListenerLoader = (*WebhooksLoader)(nil)

// WebhooksLister loads webhooks from kubernetes
type WebhooksLister interface {
	List(selector string) (*executorsv1.WebhookList, error)
}

func NewWebhookLoader(webhooksClient WebhooksLister) *WebhooksLoader {
	return &WebhooksLoader{
		WebhooksClient: webhooksClient,
	}
}

type WebhooksLoader struct {
	WebhooksClient WebhooksLister
}

func (r WebhooksLoader) Kind() string {
	return "webhook"
}

func (r WebhooksLoader) Load() (listeners common.Listeners, err error) {
	// load all webhooks from kubernetes CRDs
	webhookList, err := r.WebhooksClient.List("")
	if err != nil {
		return listeners, err
	}

	// and create listeners for each webhook spec
	for _, webhook := range webhookList.Items {
		types := webhooks.MapEventArrayToCRDEvents(webhook.Spec.Events)
		name := fmt.Sprintf("%s.%s", webhook.ObjectMeta.Namespace, webhook.ObjectMeta.Name)
		listeners = append(listeners, NewWebhookListener(name, webhook.Spec.Uri, webhook.Spec.Selector, types, webhook.Spec.PayloadObjectField, webhook.Spec.PayloadTemplate, webhook.Spec.Headers))
	}

	return listeners, nil
}
