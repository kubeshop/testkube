package webhookclient

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/log"
)

type stubWebhooksClient struct {
	listFn func() ([]testkube.Webhook, error)
	calls  atomic.Int32
	mu     sync.Mutex
	last   controlplaneclient.ListWebhookOptions
}

func (s *stubWebhooksClient) ListWebhooks(ctx context.Context, environmentId string, options controlplaneclient.ListWebhookOptions, namespace string) ([]testkube.Webhook, error) {
	s.calls.Add(1)
	s.mu.Lock()
	s.last = options
	s.mu.Unlock()
	if s.listFn != nil {
		return s.listFn()
	}
	return nil, nil
}

func TestCloudWebhookClientFiltersBySelector(t *testing.T) {
	stub := &stubWebhooksClient{
		listFn: func() ([]testkube.Webhook, error) {
			return []testkube.Webhook{
				{
					Name:      "match",
					Namespace: "ns",
					Labels:    map[string]string{"team": "dev"},
					Uri:       "http://example",
				},
				{
					Name:      "skip",
					Namespace: "ns",
					Labels:    map[string]string{"team": "qa"},
					Uri:       "http://example2",
				},
			}, nil
		},
	}

	client := NewCloudWebhookClient(stub, "env1", "ns", log.DefaultLogger)

	list, err := client.List("team=dev")
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "match", list.Items[0].Name)
	assert.Equal(t, "ns", list.Items[0].Namespace)
	assert.EqualValues(t, 1, stub.calls.Load())
}

func TestCloudWebhookClientReportsSelectorError(t *testing.T) {
	stub := &stubWebhooksClient{
		listFn: func() ([]testkube.Webhook, error) {
			return []testkube.Webhook{}, nil
		},
	}

	client := NewCloudWebhookClient(stub, "env1", "ns", log.DefaultLogger)

	list, err := client.List("!!!")
	assert.Nil(t, list)
	assert.Error(t, err)
}

func TestCloudWebhookClientReturnsLatestOnEachCall(t *testing.T) {
	var data atomic.Value
	data.Store([]testkube.Webhook{
		{Name: "first", Namespace: "ns"},
	})

	stub := &stubWebhooksClient{
		listFn: func() ([]testkube.Webhook, error) {
			return data.Load().([]testkube.Webhook), nil
		},
	}

	client := NewCloudWebhookClient(stub, "env1", "ns", log.DefaultLogger)

	firstList, err := client.List("")
	require.NoError(t, err)
	require.Len(t, firstList.Items, 1)
	assert.Equal(t, "first", firstList.Items[0].Name)

	data.Store([]testkube.Webhook{
		{Name: "second", Namespace: "ns"},
		{Name: "third", Namespace: "ns"},
	})

	updated, err := client.List("")
	require.NoError(t, err)
	require.Len(t, updated.Items, 2)

	names := map[string]struct{}{}
	for _, wh := range updated.Items {
		names[wh.Name] = struct{}{}
	}
	assert.Contains(t, names, "second")
	assert.Contains(t, names, "third")
	assert.GreaterOrEqual(t, stub.calls.Load(), int32(2))
}
