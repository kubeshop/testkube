package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnonymyzeHost(t *testing.T) {
	t.Run("testkube based hosts", func(t *testing.T) {
		assert.Equal(t, APIHostTestkubeInternal, AnonymizeHost("dashboard.testkube.io"))
	})
	t.Run("localhosts", func(t *testing.T) {
		assert.Equal(t, APIHostLocal, AnonymizeHost("localhost:8088"))
	})
	t.Run("external", func(t *testing.T) {
		assert.Equal(t, APIHostExternal, AnonymizeHost("apis.google.com"))
	})
}
