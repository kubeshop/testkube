package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	executorsv1 "github.com/kubeshop/testkube/api/executor/v1"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
)

func TestWebhookLoader(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
	mockWebhookTemplatesClient := executorsclientv1.NewMockWebhookTemplatesInterface(mockCtrl)
	mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: executorsv1.WebhookSpec{Uri: "http://localhost:3333", Events: []executorsv1.EventType{"start-test"}, PayloadObjectField: "text", PayloadTemplate: "{{ .Id }}", Headers: map[string]string{"Content-Type": "application/xml"}}},
		},
	}, nil).AnyTimes()
	mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)

	webhooksLoader := NewWebhookLoader(
		mockWebhooksClient,
		WithWebhookTemplateClient(mockWebhookTemplatesClient),
		WithMetrics(v1.NewMetrics()),
		WithWebhookResultsRepository(mockWebhookRepository),
	)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}

func TestWebhookTemplateLoader(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
	mockWebhookTemplatesClient := executorsclientv1.NewMockWebhookTemplatesInterface(mockCtrl)
	mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: executorsv1.WebhookSpec{WebhookTemplateRef: &executorsv1.WebhookTemplateRef{Name: "name"}}},
		},
	}, nil).AnyTimes()
	mockWebhookTemplatesClient.EXPECT().Get("name").Return(&executorsv1.WebhookTemplate{
		Spec: executorsv1.WebhookTemplateSpec{
			Uri: "http://localhost:3333", Events: []executorsv1.EventType{"start-test"}, PayloadObjectField: "text", PayloadTemplate: "{{ .Id }}", Headers: map[string]string{"Content-Type": "application/xml"},
		},
	}, nil).AnyTimes()

	mockWebhookRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)

	webhooksLoader := NewWebhookLoader(
		mockWebhooksClient,
		WithWebhookTemplateClient(mockWebhookTemplatesClient),
		WithMetrics(v1.NewMetrics()),
		WithWebhookResultsRepository(mockWebhookRepository),
	)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
