package cdevent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookLoader(t *testing.T) {
	t.Parallel()

	cdeventLoader, err := NewCDEventLoader("target", "", "", "", nil)
	assert.NoError(t, err)

	listeners, err := cdeventLoader.Load()

	assert.Equal(t, 1, len(listeners))
	assert.NoError(t, err)
}
