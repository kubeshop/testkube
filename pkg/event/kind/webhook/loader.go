package webhook

import (
	"fmt"

	"go.uber.org/zap"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/mapper/webhooks"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secret"
)

var _ common.ListenerLoader = (*WebhooksLoader)(nil)

func NewWebhookLoader(log *zap.SugaredLogger, webhooksClient executorsclientv1.WebhooksInterface, deprecatedClients commons.DeprecatedClients,
	deprecatedRepositories commons.DeprecatedRepositories, testWorkflowExecutionResults testworkflow.Repository,
	secretClient secret.Interface, metrics v1.Metrics, proContext *config.ProContext, envs map[string]string,
) *WebhooksLoader {
	return &WebhooksLoader{
		log:                          log,
		WebhooksClient:               webhooksClient,
		deprecatedClients:            deprecatedClients,
		deprecatedRepositories:       deprecatedRepositories,
		testWorkflowExecutionResults: testWorkflowExecutionResults,
		secretClient:                 secretClient,
		metrics:                      metrics,
		proContext:                   proContext,
		envs:                         envs,
	}
}

type WebhooksLoader struct {
	log                          *zap.SugaredLogger
	WebhooksClient               executorsclientv1.WebhooksInterface
	deprecatedClients            commons.DeprecatedClients
	deprecatedRepositories       commons.DeprecatedRepositories
	testWorkflowExecutionResults testworkflow.Repository
	secretClient                 secret.Interface
	metrics                      v1.Metrics
	proContext                   *config.ProContext
	envs                         map[string]string
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
		if webhook.Spec.WebhookTemplateRef != nil && webhook.Spec.WebhookTemplateRef.Name != "" {
			webhookTemplate, err := r.WebhooksClient.Get(webhook.Spec.WebhookTemplateRef.Name)
			if err != nil {
				r.log.Errorw("error webhook template loading", "error", err, "name", webhook.Name, "template", webhook.Spec.WebhookTemplateRef.Name)
				continue
			}

			webhook = mergeWebhooks(webhook, *webhookTemplate)
		}

		payloadTemplate := ""
		if webhook.Spec.PayloadTemplateReference != "" {
			if r.deprecatedClients == nil {
				r.log.Errorw("webhook using deprecated PayloadTemplateReference", "name", webhook.Name, "template", webhook.Spec.PayloadTemplateReference)
				continue
			}
			template, err := r.deprecatedClients.Templates().Get(webhook.Spec.PayloadTemplateReference)
			if err != nil {
				return listeners, err
			}

			if template.Spec.Type_ != nil && testkube.TemplateType(*template.Spec.Type_) == testkube.WEBHOOK_TemplateType {
				payloadTemplate = template.Spec.Body
			} else {
				r.log.Warnw("not matching template type", "template", webhook.Spec.PayloadTemplateReference)
			}
		}

		if webhook.Spec.PayloadTemplate != "" {
			payloadTemplate = webhook.Spec.PayloadTemplate
		}

		types := webhooks.MapEventArrayToCRDEvents(webhook.Spec.Events)
		name := fmt.Sprintf("%s.%s", webhook.ObjectMeta.Namespace, webhook.ObjectMeta.Name)
		vars := make(map[string]string)
		for key, value := range webhook.Spec.Config {
			data := ""
			if value.Public != nil {
				data = *value.Public
			}

			if value.Private != nil {
				var ns []string
				if value.Private.Namespace != "" {
					ns = append(ns, value.Private.Namespace)
				}

				elements, err := r.secretClient.Get(value.Private.Name, ns...)
				if err != nil {
					r.log.Errorw("error secret loading", "error", err, "name", value.Private.Name)
					continue
				}

				if element, ok := elements[value.Private.Key]; ok {
					data = element
				} else {
					r.log.Errorw("error secret key finding loading", "name", value.Private.Name, "key", value.Private.Key)
					continue
				}
			}

			vars[key] = data
		}

		for key, value := range webhook.Spec.Parameters {
			if _, ok := vars[key]; !ok && value.Default_ != nil {
				vars[key] = *value.Default_
			}
		}

		listeners = append(
			listeners,
			NewWebhookListener(
				name, webhook.Spec.Uri, webhook.Spec.Selector, types,
				webhook.Spec.PayloadObjectField, payloadTemplate, webhook.Spec.Headers, webhook.Spec.Disabled,
				r.deprecatedRepositories, r.testWorkflowExecutionResults,
				r.metrics, r.proContext, r.envs, vars,
			),
		)
	}

	return listeners, nil
}

func mergeWebhooks(dst, src executorv1.Webhook) executorv1.Webhook {
	var maps = []struct {
		d *map[string]string
		s *map[string]string
	}{
		{
			&dst.ObjectMeta.Labels,
			&src.ObjectMeta.Labels,
		},
		{
			&dst.ObjectMeta.Annotations,
			&src.ObjectMeta.Annotations,
		},
		{
			&dst.Spec.Headers,
			&src.Spec.Headers,
		},
	}

	for _, m := range maps {
		if *m.s != nil {
			if *m.d == nil {
				*m.d = map[string]string{}
			}

			for key, value := range *m.s {
				if _, ok := (*m.d)[key]; !ok {
					(*m.d)[key] = value
				}
			}
		}
	}

	var items = []struct {
		d *string
		s *string
	}{
		{
			&dst.Spec.Uri,
			&src.Spec.Uri,
		},
		{
			&dst.Spec.Selector,
			&src.Spec.Selector,
		},
		{
			&dst.Spec.PayloadObjectField,
			&src.Spec.PayloadObjectField,
		},
		{
			&dst.Spec.PayloadTemplate,
			&src.Spec.PayloadTemplate,
		},
		{
			&dst.Spec.PayloadTemplateReference,
			&src.Spec.PayloadTemplateReference,
		},
	}

	for _, item := range items {
		if *item.d == "" && *item.s != "" {
			*item.d = *item.s
		}
	}

	srcEventTypes := make(map[executorv1.EventType]struct{})
	for _, eventType := range src.Spec.Events {
		srcEventTypes[eventType] = struct{}{}
	}

	dstEventTypes := make(map[executorv1.EventType]struct{})
	for _, eventType := range dst.Spec.Events {
		dstEventTypes[eventType] = struct{}{}
	}

	for evenType := range srcEventTypes {
		if _, ok := dstEventTypes[evenType]; !ok {
			dst.Spec.Events = append(dst.Spec.Events, evenType)
		}
	}

	if !dst.Spec.Disabled && src.Spec.Disabled {
		dst.Spec.Disabled = src.Spec.Disabled
	}

	if src.Spec.Config != nil {
		if dst.Spec.Config == nil {
			dst.Spec.Config = map[string]executorv1.WebhookConfigValue{}
		}

		for key, value := range src.Spec.Config {
			if _, ok := (dst.Spec.Config)[key]; !ok {
				dst.Spec.Config[key] = value
			}
		}
	}

	if src.Spec.Parameters != nil {
		if dst.Spec.Parameters == nil {
			dst.Spec.Parameters = map[string]executorv1.WebhookParameterSchema{}
		}

		for key, value := range src.Spec.Parameters {
			if _, ok := (dst.Spec.Parameters)[key]; !ok {
				dst.Spec.Parameters[key] = value
			}
		}
	}

	return dst
}
