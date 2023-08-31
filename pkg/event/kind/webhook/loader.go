package webhook

import (
	"fmt"

	executorsv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	templatesclientv1 "github.com/kubeshop/testkube-operator/client/templates/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/mapper/webhooks"
)

var _ common.ListenerLoader = (*WebhooksLoader)(nil)

// WebhooksLister loads webhooks from kubernetes
type WebhooksLister interface {
	List(selector string) (*executorsv1.WebhookList, error)
}

func NewWebhookLoader(webhooksClient WebhooksLister, templatesClient templatesclientv1.Interface) *WebhooksLoader {
	return &WebhooksLoader{
		WebhooksClient:  webhooksClient,
		templatesClient: templatesClient,
	}
}

type WebhooksLoader struct {
	WebhooksClient  WebhooksLister
	templatesClient templatesclientv1.Interface
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
		payloadTemplate := ""
		if webhook.Spec.PayloadTemplateReference != "" {
			template, err := r.templatesClient.Get(webhook.Spec.PayloadTemplateReference)
			if err != nil {
				return listeners, err
			}

			if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.WEBHOOK_TemplateType {
				payloadTemplate = template.Spec.Body
			}
		}

		if webhook.Spec.PayloadTemplate != "" {
			payloadTemplate = webhook.Spec.PayloadTemplate
		}

		types := webhooks.MapEventArrayToCRDEvents(webhook.Spec.Events)
		name := fmt.Sprintf("%s.%s", webhook.ObjectMeta.Namespace, webhook.ObjectMeta.Name)
		listeners = append(listeners, NewWebhookListener(name, webhook.Spec.Uri, webhook.Spec.Selector, types, webhook.Spec.PayloadObjectField, payloadTemplate, webhook.Spec.Headers))
	}

	return listeners, nil
}
