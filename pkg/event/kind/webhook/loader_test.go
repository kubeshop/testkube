package webhook

import (
	"testing"

	executorsv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/stretchr/testify/assert"
)

type DummyLoader struct {
}

func (l DummyLoader) List(selector string) (*executorsv1.WebhookList, error) {
	return &executorsv1.WebhookList{
		Items: []executorsv1.Webhook{
			{Spec: executorsv1.WebhookSpec{Uri: "http://localhost:3333", Events: []string{"start-test"}}},
		},
	}, nil
}

func TestWebhookLoader(t *testing.T) {

	webhooksLoader := NewWebhookLoader(&DummyLoader{})
	listeners, err := webhooksLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
