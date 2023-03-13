package websocket

import (
	"testing"

	fastwebsocket "github.com/fasthttp/websocket"
	"github.com/gofiber/websocket/v2"
	"github.com/stretchr/testify/assert"
)

func TestLoader_Add(t *testing.T) {

	t.Run("adds connection to websockets pool", func(t *testing.T) {
		t.Skip("not implemented - TODO find a way to get real websocket connection")
		// given
		l := NewWebsocketLoader()
		ws := newTestWebsocket()

		// when
		l.Add(ws)
		l.Add(ws)

		// then
		assert.Equal(t, 2, len(l.Listener.Websockets))
	})

	t.Run("should remove websocket on connection close", func(t *testing.T) {
		t.Skip("not implemented - figure out how to call handle close")

		// given
		l := NewWebsocketLoader()
		ws := newTestWebsocket()
		l.Add(ws)
		assert.Equal(t, 1, len(l.Listener.Websockets))

		// when
		// on close is handled on frame with CloseMessage
		l.Listener.Websockets[0].Conn.Close()

		// then
		assert.Equal(t, 0, len(l.Listener.Websockets))
	})
}

// nolint
func newTestWebsocket() *websocket.Conn {
	return &websocket.Conn{Conn: &fastwebsocket.Conn{}}
}
