package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap"

	executorsv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
)

func TestWebhookLoader(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTemplatesClient := templatesclientv1.NewMockInterface(mockCtrl)
	mockWebhooksClient := executorsclientv1.NewMockWebhooksInterface(mockCtrl)
	mockWebhookTemplatesClient := executorsclientv1.NewMockWebhookTemplatesInterface(mockCtrl)
	mockWebhooksClient.EXPECT().List(gomock.Any()).Return(&executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: executorsv1.WebhookSpec{Uri: "http://localhost:3333", Events: []executorsv1.EventType{"start-test"}, PayloadObjectField: "text", PayloadTemplate: "{{ .Id }}", Headers: map[string]string{"Content-Type": "application/xml"}}},
		},
	}, nil).AnyTimes()
	mockDeprecatedClients := commons.NewMockDeprecatedClients(mockCtrl)
	mockDeprecatedClients.EXPECT().Templates().Return(mockTemplatesClient).AnyTimes()
	mockWebhooRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)

	webhooksLoader := NewWebhookLoader(zap.NewNop().Sugar(), mockWebhooksClient, mockWebhookTemplatesClient, mockDeprecatedClients, nil, nil, nil, v1.NewMetrics(), mockWebhooRepository, nil, nil)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}

func TestWebhookTemplateLoader(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTemplatesClient := templatesclientv1.NewMockInterface(mockCtrl)
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

	mockDeprecatedClients := commons.NewMockDeprecatedClients(mockCtrl)
	mockDeprecatedClients.EXPECT().Templates().Return(mockTemplatesClient).AnyTimes()
	mockWebhooRepository := cloudwebhook.NewMockWebhookRepository(mockCtrl)

	webhooksLoader := NewWebhookLoader(zap.NewNop().Sugar(), mockWebhooksClient, mockWebhookTemplatesClient, mockDeprecatedClients, nil, nil, nil, v1.NewMetrics(), mockWebhooRepository, nil, nil)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
