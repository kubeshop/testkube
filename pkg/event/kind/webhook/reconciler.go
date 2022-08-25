package webhook

import (
	executorsv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

// WebhooksLoader loads webhooks from kubernetes
type WebhooksLoader interface {
	List(selector string) (*executorsv1.WebhookList, error)
}

func NewWebhookReconciler(webhooksClient WebhooksLoader) *WebhooksReconciler {
	return &WebhooksReconciler{
		WebhooksClient: webhooksClient,
	}
}

type WebhooksReconciler struct {
	WebhooksClient WebhooksLoader
}

func (r WebhooksReconciler) Kind() string {
	return "webhook"
}

func (r WebhooksReconciler) Load() (listeners []common.Listener, err error) {
	webhookList, err := r.WebhooksClient.List("")
	if err != nil {
		return listeners, err
	}

	for _, webhook := range webhookList.Items {
		wh := NewWebhookListener(webhook.Spec.Uri, webhook.Spec.Selector, testkube.TestkubeEventTypesFromSlice(webhook.Spec.Events))
		listeners = append(listeners, wh)
	}

	return listeners, nil
}
