package executors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	executorsv1 "github.com/kubeshop/testkube/api/executor/v1"
)

func TestWebhooks(t *testing.T) {
	var wClient *WebhooksClient
	testWebhooks := []*executorsv1.Webhook{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-webhook1",
				Namespace: "test-ns",
			},
			Spec: executorsv1.WebhookSpec{
				Events: []executorsv1.EventType{"test-event1"},
			},
			Status: executorsv1.WebhookStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-webhook2",
				Namespace: "test-ns",
			},
			Spec: executorsv1.WebhookSpec{
				Events: []executorsv1.EventType{"test-event2"},
			},
			Status: executorsv1.WebhookStatus{},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-webhook3",
				Namespace: "test-ns",
			},
			Spec: executorsv1.WebhookSpec{
				Events: []executorsv1.EventType{"test-event1"},
			},
			Status: executorsv1.WebhookStatus{},
		},
	}

	t.Run("NewWebhooksClient", func(t *testing.T) {
		clientBuilder := fake.NewClientBuilder()

		groupVersion := schema.GroupVersion{Group: "executor.testkube.io", Version: "v1"}
		schemaBuilder := scheme.Builder{GroupVersion: groupVersion}
		schemaBuilder.Register(&executorsv1.WebhookList{})
		schemaBuilder.Register(&executorsv1.Webhook{})

		schema, err := schemaBuilder.Build()
		assert.NoError(t, err)
		assert.NotEmpty(t, schema)
		clientBuilder.WithScheme(schema)

		kClient := clientBuilder.Build()
		testNamespace := "test-ns"
		wClient = NewWebhooksClient(kClient, testNamespace)
		assert.NotEmpty(t, wClient)
		assert.Equal(t, testNamespace, wClient.Namespace)
	})
	t.Run("WebhookCreate", func(t *testing.T) {
		t.Run("Create new webhooks", func(t *testing.T) {
			for _, w := range testWebhooks {
				created, err := wClient.Create(w)
				assert.NoError(t, err)
				assert.Equal(t, w.Name, created.Name)

				res, err := wClient.Get(w.Name)
				assert.NoError(t, err)
				assert.Equal(t, w.Name, res.Name)
			}
		})
		t.Run("Create should fail on webhook with wrong namespace", func(t *testing.T) {
			wrongNsWebhook := executorsv1.Webhook{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-webhook",
					Namespace: "wrong-ns",
				},
				Spec: executorsv1.WebhookSpec{
					Events: []executorsv1.EventType{"test-event"},
				},
				Status: executorsv1.WebhookStatus{},
			}

			_, err := wClient.Create(&wrongNsWebhook)
			assert.Error(t, err)
		})
	})
	t.Run("WebhookList", func(t *testing.T) {
		t.Run("List without selector", func(t *testing.T) {
			l, err := wClient.List("")
			assert.NoError(t, err)
			assert.Equal(t, len(testWebhooks), len(l.Items))
		})
	})
	t.Run("WebhookGet", func(t *testing.T) {
		t.Run("Get webhook with empty name", func(t *testing.T) {
			t.Parallel()
			_, err := wClient.Get("")
			assert.Error(t, err)
		})

		t.Run("Get webhook with non existent name", func(t *testing.T) {
			t.Parallel()
			_, err := wClient.Get("no-webhook")
			assert.Error(t, err)
		})

		t.Run("Get existing webhook", func(t *testing.T) {
			res, err := wClient.Get(testWebhooks[0].Name)
			assert.NoError(t, err)
			assert.Equal(t, testWebhooks[0].Name, res.Name)
		})
	})
	t.Run("WebhookGetByEvent", func(t *testing.T) {
		t.Run("GetByEvent with non-existent event", func(t *testing.T) {
			res, err := wClient.GetByEvent("no-event")
			assert.NoError(t, err)
			assert.Equal(t, 0, len(res.Items))
		})
		t.Run("Get webhook by existing event", func(t *testing.T) {
			res, err := wClient.GetByEvent(testWebhooks[1].Spec.Events[0])
			assert.NoError(t, err)
			assert.Equal(t, testWebhooks[1].Name, res.Items[0].Name)
			assert.Equal(t, testWebhooks[1].Spec.Events[0], res.Items[0].Spec.Events[0])
		})
		t.Run("GetByEvent with multiple webhooks for one event", func(t *testing.T) {
			res, err := wClient.GetByEvent("test-event1")
			assert.NoError(t, err)
			assert.Equal(t, 2, len(res.Items))
		})
	})
	t.Run("WebhookDelete", func(t *testing.T) {
		t.Run("Delete items", func(t *testing.T) {
			for _, webhook := range testWebhooks {
				w, err := wClient.Get(webhook.Name)
				assert.NoError(t, err)
				assert.Equal(t, w.Name, webhook.Name)

				err = wClient.Delete(webhook.Name)
				assert.NoError(t, err)

				_, err = wClient.Get(webhook.Name)
				assert.Error(t, err)
			}
		})

		t.Run("Delete non-existent item", func(t *testing.T) {
			_, err := wClient.Get("no-webhook")
			assert.Error(t, err)

			err = wClient.Delete("no-webhook")
			assert.Error(t, err)
		})
	})
}
