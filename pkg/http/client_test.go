package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {

	t.Run("returns new client instance with configured timeouts correctly", func(t *testing.T) {
		// given / when
		c := NewClient()

		// then
		assert.Equal(t, ClientTimeout, c.Timeout)
		assert.Equal(t, TLSHandshakeTimeout, c.Transport.(*http.Transport).TLSHandshakeTimeout)
	})

	t.Run("returns new SSE client with a hour timeout", func(t *testing.T) {
		// given / when
		c := NewSSEClient()

		// then
		assert.Equal(t, time.Hour, c.Timeout)
	})
}
