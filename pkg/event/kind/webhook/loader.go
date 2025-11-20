package webhook

import (
	"fmt"
	"sort"

	"go.uber.org/zap"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/webhooks"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secret"
)

var _ common.ListenerLoader = (*WebhooksLoader)(nil)

// WebhookLoaderOption is an option for NewWebhookLoader
type WebhookLoaderOption func(*WebhooksLoader)

//go:generate go tool mockgen -destination=./mock_webhook_client.go -package=webhook "github.com/kubeshop/testkube/pkg/event/kind/webhook" WebhookClient
type WebhookClient interface {
	List(selector string) (*executorv1.WebhookList, error)
}

// NewWebhookLoader creates a new WebhooksLoader
func NewWebhookLoader(
	webhookClient WebhookClient,
	opts ...WebhookLoaderOption,
) *WebhooksLoader {
	loader := &WebhooksLoader{
		log:           log.DefaultLogger,
		webhookClient: webhookClient,
	}

	for _, opt := range opts {
		opt(loader)
	}

	return loader
}

type WebhooksLoader struct {
	log           *zap.SugaredLogger
	webhookClient WebhookClient

	// Optional fields
	testWorkflowResultsRepository testworkflow.Repository
	webhookResultsRepository      cloudwebhook.WebhookRepository
	webhookTemplateClient         executorsclientv1.WebhookTemplatesInterface
	secretClient                  secret.Interface
	metrics                       v1.Metrics
	envs                          map[string]string
	dashboardURI                  string
	orgID                         string
	envID                         string
}

// WithTestWorkflowResultsRepository sets the test workflow results repository
func WithTestWorkflowResultsRepository(repo testworkflow.Repository) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.testWorkflowResultsRepository = repo
	}
}

// WithWebhookResultsRepository sets the repository used for collecting webhook results
func WithWebhookResultsRepository(repo cloudwebhook.WebhookRepository) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.webhookResultsRepository = repo
	}
}

// WithWebhookTemplateClient sets the webhook template client
func WithWebhookTemplateClient(client executorsclientv1.WebhookTemplatesInterface) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.webhookTemplateClient = client
	}
}

// WithSecretClient sets the secret client
func WithSecretClient(client secret.Interface) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.secretClient = client
	}
}

// WithMetrics sets the metrics
func WithMetrics(metrics v1.Metrics) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.metrics = metrics
	}
}

// WithEnvs sets the agent's environment variables to be used in templates
func WithEnvs(envs map[string]string) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.envs = envs
	}
}

// WithDashboardURI sets the dashboard URI for the connection to the control plane
// to be used in templates
func WithDashboardURI(dashboardURI string) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.dashboardURI = dashboardURI
	}
}

// WithOrgID sets the organization ID for the connection to the control plane
// to be used in templates
func WithOrgID(orgID string) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.orgID = orgID
	}
}

// WithEnvID sets the environment ID for the connection to the control plane
// to be used in templates
func WithEnvID(envID string) WebhookLoaderOption {
	return func(loader *WebhooksLoader) {
		loader.envID = envID
	}
}

func (r WebhooksLoader) Kind() string {
	return "webhook"
}

func (r WebhooksLoader) Load() (listeners common.Listeners, err error) {
	// load all webhooks from kubernetes CRDs
	webhookList, err := r.webhookClient.List("")
	if err != nil {
		return listeners, err
	}

	// and create listeners for each webhook spec
	for _, webhook := range webhookList.Items {
		if webhook.Spec.WebhookTemplateRef != nil && webhook.Spec.WebhookTemplateRef.Name != "" {
			if r.webhookTemplateClient == nil {
				r.log.Errorw("webhook using unsupported WebhookTemplateRef", "name", webhook.Name, "template_ref", webhook.Spec.WebhookTemplateRef)
				continue
			}
			webhookTemplate, err := r.webhookTemplateClient.Get(webhook.Spec.WebhookTemplateRef.Name)
			if err != nil {
				r.log.Errorw("error webhook template loading", "error", err, "name", webhook.Name, "template", webhook.Spec.WebhookTemplateRef.Name)
				continue
			}

			if webhookTemplate.Spec.Disabled {
				r.log.Errorw("error webhook template is disabled", "name", webhook.Name, "template", webhook.Spec.WebhookTemplateRef.Name)
				continue
			}

			webhook = mergeWebhooks(webhook, *webhookTemplate)
		}

		if webhook.Spec.PayloadTemplate == "" && webhook.Spec.PayloadTemplateReference != "" {
			r.log.Errorw("webhook using deprecated PayloadTemplateReference", "name", webhook.Name, "template_ref", webhook.Spec.PayloadTemplateReference)
			continue
		}

		eventTypes := webhooks.MapEventArrayToCRDEvents(webhook.Spec.Events)
		name := fmt.Sprintf("%s.%s", webhook.Namespace, webhook.Name)
		listeners = append(
			listeners,
			NewWebhookListener(
				name,
				webhook.Spec.Uri,
				webhook.Spec.Selector,
				eventTypes,
				webhook.Spec.PayloadObjectField,
				webhook.Spec.PayloadTemplate,
				webhook.Spec.Headers,
				webhook.Spec.Disabled,
				webhook.Spec.Config,
				webhook.Spec.Parameters,
				listenerWithTestWorkflowResultsRepository(r.testWorkflowResultsRepository),
				listenerWithWebhookResultsRepository(r.webhookResultsRepository),
				listenerWithMetrics(r.metrics),
				listenerWithSecretClient(r.secretClient),
				listenerWithEnvs(r.envs),
				ListenerWithDashboardURI(r.dashboardURI),
				ListenerWithOrgID(r.orgID),
				ListenerWithEnvID(r.envID),
			),
		)
	}

	return listeners, nil
}

func mergeWebhooks(dst executorv1.Webhook, src executorv1.WebhookTemplate) executorv1.Webhook {
	maps := []struct {
		d *map[string]string
		s *map[string]string
	}{
		{
			&dst.Labels,
			&src.Labels,
		},
		{
			&dst.Annotations,
			&src.Annotations,
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

	items := []struct {
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

	sort.Slice(dst.Spec.Events, func(i, j int) bool {
		return dst.Spec.Events[i] < dst.Spec.Events[j]
	})

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
		srcParameters := make(map[string]executorv1.WebhookParameterSchema)
		for _, parameter := range src.Spec.Parameters {
			srcParameters[parameter.Name] = parameter
		}

		dstParameters := make(map[string]executorv1.WebhookParameterSchema)
		for _, parameter := range dst.Spec.Parameters {
			dstParameters[parameter.Name] = parameter
		}

		for name, parameter := range srcParameters {
			if _, ok := dstParameters[name]; !ok {
				dst.Spec.Parameters = append(dst.Spec.Parameters, parameter)
			}
		}

		sort.Slice(dst.Spec.Parameters, func(i, j int) bool {
			return dst.Spec.Parameters[i].Name < dst.Spec.Parameters[j].Name
		})
	}

	return dst
}
