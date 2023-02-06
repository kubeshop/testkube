package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {

	t.Run("returns new client instance with default timeouts configured", func(t *testing.T) {
		// given / when
		c := NewClient(0)

		// then
		assert.Equal(t, DefaultClientTimeout, c.Timeout)
		assert.Equal(t, TLSHandshakeTimeout, c.Transport.(*http.Transport).TLSHandshakeTimeout)
	})

	t.Run("returns new client instance with custom timeout", func(t *testing.T) {
		timeout := 30 * time.Second
		c := NewClient(timeout)

		// then
		assert.Equal(t, timeout, c.Timeout)
		assert.Equal(t, TLSHandshakeTimeout, c.Transport.(*http.Transport).TLSHandshakeTimeout)
	})

	t.Run("returns new SSE client with a hour timeout", func(t *testing.T) {
		// given / when
		c := NewSSEClient()

		// then
		assert.Equal(t, time.Hour, c.Timeout)
	})
}
