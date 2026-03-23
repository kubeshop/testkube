package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/utils"
)

func TestWebhook_ConvertDots_Target(t *testing.T) {
	w := &Webhook{
		Target: map[string]string{
			"app.kubernetes.io/name": "webhook-agent",
			"region":                 "eu",
		},
	}

	escapedKey := utils.EscapeDots("app.kubernetes.io/name")

	w.EscapeDots()
	assert.Equal(t, "webhook-agent", w.Target[escapedKey])
	assert.Equal(t, "eu", w.Target["region"])
	assert.NotContains(t, w.Target, "app.kubernetes.io/name")

	w.UnscapeDots()
	assert.Equal(t, "webhook-agent", w.Target["app.kubernetes.io/name"])
	assert.NotContains(t, w.Target, escapedKey)
}
