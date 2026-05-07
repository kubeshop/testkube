package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhook_EscapeDots(t *testing.T) {
	w := &Webhook{
		Labels: map[string]string{
			"app.kubernetes.io/name": "test",
		},
	}

	w.EscapeDots()
	assert.NotContains(t, w.Labels, "app.kubernetes.io/name")

	w.UnscapeDots()
	assert.Equal(t, "test", w.Labels["app.kubernetes.io/name"])
}
