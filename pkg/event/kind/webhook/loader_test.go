package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"

	executorsv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
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
	webhooksLoader := NewWebhookLoader(&DummyLoader{})
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
