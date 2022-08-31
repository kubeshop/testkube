package webhook

import (
	executorsv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

var _ common.ListenerLoader = &WebhooksLoader{}

// WebhooksLoader loads webhooks from kubernetes
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
		wh := NewWebhookListener(webhook.Spec.Uri, webhook.Spec.Selector, testkube.TestkubeEventTypesFromSlice(webhook.Spec.Events))
		listeners = append(listeners, wh)
	}

	return listeners, nil
}
