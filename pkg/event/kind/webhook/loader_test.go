package webhook

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	executorsv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
)

type DummyLoader struct {
}

func (l DummyLoader) List(selector string) (*executorsv1.WebhookList, error) {
	return &executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: executorsv1.WebhookSpec{Uri: "http://localhost:3333", Events: []executorsv1.EventType{"start-test"}, PayloadObjectField: "text", PayloadTemplate: "{{ .Id }}", Headers: map[string]string{"Content-Type": "application/xml"}}},
		},
	}, nil
}

func TestWebhookLoader(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTemplatesClient := templatesclientv1.NewMockInterface(mockCtrl)
	webhooksLoader := NewWebhookLoader(zap.NewNop().Sugar(), &DummyLoader{}, mockTemplatesClient)
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
